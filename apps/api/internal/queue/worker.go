package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
)

type EmitHandler struct {
	Cfg          *config.Config
	Tenants      *tenant.Manager
	Certs        *cert.Manager
	Motor        *motor.Cliente
	Comprobantes *comprobantes.Store
	Logger       *slog.Logger
}

func (h *EmitHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var p EmitPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("payload inválido: %w", err)
	}
	log := h.Logger.With("comprobante_id", p.ComprobanteID, "tenant_id", p.TenantID)

	_ = h.Comprobantes.MarcarEnviando(ctx, p.TenantID, p.ComprobanteID)

	d, err := h.Comprobantes.Get(ctx, p.TenantID, p.ComprobanteID)
	if err != nil {
		log.Error("worker: leer comprobante", "err", err)
		return err
	}
	if d.Estado == "aceptado" || d.Estado == "aceptado_con_obs" || d.Estado == "rechazado" {
		log.Info("worker: comprobante ya finalizado, ignorando", "estado", d.Estado)
		return nil
	}

	ten, err := h.Tenants.Get(ctx, p.TenantID)
	if err != nil {
		log.Error("worker: leer tenant", "err", err)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, "tenant no encontrado")
		return err
	}
	mat, err := h.Certs.Get(p.TenantID)
	if err != nil {
		// Si no hay cert cargado para este tenant, es un error de
		// configuración: marcar como error pero no reintentar.
		log.Error("worker: cert no disponible para este tenant", "err", err)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, "certificado no disponible — subilo desde Configuración")
		return nil // no devolver error → asynq no reintenta
	}

	var pld struct {
		Tipo          string                 `json:"tipo"`
		Serie         string                 `json:"serie"`
		Correlativo   int64                  `json:"correlativo"`
		FechaEmision  string                 `json:"fecha_emision"`
		Moneda        string                 `json:"moneda"`
		TipoOperacion string                 `json:"tipo_operacion"`
		Receptor      map[string]any         `json:"receptor"`
		Items         []map[string]any       `json:"items"`
		Totales       map[string]any         `json:"totales"`
	}
	if err := json.Unmarshal(d.Payload, &pld); err != nil {
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, "payload corrupto")
		return nil
	}

	req := motor.EmitirRequest{
		Modo:   string(h.Cfg.SunatMode), // fallback global; el tenant manda
		Tenant: tenantPayload(ten, mat),
		Comprobante: map[string]any{
			"tipo":           pld.Tipo,
			"serie":          pld.Serie,
			"correlativo":    pld.Correlativo,
			"fecha_emision":  pld.FechaEmision,
			"moneda":         pld.Moneda,
			"tipo_operacion": defaultStr(pld.TipoOperacion, "0101"),
			"receptor":       pld.Receptor,
			"items":          pld.Items,
			"totales":        pld.Totales,
		},
	}
	// El tenant manda su propio modo
	if ten.SunatMode == "prod" || ten.SunatMode == "beta" {
		req.Modo = ten.SunatMode
	}

	resp, err := h.Motor.Emitir(ctx, req)
	if err != nil {
		log.Warn("worker: motor inalcanzable, asynq reintentará", "err", err)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, err.Error())
		return err
	}

	res := comprobantes.Resultado{
		Estado:        resp.Estado,
		Codigo:        resp.Codigo,
		Mensaje:       resp.Mensaje,
		HashCPE:       resp.HashCPE,
		XMLFirmadoB64: resp.XMLFirmado,
		CDRZipB64:     resp.CDRZip,
	}
	if err := h.Comprobantes.AplicarResultado(ctx, p.TenantID, p.ComprobanteID, res, d.Tipo, d.Serie, d.Correlativo); err != nil {
		return err
	}
	log.Info("worker: comprobante procesado", "estado", resp.Estado, "codigo", resp.Codigo)
	return nil
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// Server arranca el server asynq que consume jobs.
type Server struct {
	srv *asynq.Server
}

func NewServer(redisAddr string, concurrency int, logger *slog.Logger) *Server {
	if concurrency <= 0 {
		concurrency = 5
	}
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: concurrency,
			RetryDelayFunc: func(n int, _ error, _ *asynq.Task) time.Duration {
				return time.Duration(1<<(n+1)) * time.Second
			},
			Logger: asynqSlog{l: logger},
		},
	)
	return &Server{srv: srv}
}

func (s *Server) Start(emit *EmitHandler, resumen *ResumenHandler) error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeEmit, emit.Handle)
	if resumen != nil {
		mux.HandleFunc(TypeResumenEnviar, resumen.HandleEnviar)
		mux.HandleFunc(TypeResumenStatus, resumen.HandleStatus)
	}
	return s.srv.Start(mux)
}

func (s *Server) Shutdown() { s.srv.Shutdown() }

type asynqSlog struct{ l *slog.Logger }

func (a asynqSlog) Debug(args ...interface{}) { a.l.Debug(joinArgs(args)) }
func (a asynqSlog) Info(args ...interface{})  { a.l.Info(joinArgs(args)) }
func (a asynqSlog) Warn(args ...interface{})  { a.l.Warn(joinArgs(args)) }
func (a asynqSlog) Error(args ...interface{}) { a.l.Error(joinArgs(args)) }
func (a asynqSlog) Fatal(args ...interface{}) { a.l.Error("FATAL " + joinArgs(args)) }

func joinArgs(args []any) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprint(a)
	}
	return out
}

// ErrNoCert se usa para indicar que un job no se puede procesar por falta de cert.
var ErrNoCert = errors.New("queue: certificado no disponible para el tenant")
