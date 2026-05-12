package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type TenantHandler struct {
	Store  *tenant.Store
	Logger *slog.Logger
}

type tenantDTO struct {
	ID              uuid.UUID `json:"id"`
	RUC             string    `json:"ruc"`
	RazonSocial     string    `json:"razon_social"`
	NombreComercial string    `json:"nombre_comercial,omitempty"`
	DireccionFiscal string    `json:"direccion_fiscal,omitempty"`
	Ubigeo          string    `json:"ubigeo,omitempty"`
	SunatMode       string    `json:"sunat_mode"`
	UsuarioSOL      string    `json:"usuario_sol,omitempty"`
	// La clave SOL NUNCA viaja al frontend. Solo se acepta en POST/PUT.
	CertPath string `json:"cert_path,omitempty"`
}

func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	tid, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rec, err := h.Store.ByID(r.Context(), tid)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
		return
	}
	writeJSON(w, http.StatusOK, tenantDTO{
		ID: rec.ID, RUC: rec.RUC, RazonSocial: rec.RazonSocial,
		NombreComercial: rec.NombreComercial, DireccionFiscal: rec.DireccionFiscal,
		Ubigeo: rec.Ubigeo, SunatMode: rec.SunatMode,
		UsuarioSOL: rec.UsuarioSOL, CertPath: rec.CertPath,
	})
}

type updatePerfilReq struct {
	RazonSocial     string `json:"razon_social"`
	NombreComercial string `json:"nombre_comercial"`
	DireccionFiscal string `json:"direccion_fiscal"`
	Ubigeo          string `json:"ubigeo"`
	SunatMode       string `json:"sunat_mode"`
}

func (h *TenantHandler) UpdatePerfil(w http.ResponseWriter, r *http.Request) {
	tid, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	// Solo dueño puede cambiar config
	claims, _ := auth.FromContext(r.Context())
	if claims == nil || claims.Rol != "dueno" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "solo_dueno"})
		return
	}
	var req updatePerfilReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	in := tenant.PerfilInput{
		RazonSocial:     req.RazonSocial,
		NombreComercial: req.NombreComercial,
		DireccionFiscal: req.DireccionFiscal,
		Ubigeo:          req.Ubigeo,
		SunatMode:       req.SunatMode,
	}
	if err := h.Store.UpdatePerfil(r.Context(), tid, in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validacion", "detalle": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "actualizado"})
}

type updateCredsReq struct {
	UsuarioSOL string `json:"usuario_sol"`
	ClaveSOL   string `json:"clave_sol"`
}

func (h *TenantHandler) UpdateCredenciales(w http.ResponseWriter, r *http.Request) {
	tid, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	claims, _ := auth.FromContext(r.Context())
	if claims == nil || claims.Rol != "dueno" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "solo_dueno"})
		return
	}
	var req updateCredsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	if req.UsuarioSOL == "" || req.ClaveSOL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "usuario_y_clave_requeridos"})
		return
	}
	if err := h.Store.UpdateCredencialesSOL(r.Context(), tid, req.UsuarioSOL, req.ClaveSOL); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "actualizado"})
}
