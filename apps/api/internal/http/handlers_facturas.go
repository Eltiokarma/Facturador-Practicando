package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	"github.com/eltiokarma/facturador/apps/api/internal/facturacion"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type FacturasHandler struct {
	Cfg          *config.Config
	Tenant       *tenant.Tenant // por ahora el tenant viene de env (single-tenant MVP)
	Cert         *cert.Material
	Motor        *motor.Cliente
	Comprobantes *comprobantes.Store
	Logger       *slog.Logger
}

type emitirRequest struct {
	Tipo          string                 `json:"tipo"`          // si viene vacío y la ruta es /facturas → "01"
	Serie         string                 `json:"serie"`
	FechaEmision  string                 `json:"fecha_emision"` // si viene vacío, hoy
	Moneda        string                 `json:"moneda"`
	TipoOperacion string                 `json:"tipo_operacion,omitempty"`
	Receptor      facturacion.Receptor   `json:"receptor"`
	Items         []facturacion.Item     `json:"items"`
}

type emitirRespuesta struct {
	ID            uuid.UUID            `json:"id"`
	Tipo          string               `json:"tipo"`
	Serie         string               `json:"serie"`
	Correlativo   int64                `json:"correlativo"`
	Estado        string               `json:"estado"`
	Codigo        string               `json:"codigo,omitempty"`
	Mensaje       string               `json:"mensaje,omitempty"`
	HashCPE       string               `json:"hash_cpe,omitempty"`
	Observaciones []string             `json:"observaciones,omitempty"`
	Totales       facturacion.Totales  `json:"totales"`
}

func (h *FacturasHandler) Emitir(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req emitirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido", "detalle": err.Error()})
		return
	}

	// Default tipo según ruta: facturas → "01", boletas → "03".
	if req.Tipo == "" {
		switch r.URL.Path {
		case "/api/v1/boletas":
			req.Tipo = "03"
		default:
			req.Tipo = "01"
		}
	}
	if req.Moneda == "" {
		req.Moneda = "PEN"
	}

	// Asignar correlativo atómico antes de validar el resto.
	correlativo, err := h.Comprobantes.NextCorrelativo(r.Context(), tenantID, req.Tipo, req.Serie)
	if err != nil {
		h.Logger.Error("NextCorrelativo", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "correlativo"})
		return
	}

	f := facturacion.Factura{
		Tipo: req.Tipo, Serie: req.Serie, Correlativo: correlativo,
		FechaEmision: req.FechaEmision, Moneda: req.Moneda,
		TipoOperacion: req.TipoOperacion, Receptor: req.Receptor, Items: req.Items,
	}
	if err := f.Validar(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validacion", "detalle": err.Error()})
		return
	}
	f.RecalcularTotales()

	mReq := motor.EmitirRequest{
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

	resp, err := h.Motor.Emitir(r.Context(), mReq)
	if err != nil {
		h.Logger.Error("motor.Emitir", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "motor_inalcanzable", "detalle": err.Error()})
		return
	}

	payloadJSON, _ := json.Marshal(f)
	saveIn := comprobantes.SaveInput{
		TenantID:        tenantID,
		Tipo:            f.Tipo,
		Serie:           f.Serie,
		Correlativo:     f.Correlativo,
		FechaEmision:    f.FechaEmision,
		Moneda:          f.Moneda,
		ReceptorTipoDoc: f.Receptor.TipoDoc,
		ReceptorNumDoc:  f.Receptor.NumDoc,
		ReceptorRazon:   f.Receptor.RazonSocial,
		Gravado:         f.Totales.Gravado,
		Exonerado:       f.Totales.Exonerado,
		Inafecto:        f.Totales.Inafecto,
		Gratuito:        f.Totales.Gratuito,
		IGV:             f.Totales.IGV,
		ISC:             f.Totales.ISC,
		ICBPER:          f.Totales.ICBPER,
		Total:           f.Totales.Total,
		Estado:          resp.Estado,
		SunatCodigo:     resp.Codigo,
		SunatMensaje:    resp.Mensaje,
		HashCPE:         resp.HashCPE,
		XMLFirmadoB64:   resp.XMLFirmado,
		CDRZipB64:       resp.CDRZip,
		Payload:         payloadJSON,
	}
	id, errSave := h.Comprobantes.Save(r.Context(), saveIn)
	if errSave != nil {
		h.Logger.Error("comprobantes.Save", "err", errSave)
	}

	status := http.StatusOK
	if resp.Estado == "rechazado" {
		status = http.StatusUnprocessableEntity
	} else if resp.Estado == "error" {
		status = http.StatusBadGateway
	}

	writeJSON(w, status, emitirRespuesta{
		ID: id, Tipo: f.Tipo, Serie: f.Serie, Correlativo: f.Correlativo,
		Estado: resp.Estado, Codigo: resp.Codigo, Mensaje: resp.Mensaje,
		HashCPE: resp.HashCPE, Observaciones: resp.Observaciones, Totales: f.Totales,
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
