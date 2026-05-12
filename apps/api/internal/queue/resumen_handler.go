package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/resumenes"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type ResumenHandler struct {
	Cfg       *config.Config
	Tenants   *tenant.Manager
	Certs     *cert.Manager
	Motor     *motor.Cliente
	Resumenes *resumenes.Store
	Client    *Client
	Logger    *slog.Logger
}

func (h *ResumenHandler) HandleEnviar(ctx context.Context, t *asynq.Task) error {
	var p ResumenPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("payload inválido: %w", err)
	}
	log := h.Logger.With("resumen_id", p.ResumenID, "tenant_id", p.TenantID)

	_ = h.Resumenes.MarcarEnviando(ctx, p.TenantID, p.ResumenID)

	r, err := h.Resumenes.Get(ctx, p.TenantID, p.ResumenID)
	if err != nil {
		return err
	}
	if r.Estado == "aceptado" || r.Estado == "aceptado_con_obs" || r.Estado == "rechazado" {
		log.Info("resumen ya finalizado, ignorando", "estado", r.Estado)
		return nil
	}

	ten, err := h.Tenants.Get(ctx, p.TenantID)
	if err != nil {
		_ = h.Resumenes.MarcarError(ctx, p.TenantID, p.ResumenID, "tenant no encontrado")
		return err
	}
	// Modo demo: simular aceptación inmediata sin tocar SUNAT.
	if ten.DemoMode {
		if err := h.Resumenes.AplicarResultado(ctx, p.TenantID, p.ResumenID, "aceptado", "0",
			"Modo DEMO — SUNAT no fue contactado.", ""); err != nil {
			return err
		}
		log.Info("resumen simulado en modo demo")
		return nil
	}

	mat, err := h.Certs.Get(p.TenantID)
	if err != nil {
		_ = h.Resumenes.MarcarError(ctx, p.TenantID, p.ResumenID, "certificado no disponible")
		return nil
	}

	boletas, err := h.Resumenes.Boletas(ctx, p.TenantID, p.ResumenID)
	if err != nil {
		return err
	}

	docs := make([]map[string]any, 0, len(boletas))
	for _, b := range boletas {
		docs = append(docs, map[string]any{
			"tipo_doc":          "03",
			"serie":             b.Serie,
			"correlativo":       b.Correlativo,
			"receptor_tipo_doc": "1",
			"receptor_num_doc":  b.ReceptorDoc,
			"moneda":            b.Moneda,
			"gravado":           b.Gravado,
			"exonerado":         b.Exonerado,
			"inafecto":          b.Inafecto,
			"igv":               b.IGV,
			"total":             b.Total,
		})
	}

	modo := ten.SunatMode
	if modo == "" {
		modo = string(h.Cfg.SunatMode)
	}
	req := motor.ResumenRequest{
		Modo:   modo,
		Tenant: tenantPayload(ten, mat),
		Resumen: map[string]any{
			"serie":            r.Serie,
			"correlativo":      r.Correlativo,
			"fecha_referencia": r.FechaReferencia,
			"fecha_emision":    r.FechaEmision,
			"documentos":       docs,
		},
	}
	resp, err := h.Motor.Resumen(ctx, req)
	if err != nil {
		log.Warn("motor.Resumen falló, reintentando", "err", err)
		_ = h.Resumenes.MarcarError(ctx, p.TenantID, p.ResumenID, err.Error())
		return err
	}
	if resp.Estado == "error" {
		_ = h.Resumenes.AplicarResultado(ctx, p.TenantID, p.ResumenID, "rechazado", resp.Codigo, resp.Mensaje, "")
		return nil
	}
	if resp.Ticket == "" {
		_ = h.Resumenes.MarcarError(ctx, p.TenantID, p.ResumenID, "SUNAT no devolvió ticket")
		return fmt.Errorf("sin ticket")
	}
	if err := h.Resumenes.MarcarConsultando(ctx, p.TenantID, p.ResumenID, resp.Ticket, resp.XMLFirmado); err != nil {
		return err
	}
	log.Info("resumen enviado, ticket recibido", "ticket", resp.Ticket)
	return h.Client.EnqueueResumenStatus(ctx, p, 30*time.Second)
}

func (h *ResumenHandler) HandleStatus(ctx context.Context, t *asynq.Task) error {
	var p ResumenPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log := h.Logger.With("resumen_id", p.ResumenID, "tenant_id", p.TenantID)

	r, err := h.Resumenes.Get(ctx, p.TenantID, p.ResumenID)
	if err != nil {
		return err
	}
	if r.Estado != "consultando" {
		return nil
	}
	if r.Ticket == "" {
		return fmt.Errorf("resumen sin ticket pero en estado consultando")
	}

	ten, err := h.Tenants.Get(ctx, p.TenantID)
	if err != nil {
		return err
	}
	mat, err := h.Certs.Get(p.TenantID)
	if err != nil {
		return err
	}

	modo := ten.SunatMode
	if modo == "" {
		modo = string(h.Cfg.SunatMode)
	}
	resp, err := h.Motor.ConsultaTicket(ctx, motor.ConsultaTicketRequest{
		Modo:   modo,
		Tenant: tenantPayload(ten, mat),
		Ticket: r.Ticket,
	})
	if err != nil {
		log.Warn("motor.ConsultaTicket falló", "err", err)
		return err
	}

	if resp.Estado == "procesando" {
		return h.Client.EnqueueResumenStatus(ctx, p, 60*time.Second)
	}

	estadoFinal := resp.Estado
	if estadoFinal == "" {
		estadoFinal = "error"
	}
	if err := h.Resumenes.AplicarResultado(ctx, p.TenantID, p.ResumenID, estadoFinal, resp.Codigo, resp.Mensaje, resp.CDRZip); err != nil {
		return err
	}
	log.Info("resumen finalizado", "estado", estadoFinal)
	return nil
}

func tenantPayload(t *tenant.Record, c *cert.Material) map[string]any {
	return map[string]any{
		"ruc":              t.RUC,
		"razon_social":     t.RazonSocial,
		"nombre_comercial": t.NombreComercial,
		"direccion_fiscal": t.DireccionFiscal,
		"ubigeo":           t.Ubigeo,
		"departamento":     t.Departamento,
		"provincia":        t.Provincia,
		"distrito":         t.Distrito,
		"usuario_sol":      t.UsuarioSOL,
		"clave_sol":        t.ClaveSOL,
		"cert_pem":         c.CertPEM,
		"cert_key_pem":     c.KeyPEM,
	}
}
