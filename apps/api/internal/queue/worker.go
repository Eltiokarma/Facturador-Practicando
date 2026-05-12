package queue

import (
	"context"
	"encoding/json"
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
	Tenant       *tenant.Tenant
	Cert         *cert.Material
	Motor        *motor.Cliente
	Comprobantes *comprobantes.Store
	Logger       *slog.Logger
}

func (h *EmitHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var p EmitPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("payload inválido: %w", err)
	}
	log := h.Logger.With("comprobante_id", p.ComprobanteID)

	_ = h.Comprobantes.MarcarEnviando(ctx, p.TenantID, p.ComprobanteID)

	d, err := h.Comprobantes.Get(ctx, p.TenantID, p.ComprobanteID)
	if err != nil {
		log.Error("worker: leer comprobante", "err", err)
		return err
	}

	// Si ya fue aceptado/rechazado (idempotencia ante reintentos del job)
	if d.Estado == "aceptado" || d.Estado == "aceptado_con_obs" || d.Estado == "rechazado" {
		log.Info("worker: comprobante ya finalizado, ignorando job", "estado", d.Estado)
		return nil
	}

	// Reconstruir payload del comprobante desde JSON
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
		log.Error("worker: payload corrupto", "err", err)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, "payload corrupto")
		return nil // no reintentar — el payload no se va a arreglar solo
	}

	req := motor.EmitirRequest{
		Modo: string(h.Cfg.SunatMode),
		Tenant: map[string]any{
			"ruc":              h.Tenant.RUC,
			"razon_social":     h.Tenant.RazonSocial,
			"nombre_comercial": h.Tenant.NombreComercial,
			"direccion_fiscal": h.Tenant.DireccionFiscal,
			"ubigeo":           h.Tenant.Ubigeo,
			"departamento":     h.Tenant.Departamento,
			"provincia":        h.Tenant.Provincia,
			"distrito":         h.Tenant.Distrito,
			"usuario_sol":      h.Tenant.UsuarioSOL,
			"clave_sol":        h.Tenant.ClaveSOL,
			"cert_pem":         h.Cert.CertPEM,
			"cert_key_pem":     h.Cert.KeyPEM,
		},
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

	resp, err := h.Motor.Emitir(ctx, req)
	if err != nil {
		log.Warn("worker: motor inalcanzable, asynq reintentará", "err", err)
		_ = h.Comprobantes.MarcarError(ctx, p.TenantID, p.ComprobanteID, err.Error())
		return err // retorna error → asynq reintenta
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
		log.Error("worker: aplicar resultado", "err", err)
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

// Server arranca el server asynq que consume jobs. Bloqueante.
// Llamarlo en una goroutine separada y usar Shutdown() para terminarlo limpiamente.
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
				// Backoff exponencial: 2s, 4s, 8s, 16s, 32s, 64s, 128s
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

// asynqSlog adapta slog al logger que asynq espera.
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
