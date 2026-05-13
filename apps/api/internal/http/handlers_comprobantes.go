package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/pdf"
	"github.com/eltiokarma/facturador/apps/api/internal/queue"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type ComprobantesHandler struct {
	Store      *comprobantes.Store
	PDFBuilder *pdf.Builder
	Tenants    *tenant.Manager
	Queue      *queue.Client
	Logger     *slog.Logger
}

func (h *ComprobantesHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	out, err := h.Store.List(r.Context(), comprobantes.ListFilter{
		TenantID: tenantID,
		Estado:   q.Get("estado"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		h.Logger.Error("comprobantes.List", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *ComprobantesHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	d, err := h.Store.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, comprobantes.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *ComprobantesHandler) XML(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	data, err := h.Store.XML(r.Context(), tenantID, id)
	if err != nil || len(data) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_disponible"})
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(data)
}

func (h *ComprobantesHandler) PDF(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	d, err := h.Store.Get(r.Context(), tenantID, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
		return
	}
	ten, err := h.Tenants.Get(r.Context(), tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "tenant_no_disponible"})
		return
	}
	data, err := h.PDFBuilder.Render(ten, d)
	if err != nil {
		h.Logger.Error("pdf.Render", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pdf_falló"})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`inline; filename="%s-%d.pdf"`, d.Serie, d.Correlativo))
	_, _ = w.Write(data)
}

// Reintentar re-encola un comprobante que quedó en estado 'error' o
// 'pendiente'. Útil cuando SUNAT volvió y queremos acelerar el siguiente
// intento sin esperar el backoff de asynq.
func (h *ComprobantesHandler) Reintentar(w http.ResponseWriter, r *http.Request) {
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
	d, err := h.Store.Get(r.Context(), tenantID, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
		return
	}
	switch d.Estado {
	case "error", "pendiente":
		// ok, reencolable
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "estado_no_reintenta",
			"detalle": "solo se pueden reintentar comprobantes en estado 'error' o 'pendiente' (actual: " + d.Estado + ")",
		})
		return
	}
	if err := h.Queue.EnqueueEmit(r.Context(), queue.EmitPayload{
		TenantID: tenantID, ComprobanteID: id,
	}); err != nil {
		h.Logger.Error("Reintentar EnqueueEmit", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encolar"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"ok": "encolado"})
}

func (h *ComprobantesHandler) CDR(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	data, err := h.Store.CDR(r.Context(), tenantID, id)
	if err != nil || len(data) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_disponible"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="cdr.zip"`)
	_, _ = w.Write(data)
}
