package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/users"
)

type AuthHandler struct {
	Signer *auth.Signer
	Users  *users.Store
	Logger *slog.Logger
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResp struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
	User    userDTO `json:"user"`
}

type userDTO struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Nombre   string    `json:"nombre"`
	Rol      string    `json:"rol"`
}

func toDTO(u *users.User) userDTO {
	return userDTO{
		ID: u.ID, TenantID: u.TenantID,
		Email: u.Email, Nombre: u.Nombre, Rol: string(u.Rol),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_y_password_requeridos"})
		return
	}
	u, err := h.Users.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, users.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "credenciales_invalidas"})
			return
		}
		h.Logger.Error("auth.Authenticate", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	access, err := h.Signer.Access(u.ID, u.TenantID, string(u.Rol))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token_emit"})
		return
	}
	refresh, err := h.Signer.Refresh(u.ID, u.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token_emit"})
		return
	}
	writeJSON(w, http.StatusOK, tokenResp{Access: access, Refresh: refresh, User: toDTO(u)})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token_requerido"})
		return
	}
	claims, err := h.Signer.Parse(req.RefreshToken, auth.KindRefresh)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "refresh_invalido"})
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "refresh_invalido"})
		return
	}
	u, err := h.Users.ByID(r.Context(), userID)
	if err != nil || !u.Activo {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "usuario_inactivo"})
		return
	}
	access, err := h.Signer.Access(u.ID, u.TenantID, string(u.Rol))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token_emit"})
		return
	}
	refresh, err := h.Signer.Refresh(u.ID, u.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token_emit"})
		return
	}
	writeJSON(w, http.StatusOK, tokenResp{Access: access, Refresh: refresh, User: toDTO(u)})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	u, err := h.Users.ByID(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
		return
	}
	writeJSON(w, http.StatusOK, toDTO(u))
}
