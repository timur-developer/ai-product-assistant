package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Config struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

func New(cfg Config) *Server {
	router := chi.NewRouter()
	router.Get("/health", healthHandler)
	router.Get("/ready", readyHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Address,
			Handler:      router,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, s.shutdownTimeout)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	if err := writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		slog.Error("write health response failed", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func readyHandler(w http.ResponseWriter, _ *http.Request) {
	if err := writeJSON(w, http.StatusOK, map[string]string{"status": "ready"}); err != nil {
		slog.Error("write ready response failed", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = w.Write(body)
	return err
}
