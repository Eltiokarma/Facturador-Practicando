package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eltiokarma/facturador/apps/api/internal/auth"
	"github.com/eltiokarma/facturador/apps/api/internal/catalogos"
	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/comprobantes"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	mycrypto "github.com/eltiokarma/facturador/apps/api/internal/crypto"
	"github.com/eltiokarma/facturador/apps/api/internal/db"
	httpapi "github.com/eltiokarma/facturador/apps/api/internal/http"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/pdf"
	"github.com/eltiokarma/facturador/apps/api/internal/queue"
	"github.com/eltiokarma/facturador/apps/api/internal/resumenes"
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

	cipher, err := mycrypto.New(cfg.MasterKeyHex)
	if err != nil {
		logger.Error("crypto", "err", err)
		os.Exit(1)
	}

	tenantStore := tenant.NewStore(pool, cipher)
	tenantManager := tenant.NewManager(tenantStore)
	certsDir := filepath.Dir(getEnv("CERT_PATH", "/app/certs/cert.p12"))
	certManager, err := cert.NewManager(certsDir, cipher)
	if err != nil {
		logger.Error("cert.NewManager", "err", err)
		os.Exit(1)
	}

	// Bootstrap: si no hay tenants en DB y hay TENANT_* en env, crear el primero.
	if err := bootstrapInitialTenant(ctxStart, pool, tenantStore, certManager, logger); err != nil {
		logger.Error("bootstrap inicial", "err", err)
		os.Exit(1)
	}

	// Cargar todos los tenants al caché
	if err := tenantManager.LoadAll(ctxStart); err != nil {
		logger.Error("cargar tenants", "err", err)
		os.Exit(1)
	}
	tenants := tenantManager.All()
	logger.Info("tenants cargados", "n", len(tenants))

	// Intentar cargar cada cert en memoria
	for _, t := range tenants {
		if t.CertPath == "" || len(t.CertPassCipher) == 0 {
			logger.Warn("cert no configurado para tenant — subilo desde la UI",
				"tenant_id", t.ID, "ruc", t.RUC)
			continue
		}
		if err := certManager.LoadFromDisk(t.ID, t.CertPath, t.CertPassCipher); err != nil {
			logger.Warn("cert no se pudo cargar", "tenant_id", t.ID, "ruc", t.RUC, "err", err)
			continue
		}
		logger.Info("cert cargado", "tenant_id", t.ID, "ruc", t.RUC)
	}

	usersStore := users.NewStore(pool)
	compStore, err := comprobantes.NewStore(pool, cfg.DataDir)
	if err != nil {
		logger.Error("comprobantes.NewStore", "err", err)
		os.Exit(1)
	}
	clientesStore := catalogos.NewClientesStore(pool)
	productosStore := catalogos.NewProductosStore(pool)
	resumenesStore := resumenes.NewStore(pool)
	pdfBuilder := pdf.New()

	signer := auth.NewSigner(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	mot := motor.New(cfg.MotorURL)

	queueClient := queue.NewClient(cfg.RedisAddr)
	defer queueClient.Close()

	emitHandler := &queue.EmitHandler{
		Cfg: cfg, Tenants: tenantManager, Certs: certManager, Motor: mot,
		Comprobantes: compStore, Logger: logger,
	}
	resumenHandler := &queue.ResumenHandler{
		Cfg: cfg, Tenants: tenantManager, Certs: certManager, Motor: mot,
		Resumenes: resumenesStore, Client: queueClient, Logger: logger,
	}
	queueServer := queue.NewServer(cfg.RedisAddr, 5, logger)
	if err := queueServer.Start(emitHandler, resumenHandler); err != nil {
		logger.Error("queue.Start", "err", err)
		os.Exit(1)
	}
	logger.Info("worker de cola arrancado", "redis", cfg.RedisAddr)

	authHandler := &httpapi.AuthHandler{
		Signer: signer, Users: usersStore, Tenants: tenantManager, Logger: logger,
	}
	facturasHandler := &httpapi.FacturasHandler{
		Comprobantes: compStore, Queue: queueClient, Logger: logger,
	}
	compHandler := &httpapi.ComprobantesHandler{
		Store: compStore, PDFBuilder: pdfBuilder, Tenants: tenantManager, Logger: logger,
	}
	clientesHandler := &httpapi.ClientesHandler{Store: clientesStore, Logger: logger}
	productosHandler := &httpapi.ProductosHandler{Store: productosStore, Logger: logger}
	resumenesHandler := &httpapi.ResumenesHandler{Store: resumenesStore, Queue: queueClient, Logger: logger}
	tenantHandler := &httpapi.TenantHandler{
		Tenants: tenantManager, Certs: certManager, Logger: logger,
	}

	router := httpapi.NewRouter(httpapi.Deps{
		Cfg: cfg, Logger: logger, Signer: signer,
		Auth: authHandler, Facturas: facturasHandler, Comprobantes: compHandler,
		Clientes: clientesHandler, Productos: productosHandler,
		Resumenes: resumenesHandler, Tenant: tenantHandler,
		MotorPing: func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return mot.Health(ctx)
		},
	})

	logger.Info("arrancando facturador API",
		"sunat_mode", cfg.SunatMode,
		"port", cfg.Port,
		"tenants", len(tenants),
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
		logger.Error("HTTP shutdown", "err", err)
	}
	queueServer.Shutdown()
	pool.Close()
	logger.Info("apagado limpio")
}

func getEnv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

// bootstrapInitialTenant: si la DB está vacía y hay TENANT_* en env, crea
// el primer tenant. Migra también el cert.p12 legacy (./certs/cert.p12 +
// CERT_PASSPHRASE) al nuevo formato por tenant (./certs/<id>.p12).
func bootstrapInitialTenant(
	ctx context.Context,
	pool *pgxpool.Pool,
	store *tenant.Store,
	certs *cert.Manager,
	logger *slog.Logger,
) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	in := tenant.FromEnv()
	if in == nil {
		logger.Info("bootstrap: DB vacía y sin TENANT_* en env. " +
			"Creá el primer tenant desde la UI con 'Nueva empresa'.")
		return nil
	}

	id, err := store.Create(ctx, tenant.CreateInput{
		RUC: in.RUC, RazonSocial: in.RazonSocial,
		NombreComercial: in.NombreComercial,
		DireccionFiscal: in.DireccionFiscal, Ubigeo: in.Ubigeo,
	})
	if err != nil {
		return err
	}
	logger.Info("bootstrap: tenant inicial creado", "id", id, "ruc", in.RUC)

	if in.UsuarioSOL != "" && in.ClaveSOL != "" {
		if err := store.UpdateCredencialesSOL(ctx, id, in.UsuarioSOL, in.ClaveSOL); err != nil {
			return err
		}
	}

	// Migrar cert legacy si existe
	legacyPath := in.CertPath
	if legacyPath == "" {
		legacyPath = "/app/certs/cert.p12"
	}
	if _, err := os.Stat(legacyPath); err == nil && in.CertPassphrase != "" {
		raw, err := os.ReadFile(legacyPath)
		if err == nil {
			passCipher, err := certs.SaveAndLoad(id, raw, in.CertPassphrase)
			if err == nil {
				certPath := certs.PathFor(id)
				_ = store.SetCertRef(ctx, id, certPath, passCipher)
				logger.Info("bootstrap: cert legacy migrado a multi-tenant",
					"de", legacyPath, "a", certPath)
				// Mantenemos el archivo viejo por si el operador no quiere
				// confiar todavía en el nuevo. No lo borramos.
			} else {
				logger.Warn("bootstrap: no se pudo migrar cert legacy",
					"err", err)
			}
		}
	}
	return nil
}

// uuid.UUID se importa indirectamente; evitar warning
var _ = uuid.Nil
