package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/guia"
	"github.com/eltiokarma/facturador/apps/api/internal/queue"
)

type GuiasHandler struct {
	Comprobantes *comprobantes.Store
	Queue        *queue.Client
	Logger       *slog.Logger
}

type emitirGuiaResp struct {
	ID          uuid.UUID `json:"id"`
	Tipo        string    `json:"tipo"`
	Serie       string    `json:"serie"`
	Correlativo int64     `json:"correlativo"`
	Estado      string    `json:"estado"`
}

// Emitir valida la guía, asigna correlativo atómico, crea pendiente y encola.
func (h *GuiasHandler) Emitir(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var g guia.Guia
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido", "detalle": err.Error()})
		return
	}
	if g.Serie == "" {
		g.Serie = "T001"
	}

	// Correlativo atómico, mismo flow que facturas pero con tipo=09.
	correlativo, err := h.Comprobantes.NextCorrelativo(r.Context(), tenantID, "09", g.Serie)
	if err != nil {
		h.Logger.Error("NextCorrelativo GRE", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "correlativo"})
		return
	}
	g.Correlativo = correlativo

	if err := g.Validar(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validacion", "detalle": err.Error()})
		return
	}

	payloadJSON, _ := json.Marshal(g)
	id, err := h.Comprobantes.CreatePendiente(r.Context(), comprobantes.PendienteInput{
		TenantID:        tenantID,
		Tipo:            "09",
		Serie:           g.Serie,
		Correlativo:     correlativo,
		FechaEmision:    g.FechaEmision,
		Moneda:          "PEN", // la GRE no maneja moneda real, pero la tabla lo exige
		ReceptorTipoDoc: g.Destinatario.TipoDoc,
		ReceptorNumDoc:  g.Destinatario.NumDoc,
		ReceptorRazon:   g.Destinatario.RazonSocial,
		Payload:         payloadJSON,
	})
	if err != nil {
		h.Logger.Error("CreatePendiente GRE", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persistir"})
		return
	}
	if err := h.Queue.EnqueueEmit(r.Context(), queue.EmitPayload{
		TenantID: tenantID, ComprobanteID: id,
	}); err != nil {
		h.Logger.Error("EnqueueEmit GRE", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encolar"})
		return
	}
	writeJSON(w, http.StatusAccepted, emitirGuiaResp{
		ID: id, Tipo: "09", Serie: g.Serie, Correlativo: correlativo, Estado: "pendiente",
	})
}
