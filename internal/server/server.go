package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"ai-product-assistant/internal/handler/draft"
	"ai-product-assistant/internal/handler/httpapi"
	"ai-product-assistant/internal/handler/middleware"
	"ai-product-assistant/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type Config struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	RateLimitRPM    int
	RateLimitWindow time.Duration
}

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

func New(cfg Config, draftService usecase.DraftService) (*Server, error) {
	if draftService == nil {
		return nil, errors.New("server: draft service is required")
	}

	router := chi.NewRouter()
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPM, cfg.RateLimitWindow)
	router.Use(rateLimiter.Middleware)

	draftHandler, err := draft.NewHandler(draftService)
	if err != nil {
		return nil, err
	}

	router.Get("/health", healthHandler)
	router.Get("/ready", readyHandler)
	router.Route("/drafts", func(r chi.Router) {
		r.Post("/generate", draftHandler.GenerateDraft)
		r.Get("/", draftHandler.ListDrafts)
		r.Get("/{id}", draftHandler.GetDraftByID)
		r.Post("/{id}/refine", draftHandler.RefineDraft)
	})

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Address,
			Handler:      router,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
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
	if err := httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		slog.Error("write health response failed", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func readyHandler(w http.ResponseWriter, _ *http.Request) {
	if err := httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"}); err != nil {
		slog.Error("write ready response failed", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
