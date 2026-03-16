package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ai-product-assistant/config"
	"ai-product-assistant/internal/server"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := newLogger("prod")
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger = newLogger(cfg.AppEnv)
	slog.SetDefault(logger)

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), cfg.DBPingTimeout)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	srv := server.New(server.Config{
		Address:         cfg.HTTPAddr(),
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		IdleTimeout:     cfg.IdleTimeout,
		ShutdownTimeout: cfg.ShutdownTimeout,
	})

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "address", cfg.HTTPAddr())
		serverErr <- srv.Start()
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-sigCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	logger.Info("server stopped")
	return nil
}

func newLogger(appEnv string) *slog.Logger {
	if appEnv == "dev" {
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
