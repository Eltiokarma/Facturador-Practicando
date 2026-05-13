package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

// handleGRE procesa un job de emisión cuando el comprobante es GRE (tipo 09).
// Llama al endpoint /emitir-guia del motor PHP. Si SUNAT devuelve ticket,
// el motor PHP ya consultó el ticket internamente (Greenter lo hace en un
// solo flow) — el resultado final viene en la misma response. Si no, deja
// el comprobante en estado consultando con el ticket guardado y reencola
// la consulta más tarde.
//
// Para el MVP, asumimos que el motor PHP resuelve el ticket en la misma
// llamada (con un poll interno breve). Si SUNAT tarda demasiado, el motor
// devuelve "ticket pendiente" y nosotros reencolamos.
func (h *EmitHandler) handleGRE(
	ctx context.Context,
	p EmitPayload,
	d *comprobantes.Detalle,
	ten *tenant.Record,
	log *slog.Logger,
) error {
	// La GRE no usa el cert .p12 con Clave SOL, usa OAuth2 con
	// credenciales API SUNAT (cliente_id + client_secret). Por ahora
	// el cert.Manager NO se usa para GRE.
	if len(ten.CertPassCipher) == 0 && (ten.SunatMode == "prod" || ten.SunatMode == "beta") {
		// Heurística: si no hay credenciales API GRE configuradas en el
		// tenant, marcamos error con mensaje claro.
		// En la v1 ese check se hace por columnas dedicadas
		// (gre_client_id_cifrado / gre_client_secret_cifrado).
		// Por ahora lo dejamos pasar — el motor PHP devolverá error.
	}

	var g map[string]any
	if err := json.Unmarshal(d.Payload, &g); err != nil {
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, "payload GRE corrupto")
		return nil
	}
	// Asegurar que correlativo y serie del payload reflejen lo persistido.
	g["correlativo"] = d.Correlativo
	g["serie"] = d.Serie

	modo := ten.SunatMode
	if modo == "" {
		modo = string(h.Cfg.SunatMode)
	}
	req := motor.GuiaRequest{
		Modo: modo,
		Tenant: map[string]any{
			"ruc":              ten.RUC,
			"razon_social":     ten.RazonSocial,
			"nombre_comercial": ten.NombreComercial,
			"direccion_fiscal": ten.DireccionFiscal,
			"ubigeo":           ten.Ubigeo,
			// Credenciales API GRE (vacías si no están configuradas):
			"gre_client_id":     "", // TODO: leer columnas cifradas
			"gre_client_secret": "",
		},
		Guia: g,
	}
	resp, err := h.Motor.Guia(ctx, req)
	if err != nil {
		log.Warn("motor.Guia falló, asynq reintentará", "err", err)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, err.Error())
		return err
	}
	if resp.Estado == "error" || resp.Estado == "rechazado" {
		_ = h.Comprobantes.AplicarResultado(ctx, p.TenantID, p.ComprobanteID, comprobantes.Resultado{
			Estado: "rechazado", Codigo: resp.Codigo, Mensaje: resp.Mensaje,
		}, d.Tipo, d.Serie, d.Correlativo)
		return nil
	}
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

func firstNonEmpty(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

// Silenciar unused import en algunos escenarios
var _ = fmt.Sprintf
