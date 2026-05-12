package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/eltiokarma/facturador/apps/api/internal/config"
)

type Deps struct {
	Cfg      *config.Config
	Logger   *slog.Logger
	Facturas *FacturasHandler
	MotorPing func() error
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
		if d.Facturas != nil {
			r.Post("/facturas", d.Facturas.Emitir)
			r.Post("/boletas", d.Facturas.Emitir) // mismo handler, distingue por tipo en el payload
		}
		r.Post("/notas-credito", notImplemented("emitir nota de crédito"))
		r.Post("/notas-debito", notImplemented("emitir nota de débito"))
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

func notImplemented(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error":  "not_implemented",
			"action": action,
		})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
