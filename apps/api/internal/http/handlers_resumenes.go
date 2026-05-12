package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/queue"
	"github.com/eltiokarma/facturador/apps/api/internal/resumenes"
)

type ResumenesHandler struct {
	Store  *resumenes.Store
	Queue  *queue.Client
	Logger *slog.Logger
}

// GET /api/v1/resumenes/pendientes?fecha=YYYY-MM-DD
func (h *ResumenesHandler) Pendientes(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	fechaStr := r.URL.Query().Get("fecha")
	if fechaStr == "" {
		fechaStr = time.Now().Format("2006-01-02")
	}
	fecha, err := time.Parse("2006-01-02", fechaStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fecha_invalida"})
		return
	}
	items, err := h.Store.PendientesPorFecha(r.Context(), tenantID, fecha)
	if err != nil {
		h.Logger.Error("resumenes.Pendientes", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fecha": fechaStr, "items": items})
}

type crearReq struct {
	Fecha    string      `json:"fecha"` // YYYY-MM-DD
	Boletas  []uuid.UUID `json:"boletas,omitempty"` // si vacío, todas las pendientes del día
}

func (h *ResumenesHandler) Crear(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	var req crearReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	fecha, err := time.Parse("2006-01-02", req.Fecha)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fecha_invalida"})
		return
	}
	// Si no se especificaron boletas, tomar todas las pendientes del día.
	ids := req.Boletas
	if len(ids) == 0 {
		pendientes, err := h.Store.PendientesPorFecha(r.Context(), tenantID, fecha)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		ids = make([]uuid.UUID, 0, len(pendientes))
		for _, b := range pendientes {
			ids = append(ids, b.ID)
		}
	}
	if len(ids) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sin_boletas", "detalle": "no hay boletas pendientes para esa fecha"})
		return
	}

	resumen, err := h.Store.Crear(r.Context(), tenantID, fecha, ids)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "crear", "detalle": err.Error()})
		return
	}
	if err := h.Queue.EnqueueResumenEnviar(r.Context(), queue.ResumenPayload{
		TenantID: tenantID, ResumenID: resumen.ID,
	}); err != nil {
		h.Logger.Error("EnqueueResumenEnviar", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encolar"})
		return
	}
	writeJSON(w, http.StatusAccepted, resumen)
}

func (h *ResumenesHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	items, err := h.Store.List(r.Context(), tenantID, 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type detalleResumen struct {
	*resumenes.Resumen
	Boletas []resumenes.BoletaPendiente `json:"boletas"`
}

func (h *ResumenesHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	resumen, err := h.Store.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, resumenes.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	boletas, _ := h.Store.Boletas(r.Context(), tenantID, id)
	writeJSON(w, http.StatusOK, detalleResumen{Resumen: resumen, Boletas: boletas})
}
