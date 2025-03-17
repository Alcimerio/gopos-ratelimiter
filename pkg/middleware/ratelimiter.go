package middleware

import (
	"net/http"

	"github.com/alcimerio/gopos-ratelimiter/pkg/limiter"
)

type RateLimiterMiddleware struct {
	limiter *limiter.RateLimiter
}

func NewRateLimiterMiddleware(limiter *limiter.RateLimiter) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		limiter: limiter,
	}
}

func (m *RateLimiterMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get IP address from request
		ip := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ip = forwardedFor
		}

		// Get token from header
		token := r.Header.Get("API_KEY")

		// Check rate limit
		if err := m.limiter.CheckLimit(r.Context(), ip, token); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "you have reached the maximum number of requests or actions allowed within a certain time frame"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
