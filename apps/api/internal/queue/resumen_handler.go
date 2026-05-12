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
	Tenant    *tenant.Tenant
	Cert      *cert.Material
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
	log := h.Logger.With("resumen_id", p.ResumenID)

	_ = h.Resumenes.MarcarEnviando(ctx, p.TenantID, p.ResumenID)

	r, err := h.Resumenes.Get(ctx, p.TenantID, p.ResumenID)
	if err != nil {
		return err
	}
	if r.Estado == "aceptado" || r.Estado == "aceptado_con_obs" || r.Estado == "rechazado" {
		log.Info("resumen ya finalizado, ignorando", "estado", r.Estado)
		return nil
	}

	boletas, err := h.Resumenes.Boletas(ctx, p.TenantID, p.ResumenID)
	if err != nil {
		return err
	}

	docs := make([]map[string]any, 0, len(boletas))
	for _, b := range boletas {
		docs = append(docs, map[string]any{
			"tipo_doc":     "03",
			"serie":        b.Serie,
			"correlativo":  b.Correlativo,
			"receptor_tipo_doc": "1", // mayormente DNI en boletas; el motor decide según el doc
			"receptor_num_doc":  b.ReceptorDoc,
			"moneda":       b.Moneda,
			"gravado":      b.Gravado,
			"exonerado":    b.Exonerado,
			"inafecto":     b.Inafecto,
			"igv":          b.IGV,
			"total":        b.Total,
		})
	}

	req := motor.ResumenRequest{
		Modo:   string(h.Cfg.SunatMode),
		Tenant: tenantPayload(h.Tenant, h.Cert),
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

	// Encolar consulta del ticket en 30s
	return h.Client.EnqueueResumenStatus(ctx, p, 30*time.Second)
}

func (h *ResumenHandler) HandleStatus(ctx context.Context, t *asynq.Task) error {
	var p ResumenPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log := h.Logger.With("resumen_id", p.ResumenID)

	r, err := h.Resumenes.Get(ctx, p.TenantID, p.ResumenID)
	if err != nil {
		return err
	}
	if r.Estado != "consultando" {
		log.Info("resumen ya no está consultando, ignorando", "estado", r.Estado)
		return nil
	}
	if r.Ticket == "" {
		return fmt.Errorf("resumen sin ticket pero en estado consultando")
	}

	resp, err := h.Motor.ConsultaTicket(ctx, motor.ConsultaTicketRequest{
		Modo:   string(h.Cfg.SunatMode),
		Tenant: tenantPayload(h.Tenant, h.Cert),
		Ticket: r.Ticket,
	})
	if err != nil {
		log.Warn("motor.ConsultaTicket falló, reintentando", "err", err)
		return err
	}

	if resp.Estado == "procesando" {
		// Reintentar en 60s
		log.Info("ticket aún procesándose, reintentando en 60s")
		return h.Client.EnqueueResumenStatus(ctx, p, 60*time.Second)
	}

	estadoFinal := resp.Estado
	if estadoFinal == "" {
		estadoFinal = "error"
	}
	if err := h.Resumenes.AplicarResultado(ctx, p.TenantID, p.ResumenID, estadoFinal, resp.Codigo, resp.Mensaje, resp.CDRZip); err != nil {
		return err
	}
	log.Info("resumen finalizado", "estado", estadoFinal, "codigo", resp.Codigo)
	return nil
}

func tenantPayload(t *tenant.Tenant, c *cert.Material) map[string]any {
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
