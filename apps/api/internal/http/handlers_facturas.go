package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	"github.com/eltiokarma/facturador/apps/api/internal/facturacion"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/storage"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type FacturasHandler struct {
	Cfg     *config.Config
	Tenant  *tenant.Tenant
	Cert    *cert.Material
	Motor   *motor.Cliente
	Store   *storage.Store
	Logger  *slog.Logger
}

type emitirRespuesta struct {
	Estado        string                 `json:"estado"`
	Codigo        string                 `json:"codigo,omitempty"`
	Mensaje       string                 `json:"mensaje,omitempty"`
	HashCPE       string                 `json:"hash_cpe,omitempty"`
	Observaciones []string               `json:"observaciones,omitempty"`
	Totales       facturacion.Totales    `json:"totales"`
	GuardadoEn    string                 `json:"guardado_en,omitempty"`
}

func (h *FacturasHandler) Emitir(w http.ResponseWriter, r *http.Request) {
	var f facturacion.Factura
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido", "detalle": err.Error()})
		return
	}
	if err := f.Validar(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validacion", "detalle": err.Error()})
		return
	}

	// Recalcular totales server-side. Lo que mandó el cliente es referencia, no verdad.
	f.RecalcularTotales()

	req := motor.EmitirRequest{
		Modo: string(h.Cfg.SunatMode),
		Tenant: map[string]any{
			"ruc":              h.Tenant.RUC,
			"razon_social":     h.Tenant.RazonSocial,
			"nombre_comercial": h.Tenant.NombreComercial,
			"direccion_fiscal": h.Tenant.DireccionFiscal,
			"ubigeo":           h.Tenant.Ubigeo,
			"departamento":     h.Tenant.Departamento,
			"provincia":        h.Tenant.Provincia,
			"distrito":         h.Tenant.Distrito,
			"usuario_sol":      h.Tenant.UsuarioSOL,
			"clave_sol":        h.Tenant.ClaveSOL,
			"cert_pem":         h.Cert.CertPEM,
			"cert_key_pem":     h.Cert.KeyPEM,
		},
		Comprobante: map[string]any{
			"tipo":           f.Tipo,
			"serie":          f.Serie,
			"correlativo":    f.Correlativo,
			"fecha_emision":  f.FechaEmision,
			"moneda":         f.Moneda,
			"tipo_operacion": defaultStr(f.TipoOperacion, "0101"),
			"receptor": map[string]any{
				"tipo_doc":     f.Receptor.TipoDoc,
				"num_doc":      f.Receptor.NumDoc,
				"razon_social": f.Receptor.RazonSocial,
				"direccion":    f.Receptor.Direccion,
			},
			"items":   itemsToMap(f.Items),
			"totales": totalesToMap(f.Totales),
		},
	}

	resp, err := h.Motor.Emitir(r.Context(), req)
	if err != nil {
		h.Logger.Error("motor falló", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "motor_inalcanzable", "detalle": err.Error()})
		return
	}

	payloadJSON, _ := json.Marshal(f)
	reg := storage.Registro{
		Tipo:         f.Tipo,
		Serie:        f.Serie,
		Correlativo:  f.Correlativo,
		FechaEmision: f.FechaEmision,
		Estado:       resp.Estado,
		Codigo:       resp.Codigo,
		Mensaje:      resp.Mensaje,
		HashCPE:      resp.HashCPE,
		Payload:      payloadJSON,
	}
	dir, errStore := h.Store.Guardar(reg, resp.XMLFirmado, resp.CDRZip)
	if errStore != nil {
		h.Logger.Error("guardar artefactos falló", "err", errStore)
	}

	status := http.StatusOK
	if resp.Estado == "rechazado" {
		status = http.StatusUnprocessableEntity
	} else if resp.Estado == "error" {
		status = http.StatusBadGateway
	}

	writeJSON(w, status, emitirRespuesta{
		Estado:        resp.Estado,
		Codigo:        resp.Codigo,
		Mensaje:       resp.Mensaje,
		HashCPE:       resp.HashCPE,
		Observaciones: resp.Observaciones,
		Totales:       f.Totales,
		GuardadoEn:    dir,
	})
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func itemsToMap(items []facturacion.Item) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"codigo":          it.Codigo,
			"descripcion":     it.Descripcion,
			"unidad":          it.Unidad,
			"cantidad":        it.Cantidad,
			"valor_unitario":  it.ValorUnitario,
			"precio_unitario": it.PrecioUnitario,
			"afectacion_igv":  it.AfectacionIGV,
			"porcentaje_igv":  it.PorcentajeIGV,
			"igv":             it.IGV,
			"total":           it.Total,
		})
	}
	return out
}

func totalesToMap(t facturacion.Totales) map[string]any {
	return map[string]any{
		"gravado":   t.Gravado,
		"exonerado": t.Exonerado,
		"inafecto":  t.Inafecto,
		"gratuito":  t.Gratuito,
		"igv":       t.IGV,
		"isc":       t.ISC,
		"icbper":    t.ICBPER,
		"total":     t.Total,
	}
}
