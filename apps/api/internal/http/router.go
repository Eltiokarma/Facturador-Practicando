package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
)

type Deps struct {
	Cfg          *config.Config
	Logger       *slog.Logger
	Signer       *auth.Signer
	Auth         *AuthHandler
	Facturas     *FacturasHandler
	Comprobantes *ComprobantesHandler
	Clientes     *ClientesHandler
	Productos    *ProductosHandler
	Resumenes    *ResumenesHandler
	Tenant       *TenantHandler
	MotorPing    func() error
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(90 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   d.Cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", health(d.Cfg))
	r.Get("/ready", ready(d))

	r.Route("/api/v1", func(r chi.Router) {
		// Rutas públicas
		if d.Auth != nil {
			r.Post("/auth/login", d.Auth.Login)
			r.Post("/auth/refresh", d.Auth.Refresh)
		}

		// Rutas autenticadas
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(d.Signer))
			if d.Auth != nil {
				r.Get("/me", d.Auth.Me)
			}
			if d.Facturas != nil {
				r.Post("/facturas", d.Facturas.Emitir)
				r.Post("/boletas", d.Facturas.Emitir)
			}
			if d.Comprobantes != nil {
				r.Get("/comprobantes", d.Comprobantes.List)
				r.Get("/comprobantes/{id}", d.Comprobantes.Get)
				r.Get("/comprobantes/{id}/xml", d.Comprobantes.XML)
				r.Get("/comprobantes/{id}/cdr", d.Comprobantes.CDR)
				r.Get("/comprobantes/{id}/pdf", d.Comprobantes.PDF)
			}
			if d.Clientes != nil {
				r.Get("/clientes", d.Clientes.List)
				r.Post("/clientes", d.Clientes.Upsert)
				r.Put("/clientes/{id}", d.Clientes.Upsert)
				r.Delete("/clientes/{id}", d.Clientes.Delete)
			}
			if d.Productos != nil {
				r.Get("/productos", d.Productos.List)
				r.Post("/productos", d.Productos.Upsert)
				r.Put("/productos/{id}", d.Productos.Upsert)
				r.Delete("/productos/{id}", d.Productos.Delete)
			}
			if d.Resumenes != nil {
				r.Get("/resumenes", d.Resumenes.List)
				r.Get("/resumenes/pendientes", d.Resumenes.Pendientes)
				r.Post("/resumenes", d.Resumenes.Crear)
				r.Get("/resumenes/{id}", d.Resumenes.Get)
			}
			if d.Tenant != nil {
				r.Get("/tenant", d.Tenant.Get)
				r.Put("/tenant", d.Tenant.UpdatePerfil)
				r.Put("/tenant/credenciales", d.Tenant.UpdateCredenciales)
			}
		})
	})

	return r
}

type healthResp struct {
	Status    string    `json:"status"`
	SunatMode string    `json:"sunat_mode"`
	Time      time.Time `json:"time"`
}

func health(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResp{
			Status:    "ok",
			SunatMode: string(cfg.SunatMode),
			Time:      time.Now().UTC(),
		})
	}
}

func ready(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		motorErr := ""
		if d.MotorPing != nil {
			if err := d.MotorPing(); err != nil {
				motorErr = err.Error()
			}
		}
		status := http.StatusOK
		body := map[string]string{"motor": "ok"}
		if motorErr != "" {
			status = http.StatusServiceUnavailable
			body["motor"] = motorErr
		}
		writeJSON(w, status, body)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
