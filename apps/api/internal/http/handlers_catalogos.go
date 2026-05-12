package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/catalogos"
)

type ClientesHandler struct {
	Store  *catalogos.ClientesStore
	Logger *slog.Logger
}

func (h *ClientesHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.Store.List(r.Context(), tenantID, q, limit)
	if err != nil {
		h.Logger.Error("clientes.List", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ClientesHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	var c catalogos.Cliente
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	// permite mandar id en URL para edición
	if idStr := chi.URLParam(r, "id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
			return
		}
		c.ID = id
	}
	if err := h.Store.Upsert(r.Context(), tenantID, &c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validacion", "detalle": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *ClientesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	if err := h.Store.Delete(r.Context(), tenantID, id); err != nil {
		if errors.Is(err, catalogos.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------- Productos ---------------------------------

type ProductosHandler struct {
	Store  *catalogos.ProductosStore
	Logger *slog.Logger
}

func (h *ProductosHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.Store.List(r.Context(), tenantID, q, limit)
	if err != nil {
		h.Logger.Error("productos.List", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ProductosHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	var p catalogos.Producto
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	if idStr := chi.URLParam(r, "id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
			return
		}
		p.ID = id
	}
	if err := h.Store.Upsert(r.Context(), tenantID, &p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validacion", "detalle": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProductosHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := auth.TenantIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id_invalido"})
		return
	}
	if err := h.Store.Delete(r.Context(), tenantID, id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
