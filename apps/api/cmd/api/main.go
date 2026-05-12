package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/catalogos"
	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	"github.com/eltiokarma/facturador/apps/api/internal/db"
	httpapi "github.com/eltiokarma/facturador/apps/api/internal/http"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/pdf"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
	"github.com/eltiokarma/facturador/apps/api/internal/users"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config inválida", "err", err)
		os.Exit(1)
	}

	ten, err := tenant.FromEnv()
	if err != nil {
		logger.Error("config del tenant inválida", "err", err)
		os.Exit(1)
	}

	mat, err := cert.Load(ten.CertPath, ten.CertPassphrase)
	if err != nil {
		logger.Error("no pude cargar el certificado", "err", err, "path", ten.CertPath)
		os.Exit(1)
	}
	ten.CertPassphrase = ""
	logger.Info("certificado cargado en memoria", "path", ten.CertPath)

	ctxStart, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStart()

	pool, err := db.Connect(ctxStart, cfg.PostgresDSN)
	if err != nil {
		logger.Error("DB connect", "err", err)
		os.Exit(1)
	}
	if err := db.Migrate(ctxStart, pool); err != nil {
		logger.Error("DB migrate", "err", err)
		os.Exit(1)
	}
	logger.Info("DB conectada y migrada")

	tenantID, err := ensureTenant(ctxStart, pool, ten)
	if err != nil {
		logger.Error("ensureTenant", "err", err)
		os.Exit(1)
	}
	logger.Info("tenant resuelto", "ruc", ten.RUC, "id", tenantID)

	usersStore := users.NewStore(pool)
	compStore, err := comprobantes.NewStore(pool, cfg.DataDir)
	if err != nil {
		logger.Error("comprobantes.NewStore", "err", err)
		os.Exit(1)
	}

	signer := auth.NewSigner(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	mot := motor.New(cfg.MotorURL)

	clientesStore := catalogos.NewClientesStore(pool)
	productosStore := catalogos.NewProductosStore(pool)
	pdfBuilder := pdf.New(ten)

	authHandler := &httpapi.AuthHandler{Signer: signer, Users: usersStore, Logger: logger}
	facturasHandler := &httpapi.FacturasHandler{
		Cfg: cfg, Tenant: ten, Cert: mat, Motor: mot,
		Comprobantes: compStore, Logger: logger,
	}
	compHandler := &httpapi.ComprobantesHandler{Store: compStore, PDFBuilder: pdfBuilder, Logger: logger}
	clientesHandler := &httpapi.ClientesHandler{Store: clientesStore, Logger: logger}
	productosHandler := &httpapi.ProductosHandler{Store: productosStore, Logger: logger}

	router := httpapi.NewRouter(httpapi.Deps{
		Cfg: cfg, Logger: logger, Signer: signer,
		Auth: authHandler, Facturas: facturasHandler, Comprobantes: compHandler,
		Clientes: clientesHandler, Productos: productosHandler,
		MotorPing: func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return mot.Health(ctx)
		},
	})

	logger.Info("arrancando facturador API",
		"sunat_mode", cfg.SunatMode,
		"port", cfg.Port,
		"tenant_ruc", ten.RUC,
		"motor_url", cfg.MotorURL,
	)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("HTTP escuchando", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server falló", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("recibida señal, apagando")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown falló", "err", err)
	}
	pool.Close()
	logger.Info("apagado limpio")
}

// ensureTenant crea o actualiza la fila de la tabla tenants que corresponde
// al RUC configurado por env. Devuelve el UUID del tenant.
func ensureTenant(ctx context.Context, pool *pgxpool.Pool, t *tenant.Tenant) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO tenants (ruc, razon_social, nombre_comercial, direccion_fiscal, ubigeo, sunat_mode)
		VALUES ($1,$2,$3,$4,$5,'beta')
		ON CONFLICT (ruc) DO UPDATE SET
			razon_social = EXCLUDED.razon_social,
			nombre_comercial = EXCLUDED.nombre_comercial,
			direccion_fiscal = EXCLUDED.direccion_fiscal,
			ubigeo = EXCLUDED.ubigeo,
			updated_at = now()
		RETURNING id
	`, t.RUC, t.RazonSocial, t.NombreComercial, t.DireccionFiscal, t.Ubigeo).Scan(&id)
	return id, err
}
