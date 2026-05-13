package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

// handleGRE procesa un job de emisión cuando el comprobante es GRE (tipo 09).
// Llama al motor /emitir-guia. El motor hace el flow OAuth2 + REST y devuelve:
//   - estado="ticket" + ticket → SUNAT está procesando; encolamos consulta.
//   - estado="aceptado"/"aceptado_con_obs"/"rechazado" → resultado final.
//   - estado="error" → reintentable.
func (h *EmitHandler) handleGRE(
	ctx context.Context,
	p EmitPayload,
	d *comprobantes.Detalle,
	ten *tenant.Record,
	log *slog.Logger,
) error {
	mat, certErr := h.Certs.Get(p.TenantID)
	if certErr != nil {
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID,
			"GRE requiere certificado .p12 — subilo desde Configuración")
		return nil
	}
	if ten.GREClientID == "" || ten.GREClientSecret == "" {
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID,
			"GRE requiere Client ID + Client Secret de la API SUNAT — configuralos en 'Credenciales API GRE'")
		return nil
	}

	var g map[string]any
	if err := json.Unmarshal(d.Payload, &g); err != nil {
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, "payload GRE corrupto")
		return nil
	}
	g["correlativo"] = d.Correlativo
	g["serie"] = d.Serie

	modo := ten.SunatMode
	if modo == "" {
		modo = string(h.Cfg.SunatMode)
	}
	req := motor.GuiaRequest{
		Modo: modo,
		Tenant: map[string]any{
			"ruc":               ten.RUC,
			"razon_social":      ten.RazonSocial,
			"nombre_comercial":  ten.NombreComercial,
			"direccion_fiscal":  ten.DireccionFiscal,
			"ubigeo":            ten.Ubigeo,
			"cert_pem":          mat.CertPEM,
			"cert_key_pem":      mat.KeyPEM,
			"gre_client_id":     ten.GREClientID,
			"gre_client_secret": ten.GREClientSecret,
		},
		Guia: g,
	}
	resp, err := h.Motor.Guia(ctx, req)
	if err != nil {
		log.Warn("motor.Guia falló, asynq reintentará", "err", err)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, err.Error())
		return err
	}

	switch resp.Estado {
	case "ticket":
		// SUNAT recibió y devolvió ticket; consultamos en 10s.
		if resp.Ticket == "" {
			_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID,
				"motor no devolvió ticket de GRE")
			return fmt.Errorf("sin ticket")
		}
		if err := h.Comprobantes.MarcarConsultando(ctx, p.TenantID, p.ComprobanteID, resp.Ticket); err != nil {
			return err
		}
		log.Info("GRE enviada, ticket recibido", "ticket", resp.Ticket)
		return h.Client.EnqueueGuiaStatus(ctx, p, 10*time.Second)

	case "error":
		log.Warn("motor.Guia devolvió error transitorio", "codigo", resp.Codigo, "msg", resp.Mensaje)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, resp.Codigo+" "+resp.Mensaje)
		return fmt.Errorf("error transitorio: %s", resp.Mensaje)

	case "rechazado":
		_ = h.Comprobantes.AplicarResultado(ctx, p.TenantID, p.ComprobanteID, comprobantes.Resultado{
			Estado: "rechazado", Codigo: resp.Codigo, Mensaje: resp.Mensaje,
			XMLFirmadoB64: resp.XMLFirmado,
		}, d.Tipo, d.Serie, d.Correlativo)
		return nil

	default: // aceptado / aceptado_con_obs
		res := comprobantes.Resultado{
			Estado:        firstNonEmpty(resp.Estado, "aceptado"),
			Codigo:        resp.Codigo,
			Mensaje:       resp.Mensaje,
			HashCPE:       resp.HashCPE,
			XMLFirmadoB64: resp.XMLFirmado,
			CDRZipB64:     resp.CDRZip,
		}
		if err := h.Comprobantes.AplicarResultado(ctx, p.TenantID, p.ComprobanteID, res, d.Tipo, d.Serie, d.Correlativo); err != nil {
			return err
		}
		log.Info("GRE procesada", "estado", res.Estado, "codigo", resp.Codigo)
		return nil
	}
}

// HandleGuiaStatus consulta el ticket de una GRE pendiente.
func (h *EmitHandler) HandleGuiaStatus(ctx context.Context, t *asynq.Task) error {
	var p EmitPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log := h.Logger.With("comprobante_id", p.ComprobanteID, "tenant_id", p.TenantID)

	d, err := h.Comprobantes.Get(ctx, p.TenantID, p.ComprobanteID)
	if err != nil {
		return err
	}
	if d.Estado != "consultando" {
		return nil
	}
	if d.SunatTicket == "" {
		return fmt.Errorf("GRE en consultando pero sin ticket")
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
	resp, err := h.Motor.ConsultaTicketGuia(ctx, motor.ConsultaTicketGuiaRequest{
		Modo: modo,
		Tenant: map[string]any{
			"ruc":               ten.RUC,
			"cert_pem":          mat.CertPEM,
			"cert_key_pem":      mat.KeyPEM,
			"gre_client_id":     ten.GREClientID,
			"gre_client_secret": ten.GREClientSecret,
		},
		Ticket: d.SunatTicket,
	})
	if err != nil {
		log.Warn("ConsultaTicketGuia falló", "err", err)
		return err
	}

	if resp.Estado == "procesando" {
		log.Info("ticket GRE aún procesándose, reintentando en 30s")
		return h.Client.EnqueueGuiaStatus(ctx, p, 30*time.Second)
	}

	estado := resp.Estado
	if estado == "" {
		estado = "error"
	}
	if err := h.Comprobantes.AplicarResultado(ctx, p.TenantID, p.ComprobanteID, comprobantes.Resultado{
		Estado:    estado,
		Codigo:    resp.Codigo,
		Mensaje:   resp.Mensaje,
		HashCPE:   resp.HashCPE,
		CDRZipB64: resp.CDRZip,
	}, d.Tipo, d.Serie, d.Correlativo); err != nil {
		return err
	}
	log.Info("GRE finalizada", "estado", estado)
	return nil
}

func firstNonEmpty(a, b string) string {
	if a == "" {
		return b
	}
	return a
}
