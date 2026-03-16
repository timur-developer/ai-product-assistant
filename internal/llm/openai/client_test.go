package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ai-product-assistant/internal/model"
)

func TestClientGenerateDraftSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"summary":"sum","audience":"aud","value":"val","scenarios":["s1"],"constraints":["c1"],"risks":["r1"],"questions":["q1"]}`,
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 0)
	result, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err != nil {
		t.Fatalf("generate draft: %v", err)
	}
	if result.Content.Summary != "sum" {
		t.Fatalf("unexpected summary: %s", result.Content.Summary)
	}
	if result.Usage.TotalTokens != 30 {
		t.Fatalf("unexpected total tokens: %d", result.Usage.TotalTokens)
	}
}

func TestClientRefineDraftSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"summary":"sum2","audience":"aud","value":"val","scenarios":["s1"],"constraints":["c1"],"risks":["r1"],"questions":["q1"]}`,
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     11,
				"completion_tokens": 21,
				"total_tokens":      32,
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 0)
	result, err := client.RefineDraft(context.Background(), model.LLMRefineParams{
		RawIdea: "idea",
		CurrentContent: model.DraftContent{
			Summary: "old",
		},
		Sections: []string{"summary"},
		Language: "ru",
	})
	if err != nil {
		t.Fatalf("refine draft: %v", err)
	}
	if result.Content.Summary != "sum2" {
		t.Fatalf("unexpected summary: %s", result.Content.Summary)
	}
}

func TestClientRetryOnProvider500(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			writeJSON(t, w, http.StatusInternalServerError, map[string]any{
				"error": map[string]any{
					"message": "temporary failure",
					"type":    "server_error",
				},
			})
			return
		}

		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"summary":"sum","audience":"aud","value":"val","scenarios":[],"constraints":[],"risks":[],"questions":[]}`,
					},
				},
			},
			"usage": map[string]any{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 1)
	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err != nil {
		t.Fatalf("generate with retry: %v", err)
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Fatalf("unexpected hits: %d", hits)
	}
}

func TestClientRetryOnProvider429(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			writeJSON(t, w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]any{
					"message": "rate limited",
					"type":    "rate_limit_error",
				},
			})
			return
		}

		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"summary":"sum","audience":"aud","value":"val","scenarios":[],"constraints":[],"risks":[],"questions":[]}`,
					},
				},
			},
			"usage": map[string]any{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 1)
	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err != nil {
		t.Fatalf("generate with retry: %v", err)
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Fatalf("unexpected hits: %d", hits)
	}
}

func TestClientRetryAfterHeader(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			writeJSON(t, w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]any{
					"message": "rate limited",
					"type":    "rate_limit_error",
				},
			})
			return
		}

		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"summary":"sum","audience":"aud","value":"val","scenarios":[],"constraints":[],"risks":[],"questions":[]}`,
					},
				},
			},
			"usage": map[string]any{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 1)
	var gotDelay time.Duration
	client.sleepFn = func(_ context.Context, d time.Duration) error {
		gotDelay = d
		return nil
	}

	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err != nil {
		t.Fatalf("generate with retry-after: %v", err)
	}

	if gotDelay < time.Second {
		t.Fatalf("expected retry-after >= 1s, got: %v", gotDelay)
	}
}

func TestClientNoRetryOnProvider400(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		writeJSON(t, w, http.StatusBadRequest, map[string]any{
			"error": map[string]any{
				"message": "bad request",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 2)
	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("expected ErrProvider, got: %v", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("unexpected hits: %d", hits)
	}
}

func TestClientContextCanceledDuringBackoff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{
				"message": "temporary failure",
				"type":    "server_error",
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:        server.URL,
		APIKey:         "test-key",
		Model:          "test-model",
		Timeout:        time.Second,
		MaxRetries:     2,
		RetryBaseDelay: time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GenerateDraft(ctx, model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got: %v", err)
	}
}

func TestClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(120 * time.Millisecond)
		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"summary":"sum","audience":"aud","value":"val","scenarios":[],"constraints":[],"risks":[],"questions":[]}`,
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:        server.URL,
		APIKey:         "test-key",
		Model:          "test-model",
		Timeout:        50 * time.Millisecond,
		MaxRetries:     0,
		RetryBaseDelay: time.Millisecond,
	})

	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got: %v", err)
	}
}

func TestClientInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{},
			"usage":   map[string]any{},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 0)
	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err == nil {
		t.Fatal("expected invalid response error")
	}
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got: %v", err)
	}
}

func TestClientInvalidJSONContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": "not-json",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 0)
	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err == nil {
		t.Fatal("expected invalid response error")
	}
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got: %v", err)
	}
}

func TestClientMarkdownFencedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": "```json\n{\"summary\":\"sum\",\"audience\":\"aud\",\"value\":\"val\",\"scenarios\":[],\"constraints\":[],\"risks\":[],\"questions\":[]}\n```",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL, 0)
	result, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err != nil {
		t.Fatalf("expected successful parse for fenced json: %v", err)
	}
	if result.Content.Summary != "sum" {
		t.Fatalf("unexpected summary: %s", result.Content.Summary)
	}
}

func TestClientResponseBodyTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		largeSummary := strings.Repeat("a", 512)
		writeJSON(t, w, http.StatusOK, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"summary":"` + largeSummary + `","audience":"aud","value":"val","scenarios":[],"constraints":[],"risks":[],"questions":[]}`,
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:          server.URL,
		APIKey:           "test-key",
		Model:            "test-model",
		Timeout:          time.Second,
		MaxRetries:       0,
		RetryBaseDelay:   time.Millisecond,
		MaxResponseBytes: 64,
	})

	_, err := client.GenerateDraft(context.Background(), model.LLMGenerateParams{
		RawIdea:  "idea",
		Language: "ru",
	})
	if err == nil {
		t.Fatal("expected error for large response body")
	}
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("expected ErrProvider, got: %v", err)
	}
}

func newTestClient(baseURL string, retries int) *Client {
	return NewClient(Config{
		BaseURL:        baseURL,
		APIKey:         "test-key",
		Model:          "test-model",
		Timeout:        time.Second,
		MaxRetries:     retries,
		RetryBaseDelay: time.Millisecond,
	})
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write response: %v", err)
	}
}
