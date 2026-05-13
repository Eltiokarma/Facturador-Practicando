package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/eltiokarma/facturador/apps/api/internal/anulaciones"
	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type AnulHandler struct {
	Cfg          *config.Config
	Tenants      *tenant.Manager
	Certs        *cert.Manager
	Motor        *motor.Cliente
	Anulaciones  *anulaciones.Store
	Comprobantes *comprobantes.Store
	Client       *Client
	Logger       *slog.Logger
}

func (h *AnulHandler) HandleEnviar(ctx context.Context, t *asynq.Task) error {
	var p AnularPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("payload inválido: %w", err)
	}
	log := h.Logger.With("anulacion_id", p.AnulacionID, "tenant_id", p.TenantID)

	_ = h.Anulaciones.MarcarEnviando(ctx, p.TenantID, p.AnulacionID)

	a, err := h.Anulaciones.Get(ctx, p.TenantID, p.AnulacionID)
	if err != nil {
		return err
	}
	if a.Estado == "aceptado" || a.Estado == "aceptado_con_obs" || a.Estado == "rechazado" {
		return nil
	}

	ten, err := h.Tenants.Get(ctx, p.TenantID)
	if err != nil {
		return err
	}

	// Modo DEMO: aceptar sin tocar SUNAT y marcar comprobantes como anulados.
	if ten.DemoMode {
		if err := h.Anulaciones.AplicarResultado(ctx, p.TenantID, p.AnulacionID,
			"aceptado", "0", "Modo DEMO — SUNAT no fue contactado.", ""); err != nil {
			return err
		}
		log.Info("anulación simulada en modo DEMO")
		return h.marcarComprobantesAnulados(ctx, p.TenantID, p.AnulacionID, log)
	}

	mat, err := h.Certs.Get(p.TenantID)
	if err != nil {
		_ = h.Anulaciones.MarcarError(ctx, p.TenantID, p.AnulacionID, "certificado no disponible")
		return nil
	}

	items, err := h.Anulaciones.Items(ctx, p.TenantID, p.AnulacionID)
	if err != nil {
		return err
	}
	docs := make([]map[string]any, 0, len(items))
	for _, it := range items {
		docs = append(docs, map[string]any{
			"tipo_doc":    it.Tipo,
			"serie":       it.Serie,
			"correlativo": it.Correlativo,
			"motivo":      it.Motivo,
		})
	}

	modo := ten.SunatMode
	if modo == "" {
		modo = string(h.Cfg.SunatMode)
	}
	req := motor.AnularRequest{
		Modo:   modo,
		Tenant: tenantPayload(ten, mat),
		Anulacion: map[string]any{
			"serie":            a.Serie,
			"correlativo":      a.Correlativo,
			"fecha_referencia": a.FechaReferencia,
			"fecha_emision":    a.FechaEmision,
			"documentos":       docs,
		},
	}
	resp, err := h.Motor.Anular(ctx, req)
	if err != nil {
		_ = h.Anulaciones.MarcarError(ctx, p.TenantID, p.AnulacionID, err.Error())
		return err
	}
	if resp.Estado == "error" {
		_ = h.Anulaciones.AplicarResultado(ctx, p.TenantID, p.AnulacionID,
			"rechazado", resp.Codigo, resp.Mensaje, "")
		return nil
	}
	if resp.Ticket == "" {
		_ = h.Anulaciones.MarcarError(ctx, p.TenantID, p.AnulacionID, "SUNAT no devolvió ticket")
		return fmt.Errorf("sin ticket")
	}
	if err := h.Anulaciones.MarcarConsultando(ctx, p.TenantID, p.AnulacionID, resp.Ticket, resp.XMLFirmado); err != nil {
		return err
	}
	log.Info("anulación enviada, ticket recibido", "ticket", resp.Ticket)
	return h.Client.EnqueueAnularStatus(ctx, p, 30*time.Second)
}

func (h *AnulHandler) HandleStatus(ctx context.Context, t *asynq.Task) error {
	var p AnularPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log := h.Logger.With("anulacion_id", p.AnulacionID, "tenant_id", p.TenantID)

	a, err := h.Anulaciones.Get(ctx, p.TenantID, p.AnulacionID)
	if err != nil {
		return err
	}
	if a.Estado != "consultando" {
		return nil
	}
	if a.Ticket == "" {
		return fmt.Errorf("anulación en consultando sin ticket")
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
		Ticket: a.Ticket,
	})
	if err != nil {
		return err
	}
	if resp.Estado == "procesando" {
		return h.Client.EnqueueAnularStatus(ctx, p, 60*time.Second)
	}
	estado := resp.Estado
	if estado == "" {
		estado = "error"
	}
	if err := h.Anulaciones.AplicarResultado(ctx, p.TenantID, p.AnulacionID, estado, resp.Codigo, resp.Mensaje, resp.CDRZip); err != nil {
		return err
	}
	if estado == "aceptado" || estado == "aceptado_con_obs" {
		if err := h.marcarComprobantesAnulados(ctx, p.TenantID, p.AnulacionID, log); err != nil {
			log.Error("marcarComprobantesAnulados", "err", err)
		}
	}
	log.Info("anulación finalizada", "estado", estado)
	return nil
}

func (h *AnulHandler) marcarComprobantesAnulados(ctx context.Context, tenantID, anulacionID uuid.UUID, log *slog.Logger) error {
	ids, err := h.Anulaciones.ItemIDs(ctx, anulacionID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := h.Comprobantes.MarcarAnulado(ctx, tenantID, id); err != nil {
			log.Warn("MarcarAnulado", "comprobante_id", id, "err", err)
		}
	}
	return nil
}
