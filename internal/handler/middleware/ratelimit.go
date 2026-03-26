package middleware

import (
	"net/http"
	"sync"
	"time"

	"ai-product-assistant/internal/handler/httpapi"
)

type RateLimiter struct {
	limit  int
	window time.Duration
	nowFn  func() time.Time

	mu    sync.Mutex
	state map[string]rateState
}

type rateState struct {
	windowStart time.Time
	count       int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = time.Minute
	}

	return &RateLimiter{
		limit:  limit,
		window: window,
		nowFn:  time.Now,
		state:  make(map[string]rateState),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := ClientIP(r)
		if !rl.allow(key) {
			httpapi.WriteError(w, http.StatusTooManyRequests, httpapi.ErrCodeRateLimited, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(key string) bool {
	now := rl.nowFn()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	st, ok := rl.state[key]
	if !ok || now.Sub(st.windowStart) >= rl.window {
		rl.state[key] = rateState{
			windowStart: now,
			count:       1,
		}
		return true
	}

	if st.count >= rl.limit {
		return false
	}

	st.count++
	rl.state[key] = st
	return true
}
