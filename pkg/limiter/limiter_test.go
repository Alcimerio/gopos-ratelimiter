package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/alcimerio/gopos-ratelimiter/pkg/storage"
)

func TestRateLimiter(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	config := Config{
		IPLimit:       5,
		TokenLimit:    10,
		BlockDuration: 5 * time.Minute,
	}
	limiter := NewRateLimiter(mockStorage, config)
	ctx := context.Background()

	t.Run("IP-based rate limiting", func(t *testing.T) {
		ip := "192.168.1.1"
		
		// Should allow 5 requests
		for i := 0; i < 5; i++ {
			if err := limiter.CheckLimit(ctx, ip, ""); err != nil {
				t.Errorf("Expected request %d to be allowed, got error: %v", i+1, err)
			}
		}

		// 6th request should be blocked
		if err := limiter.CheckLimit(ctx, ip, ""); err == nil {
			t.Error("Expected 6th request to be blocked, but it was allowed")
		}

		// Advance time by block duration
		mockStorage.AdvanceTime(config.BlockDuration)

		// Should allow request after block duration
		if err := limiter.CheckLimit(ctx, ip, ""); err != nil {
			t.Errorf("Expected request after block duration to be allowed, got error: %v", err)
		}
	})

	t.Run("Token-based rate limiting", func(t *testing.T) {
		token := "abc123"
		ip := "192.168.1.2"

		// Reset storage for this test
		mockStorage = storage.NewMockStorage()
		limiter = NewRateLimiter(mockStorage, config)

		// Should allow 10 requests with token
		for i := 0; i < 10; i++ {
			if err := limiter.CheckLimit(ctx, ip, token); err != nil {
				t.Errorf("Expected request %d to be allowed, got error: %v", i+1, err)
			}
		}

		// 11th request should be blocked
		if err := limiter.CheckLimit(ctx, ip, token); err == nil {
			t.Error("Expected 11th request to be blocked, but it was allowed")
		}

		// Advance time by block duration
		mockStorage.AdvanceTime(config.BlockDuration)

		// Should allow request after block duration
		if err := limiter.CheckLimit(ctx, ip, token); err != nil {
			t.Errorf("Expected request after block duration to be allowed, got error: %v", err)
		}
	})

	t.Run("Token limit overrides IP limit", func(t *testing.T) {
		token := "abc456"
		ip := "192.168.1.3"

		// Reset storage for this test
		mockStorage = storage.NewMockStorage()
		limiter = NewRateLimiter(mockStorage, config)

		// Make 6 requests (exceeds IP limit but within token limit)
		for i := 0; i < 6; i++ {
			if err := limiter.CheckLimit(ctx, ip, token); err != nil {
				t.Errorf("Expected request %d to be allowed due to token limit, got error: %v", i+1, err)
			}
		}
	})

	t.Run("Requests without token are validated by IP", func(t *testing.T) {
		ip := "192.0.0.1"
		
		// Reset storage for this test
		mockStorage = storage.NewMockStorage()
		limiter = NewRateLimiter(mockStorage, config)
		
		// Should allow IPLimit (5) requests
		for i := 0; i < config.IPLimit; i++ {
			if err := limiter.CheckLimit(ctx, ip, ""); err != nil {
				t.Errorf("Expected request %d to be allowed, got error: %v", i+1, err)
			}
		}
		
		// Next request should be blocked (exceeds IP limit)
		if err := limiter.CheckLimit(ctx, ip, ""); err == nil {
			t.Error("Expected request to be blocked after exceeding IP limit, but it was allowed")
		}
	})
	
	t.Run("Token rate limit overrides IP limit", func(t *testing.T) {
		token := "token789"
		ip := "192.0.0.1"
		
		// Reset storage for this test
		mockStorage = storage.NewMockStorage()
		limiter = NewRateLimiter(mockStorage, config)
		
		// Make IP-only requests to reach IP limit
		for i := 0; i < config.IPLimit; i++ {
			if err := limiter.CheckLimit(ctx, ip, ""); err != nil {
				t.Errorf("Expected IP-only request %d to be allowed, got error: %v", i+1, err)
			}
		}
		
		// Verify IP is now rate limited
		if err := limiter.CheckLimit(ctx, ip, ""); err == nil {
			t.Error("Expected IP to be rate limited, but request was allowed")
		}
		
		// Now use the same IP but with a token
		for i := 0; i < config.TokenLimit; i++ {
			if err := limiter.CheckLimit(ctx, ip, token); err != nil {
				t.Errorf("Expected request with token %d to be allowed, got error: %v", i+1, err)
			}
		}
		
		// Verify token limit is enforced
		if err := limiter.CheckLimit(ctx, ip, token); err == nil {
			t.Error("Expected token to be rate limited after exceeding token limit, but request was allowed")
		}
	})
}
