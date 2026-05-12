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

	"github.com/eltiokarma/facturador/apps/api/internal/config"
	httpapi "github.com/eltiokarma/facturador/apps/api/internal/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config inválida", "err", err)
		os.Exit(1)
	}

	logger.Info("arrancando facturador API",
		"sunat_mode", cfg.SunatMode,
		"port", cfg.Port,
	)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(cfg, logger),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
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
