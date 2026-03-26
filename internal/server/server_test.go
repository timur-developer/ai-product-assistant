package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-product-assistant/internal/model"
)

type draftServiceMock struct{}

func (m *draftServiceMock) GenerateDraft(_ context.Context, _, _ string) (model.Draft, model.DraftVersion, error) {
	return model.Draft{}, model.DraftVersion{}, nil
}

func (m *draftServiceMock) RefineDraft(_ context.Context, _ int64, _ []string, _ string) (model.Draft, model.DraftVersion, error) {
	return model.Draft{}, model.DraftVersion{}, nil
}

func (m *draftServiceMock) GetDraft(_ context.Context, _ int64) (model.Draft, model.DraftVersion, error) {
	return model.Draft{}, model.DraftVersion{}, nil
}

func (m *draftServiceMock) ListDrafts(_ context.Context, _, _ int) ([]model.DraftListItem, error) {
	return nil, nil
}

func TestServerHealthEndpoint(t *testing.T) {
	srv, err := New(Config{
		Address:         ":0",
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
		RateLimitRPM:    20,
		RateLimitWindow: time.Minute,
	}, &draftServiceMock{})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestServerDraftRouteRegistered(t *testing.T) {
	srv, err := New(Config{
		Address:         ":0",
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
		RateLimitRPM:    20,
		RateLimitWindow: time.Minute,
	}, &draftServiceMock{})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/drafts/generate", nil)
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
