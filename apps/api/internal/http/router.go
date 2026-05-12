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

func NewRouter(cfg *config.Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", health(cfg))
	r.Get("/ready", ready(cfg))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/facturas", notImplemented("emitir factura"))
		r.Get("/facturas/{id}", notImplemented("consultar factura"))
		r.Post("/boletas", notImplemented("emitir boleta"))
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

func ready(_ *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: chequear postgres, redis y motor.
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
