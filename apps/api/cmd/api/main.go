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

	"github.com/eltiokarma/facturador/apps/api/internal/cert"
	"github.com/eltiokarma/facturador/apps/api/internal/config"
	httpapi "github.com/eltiokarma/facturador/apps/api/internal/http"
	"github.com/eltiokarma/facturador/apps/api/internal/motor"
	"github.com/eltiokarma/facturador/apps/api/internal/storage"
	"github.com/eltiokarma/facturador/apps/api/internal/tenant"
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
	// Limpiar passphrase de memoria del struct: ya descifró, no la necesitamos más.
	ten.CertPassphrase = ""
	logger.Info("certificado cargado en memoria", "path", ten.CertPath)

	mot := motor.New(cfg.MotorURL)
	st, err := storage.New(cfg.DataDir)
	if err != nil {
		logger.Error("no pude inicializar storage", "err", err, "dir", cfg.DataDir)
		os.Exit(1)
	}

	handler := &httpapi.FacturasHandler{
		Cfg:    cfg,
		Tenant: ten,
		Cert:   mat,
		Motor:  mot,
		Store:  st,
		Logger: logger,
	}

	router := httpapi.NewRouter(httpapi.Deps{
		Cfg:      cfg,
		Logger:   logger,
		Facturas: handler,
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
		os.Exit(1)
	}
	logger.Info("apagado limpio")
}
