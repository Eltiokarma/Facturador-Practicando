package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/anulaciones"
	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/queue"
)

type AnulacionesHandler struct {
	Store  *anulaciones.Store
	Queue  *queue.Client
	Logger *slog.Logger
}

type anularReq struct {
	Motivo string `json:"motivo"`
}

// POST /api/v1/comprobantes/{id}/anular
func (h *AnulacionesHandler) AnularComprobante(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	var req anularReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	a, err := h.Store.CrearIndividual(r.Context(), tenantID, id, req.Motivo)
	if err != nil {
		switch {
		case errors.Is(err, anulaciones.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
		case errors.Is(err, anulaciones.ErrFueraPlazo),
			errors.Is(err, anulaciones.ErrYaAnulado),
			errors.Is(err, anulaciones.ErrNoAceptado):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": "no_se_puede_anular", "detalle": err.Error(),
			})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "crear", "detalle": err.Error()})
		}
		return
	}
	if err := h.Queue.EnqueueAnularEnviar(r.Context(), queue.AnularPayload{
		TenantID: tenantID, AnulacionID: a.ID,
	}); err != nil {
		h.Logger.Error("EnqueueAnularEnviar", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encolar"})
		return
	}
	writeJSON(w, http.StatusAccepted, a)
}

func (h *AnulacionesHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	items, err := h.Store.List(r.Context(), tenantID, 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type detalleAnulacion struct {
	*anulaciones.Anulacion
	Comprobantes []anulaciones.ComprobanteAnular `json:"comprobantes"`
}

func (h *AnulacionesHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	a, err := h.Store.Get(r.Context(), tenantID, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
		return
	}
	items, _ := h.Store.Items(r.Context(), tenantID, id)
	writeJSON(w, http.StatusOK, detalleAnulacion{Anulacion: a, Comprobantes: items})
}
