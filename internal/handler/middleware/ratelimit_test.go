package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	first := makeReq()
	if first.Code != http.StatusOK {
		t.Fatalf("first request status: %d", first.Code)
	}

	second := makeReq()
	if second.Code != http.StatusOK {
		t.Fatalf("second request status: %d", second.Code)
	}

	third := makeReq()
	if third.Code != http.StatusTooManyRequests {
		t.Fatalf("third request status: %d", third.Code)
	}
}
