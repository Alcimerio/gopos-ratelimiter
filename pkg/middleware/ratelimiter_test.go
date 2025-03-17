package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alcimerio/gopos-ratelimiter/pkg/limiter"
	"github.com/alcimerio/gopos-ratelimiter/pkg/storage"
)

func TestRateLimiterMiddleware(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	config := limiter.Config{
		IPLimit:       5,
		TokenLimit:    10,
		BlockDuration: 5 * time.Minute,
	}
	rateLimiter := limiter.NewRateLimiter(mockStorage, config)
	middleware := NewRateLimiterMiddleware(rateLimiter)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	t.Run("Middleware injection test", func(t *testing.T) {
		handler := middleware.Handler(nextHandler)
		if handler == nil {
			t.Error("Expected middleware to return a handler")
		}
	})

	t.Run("IP-based rate limiting", func(t *testing.T) {
		handler := middleware.Handler(nextHandler)
		ip := "192.168.1.1"

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Request %d: expected status code %d, got %d", i+1, http.StatusOK, rr.Code)
			}
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, rr.Code)
		}

		var response map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response body: %v", err)
		}

		expectedMsg := "you have reached the maximum number of requests or actions allowed within a certain time frame"
		if msg, exists := response["error"]; !exists {
			t.Error("Expected error message in response, but got none")
		} else if msg != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, msg)
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("Token-based rate limiting", func(t *testing.T) {
		mockStorage = storage.NewMockStorage()
		rateLimiter = limiter.NewRateLimiter(mockStorage, config)
		middleware = NewRateLimiterMiddleware(rateLimiter)
		handler := middleware.Handler(nextHandler)

		token := "abc123"

		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("API_KEY", token)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Request %d: expected status code %d, got %d", i+1, http.StatusOK, rr.Code)
			}
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("API_KEY", token)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, rr.Code)
		}

		var response map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response body: %v", err)
		}

		expectedMsg := "you have reached the maximum number of requests or actions allowed within a certain time frame"
		if msg, exists := response["error"]; !exists {
			t.Error("Expected error message in response, but got none")
		} else if msg != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, msg)
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}
