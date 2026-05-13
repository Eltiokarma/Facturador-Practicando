package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type TenantHandler struct {
	Tenants *tenant.Manager
	Certs   *cert.Manager
	Logger  *slog.Logger
}

type tenantDTO struct {
	ID                  uuid.UUID `json:"id"`
	RUC                 string    `json:"ruc"`
	RazonSocial         string    `json:"razon_social"`
	NombreComercial     string    `json:"nombre_comercial,omitempty"`
	DireccionFiscal     string    `json:"direccion_fiscal,omitempty"`
	Ubigeo              string    `json:"ubigeo,omitempty"`
	SunatMode           string    `json:"sunat_mode"`
	UsuarioSOL          string    `json:"usuario_sol,omitempty"`
	GREClientID         string    `json:"gre_client_id,omitempty"`
	HasGRECredenciales  bool      `json:"has_gre_credenciales"`
	CertPath            string    `json:"cert_path,omitempty"`
	HasCert             bool      `json:"has_cert"`
	DemoMode            bool      `json:"demo_mode"`
}

func (h *TenantHandler) toDTO(r *tenant.Record) tenantDTO {
	return tenantDTO{
		ID: r.ID, RUC: r.RUC, RazonSocial: r.RazonSocial,
		NombreComercial: r.NombreComercial, DireccionFiscal: r.DireccionFiscal,
		Ubigeo: r.Ubigeo, SunatMode: r.SunatMode,
		UsuarioSOL: r.UsuarioSOL, CertPath: r.CertPath,
		HasCert:            h.Certs.HasCert(r.ID),
		DemoMode:           r.DemoMode,
		GREClientID:        r.GREClientID,
		HasGRECredenciales: r.GREClientID != "" && r.GREClientSecret != "",
	}
}

func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	tid, ok := auth.TenantIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rec, err := h.Tenants.Get(r.Context(), tid)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_encontrado"})
		return
	}
	writeJSON(w, http.StatusOK, h.toDTO(rec))
}

type updatePerfilReq struct {
	RazonSocial     string `json:"razon_social"`
	NombreComercial string `json:"nombre_comercial"`
	DireccionFiscal string `json:"direccion_fiscal"`
	Ubigeo          string `json:"ubigeo"`
	SunatMode       string `json:"sunat_mode"`
	DemoMode        *bool  `json:"demo_mode,omitempty"`
}

func (h *TenantHandler) UpdatePerfil(w http.ResponseWriter, r *http.Request) {
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
	var req updatePerfilReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	in := tenant.PerfilInput{
		RazonSocial: req.RazonSocial, NombreComercial: req.NombreComercial,
		DireccionFiscal: req.DireccionFiscal, Ubigeo: req.Ubigeo, SunatMode: req.SunatMode,
	}
	if err := h.Tenants.Store().UpdatePerfil(r.Context(), tid, in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "validacion", "detalle": err.Error()})
		return
	}
	// Si pidieron cambiar demo_mode, validar que si lo desactivan haya cert.
	if req.DemoMode != nil {
		if !*req.DemoMode && !h.Certs.HasCert(tid) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "no_cert",
				"detalle": "no se puede salir de modo DEMO sin certificado .p12 cargado",
			})
			return
		}
		if err := h.Tenants.Store().SetDemoMode(r.Context(), tid, *req.DemoMode); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
	}
	if _, err := h.Tenants.Refresh(r.Context(), tid); err != nil {
		h.Logger.Warn("refresh tenant tras update", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "actualizado"})
}

type updateCredsReq struct {
	UsuarioSOL string `json:"usuario_sol"`
	ClaveSOL   string `json:"clave_sol"`
}

type updateGRECredsReq struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// UpdateGRECredenciales: el dueño guarda las credenciales OAuth2 que SUNAT
// entrega para emitir GRE vía REST. Se cifran con MASTER_KEY.
func (h *TenantHandler) UpdateGRECredenciales(w http.ResponseWriter, r *http.Request) {
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
	var req updateGRECredsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	if req.ClientID == "" || req.ClientSecret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id_y_secret_requeridos"})
		return
	}
	if err := h.Tenants.Store().UpdateGRECredenciales(r.Context(), tid, req.ClientID, req.ClientSecret); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if _, err := h.Tenants.Refresh(r.Context(), tid); err != nil {
		h.Logger.Warn("refresh tenant", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "actualizado"})
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
	if err := h.Tenants.Store().UpdateCredencialesSOL(r.Context(), tid, req.UsuarioSOL, req.ClaveSOL); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if _, err := h.Tenants.Refresh(r.Context(), tid); err != nil {
		h.Logger.Warn("refresh tenant", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "actualizado"})
}

// UploadCert recibe un .p12 multipart con la passphrase, lo guarda en disco,
// intenta cargarlo y lo cachea. Si funciona, cifra la passphrase y la guarda
// en DB para auto-cargas futuras.
//
// form fields:
//   - file: el archivo .p12
//   - passphrase: la passphrase para descifrarlo
func (h *TenantHandler) UploadCert(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(4 << 20); err != nil { // 4 MB máx — un .p12 pesa <50 KB
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "form_invalido", "detalle": err.Error()})
		return
	}
	passphrase := r.FormValue("passphrase")
	if passphrase == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "passphrase_requerida"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file_requerido"})
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, 4<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "leer_archivo"})
		return
	}

	passCipher, err := h.Certs.SaveAndLoad(tid, body, passphrase)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cert_invalido", "detalle": err.Error()})
		return
	}
	certPath := h.Certs.PathFor(tid)
	if err := h.Tenants.Store().SetCertRef(r.Context(), tid, certPath, passCipher); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persistir"})
		return
	}
	if _, err := h.Tenants.Refresh(r.Context(), tid); err != nil {
		h.Logger.Warn("refresh tenant tras subir cert", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "cert_cargado"})
}

// Create permite a un usuario logueado crear un tenant nuevo. Se vuelve
// automáticamente "dueno" del tenant nuevo.
type createReq struct {
	RUC             string `json:"ruc"`
	RazonSocial     string `json:"razon_social"`
	NombreComercial string `json:"nombre_comercial"`
	DireccionFiscal string `json:"direccion_fiscal"`
	Ubigeo          string `json:"ubigeo"`
}

func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json_invalido"})
		return
	}
	id, err := h.Tenants.Store().Create(r.Context(), tenant.CreateInput{
		RUC: req.RUC, RazonSocial: req.RazonSocial,
		NombreComercial: req.NombreComercial,
		DireccionFiscal: req.DireccionFiscal,
		Ubigeo: req.Ubigeo,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "crear", "detalle": err.Error()})
		return
	}
	if err := h.Tenants.Store().AddMember(r.Context(), id, userID, "dueno"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "add_member", "detalle": err.Error()})
		return
	}
	rec, err := h.Tenants.Refresh(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "refresh"})
		return
	}
	writeJSON(w, http.StatusCreated, h.toDTO(rec))
}

// MisTenants lista los tenants a los que pertenece el usuario actual.
func (h *TenantHandler) MisTenants(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	list, err := h.Tenants.Store().ByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	dtos := make([]tenantDTO, 0, len(list))
	for _, t := range list {
		dtos = append(dtos, h.toDTO(t))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": dtos})
}

var ErrFormBoundary = errors.New("missing form boundary")
