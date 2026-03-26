package draft

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-product-assistant/internal/model"
	"ai-product-assistant/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type serviceMock struct {
	generateFn func(ctx context.Context, rawIdea, language string) (model.Draft, model.DraftVersion, error)
	refineFn   func(ctx context.Context, draftID int64, sections []string, language string) (model.Draft, model.DraftVersion, error)
	getFn      func(ctx context.Context, draftID int64) (model.Draft, model.DraftVersion, error)
	listFn     func(ctx context.Context, limit, offset int) ([]model.DraftListItem, error)
}

func (m *serviceMock) GenerateDraft(ctx context.Context, rawIdea, language string) (model.Draft, model.DraftVersion, error) {
	return m.generateFn(ctx, rawIdea, language)
}

func (m *serviceMock) RefineDraft(ctx context.Context, draftID int64, sections []string, language string) (model.Draft, model.DraftVersion, error) {
	return m.refineFn(ctx, draftID, sections, language)
}

func (m *serviceMock) GetDraft(ctx context.Context, draftID int64) (model.Draft, model.DraftVersion, error) {
	return m.getFn(ctx, draftID)
}

func (m *serviceMock) ListDrafts(ctx context.Context, limit, offset int) ([]model.DraftListItem, error) {
	return m.listFn(ctx, limit, offset)
}

func TestGenerateDraftSuccess(t *testing.T) {
	h, err := NewHandler(&serviceMock{
		generateFn: func(_ context.Context, rawIdea, language string) (model.Draft, model.DraftVersion, error) {
			if rawIdea != "idea" {
				t.Fatalf("unexpected raw idea: %s", rawIdea)
			}
			if language != "ru" {
				t.Fatalf("unexpected language: %s", language)
			}

			now := time.Now().UTC()
			return model.Draft{
					ID:            1,
					RawIdea:       rawIdea,
					Language:      language,
					LatestVersion: 1,
					CreatedAt:     now,
					UpdatedAt:     now,
				}, model.DraftVersion{
					ID:      10,
					DraftID: 1,
					Version: 1,
					Content: model.DraftContent{
						Summary:     "summary",
						Audience:    "audience",
						Value:       "value",
						Scenarios:   []string{},
						Constraints: []string{},
						Risks:       []string{},
						Questions:   []string{},
					},
					Provider:  "openai",
					ModelName: "gpt-4o-mini",
					Usage: model.LLMUsage{
						PromptTokens:     1,
						CompletionTokens: 2,
						TotalTokens:      3,
					},
					CreatedAt: now,
				}, nil
		},
		refineFn: nil,
		getFn:    nil,
		listFn:   nil,
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/drafts/generate", strings.NewReader(`{"raw_idea":"idea","language":"ru"}`))
	rec := httptest.NewRecorder()

	h.GenerateDraft(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if !strings.Contains(rec.Body.String(), `"id":1`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestGenerateDraftInvalidIdeaLength(t *testing.T) {
	h, err := NewHandler(&serviceMock{
		generateFn: func(_ context.Context, _, _ string) (model.Draft, model.DraftVersion, error) {
			t.Fatal("service must not be called")
			return model.Draft{}, model.DraftVersion{}, nil
		},
		refineFn: nil,
		getFn:    nil,
		listFn:   nil,
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rawIdea := strings.Repeat("a", 4001)
	body := `{"raw_idea":"` + rawIdea + `","language":"ru"}`
	req := httptest.NewRequest(http.MethodPost, "/drafts/generate", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.GenerateDraft(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "raw_idea is too long") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestGenerateDraftUsecaseErrorMapping(t *testing.T) {
	h, err := NewHandler(&serviceMock{
		generateFn: func(_ context.Context, _, _ string) (model.Draft, model.DraftVersion, error) {
			return model.Draft{}, model.DraftVersion{}, errors.Join(usecase.ErrProviderFailed, errors.New("provider down"))
		},
		refineFn: nil,
		getFn:    nil,
		listFn:   nil,
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/drafts/generate", strings.NewReader(`{"raw_idea":"idea","language":"ru"}`))
	rec := httptest.NewRecorder()

	h.GenerateDraft(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"provider_failed"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestGetDraftByIDNotFound(t *testing.T) {
	h, err := NewHandler(&serviceMock{
		generateFn: nil,
		refineFn:   nil,
		getFn: func(_ context.Context, _ int64) (model.Draft, model.DraftVersion, error) {
			return model.Draft{}, model.DraftVersion{}, usecase.ErrNotFound
		},
		listFn: nil,
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	router := chi.NewRouter()
	router.Get("/drafts/{id}", h.GetDraftByID)

	req := httptest.NewRequest(http.MethodGet, "/drafts/123", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestListDraftsInvalidQuery(t *testing.T) {
	h, err := NewHandler(&serviceMock{
		generateFn: nil,
		refineFn:   nil,
		getFn:      nil,
		listFn: func(_ context.Context, _, _ int) ([]model.DraftListItem, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drafts?offset=-1", nil)
	rec := httptest.NewRecorder()

	h.ListDrafts(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "offset must be non-negative") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
