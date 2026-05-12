package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/facturacion"
	"github.com/eltiokarma/facturador/apps/api/internal/queue"
)

type FacturasHandler struct {
	Comprobantes *comprobantes.Store
	Queue        *queue.Client
	Logger       *slog.Logger
}

type emitirRequest struct {
	Tipo          string               `json:"tipo"`
	Serie         string               `json:"serie"`
	FechaEmision  string               `json:"fecha_emision"`
	Moneda        string               `json:"moneda"`
	TipoOperacion string               `json:"tipo_operacion,omitempty"`
	Receptor      facturacion.Receptor `json:"receptor"`
	Items         []facturacion.Item   `json:"items"`
}

type emitirRespuesta struct {
	ID          uuid.UUID           `json:"id"`
	Tipo        string              `json:"tipo"`
	Serie       string              `json:"serie"`
	Correlativo int64               `json:"correlativo"`
	Estado      string              `json:"estado"`
	Totales     facturacion.Totales `json:"totales"`
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

	payloadDoc := map[string]any{
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
	}
	payloadJSON, _ := json.Marshal(payloadDoc)

	id, err := h.Comprobantes.CreatePendiente(r.Context(), comprobantes.PendienteInput{
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
		Payload:         payloadJSON,
	})
	if err != nil {
		h.Logger.Error("CreatePendiente", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persistir"})
		return
	}

	if err := h.Queue.EnqueueEmit(r.Context(), queue.EmitPayload{
		TenantID: tenantID, ComprobanteID: id,
	}); err != nil {
		h.Logger.Error("EnqueueEmit", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encolar"})
		return
	}

	writeJSON(w, http.StatusAccepted, emitirRespuesta{
		ID: id, Tipo: f.Tipo, Serie: f.Serie, Correlativo: f.Correlativo,
		Estado: "pendiente", Totales: f.Totales,
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
