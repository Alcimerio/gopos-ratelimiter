package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func setupTestRedis(t *testing.T) (*RedisStorage, func()) {
	_ = godotenv.Load("../../.env")

	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 6379
	storage, err := NewRedisStorage(
		host,
		port,
		os.Getenv("REDIS_PASSWORD"),
		0,
	)
	if err != nil {
		t.Fatalf("Failed to create Redis storage: %v", err)
	}

	cleanup := func() {
		ctx := context.Background()
		storage.client.FlushDB(ctx)
		storage.Close()
	}

	return storage, cleanup
}

func TestRedisStorage_Integration(t *testing.T) {
	storage, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Connection test", func(t *testing.T) {
		if err := storage.client.Ping(ctx).Err(); err != nil {
			t.Fatalf("Failed to connect to Redis: %v", err)
		}
	})

	t.Run("Rate limiting scenario - IP based", func(t *testing.T) {
		ip := "192.168.1.1"
		expiration := time.Second

		for i := 1; i <= 5; i++ {
			count, err := storage.Increment(ctx, ip, expiration)
			if err != nil {
				t.Fatalf("Failed to increment counter: %v", err)
			}
			if count != int64(i) {
				t.Errorf("Expected count %d, got %d", i, count)
			}
		}

		ttl, err := storage.client.TTL(ctx, ip).Result()
		if err != nil {
			t.Fatalf("Failed to get TTL: %v", err)
		}
		if ttl <= 0 {
			t.Error("Expected positive TTL for rate limit key")
		}

		blockDuration := 5 * time.Second
		if err := storage.Block(ctx, ip, blockDuration); err != nil {
			t.Fatalf("Failed to block IP: %v", err)
		}

		blocked, err := storage.IsBlocked(ctx, ip)
		if err != nil {
			t.Fatalf("Failed to check blocked status: %v", err)
		}
		if !blocked {
			t.Error("Expected IP to be blocked")
		}

		time.Sleep(blockDuration + 100*time.Millisecond)

		blocked, err = storage.IsBlocked(ctx, ip)
		if err != nil {
			t.Fatalf("Failed to check blocked status: %v", err)
		}
		if blocked {
			t.Error("Expected IP to be unblocked after duration")
		}
	})

	t.Run("Rate limiting scenario - Token based", func(t *testing.T) {
		token := "test-token"
		expiration := time.Second

		for i := 1; i <= 10; i++ {
			count, err := storage.Increment(ctx, token, expiration)
			if err != nil {
				t.Fatalf("Failed to increment counter: %v", err)
			}
			if count != int64(i) {
				t.Errorf("Expected count %d, got %d", i, count)
			}
		}

		blockDuration := 5 * time.Second
		if err := storage.Block(ctx, token, blockDuration); err != nil {
			t.Fatalf("Failed to block token: %v", err)
		}

		blocked, err := storage.IsBlocked(ctx, token)
		if err != nil {
			t.Fatalf("Failed to check blocked status: %v", err)
		}
		if !blocked {
			t.Error("Expected token to be blocked")
		}

		if err := storage.Reset(ctx, token); err != nil {
			t.Fatalf("Failed to reset token: %v", err)
		}

		count, err := storage.client.Get(ctx, token).Int64()
		if err == nil {
			t.Errorf("Expected key to be deleted, but got count %d", count)
		}

		blocked, err = storage.IsBlocked(ctx, token)
		if err != nil {
			t.Fatalf("Failed to check blocked status after reset: %v", err)
		}
		if blocked {
			t.Error("Expected token to be unblocked after reset")
		}
	})

	t.Run("Concurrent access", func(t *testing.T) {
		key := "concurrent-test"
		expiration := time.Second
		iterations := 50
		done := make(chan bool)

		for i := 0; i < 5; i++ {
			go func() {
				for j := 0; j < iterations; j++ {
					_, err := storage.Increment(ctx, key, expiration)
					if err != nil {
						t.Errorf("Failed to increment in goroutine: %v", err)
					}
				}
				done <- true
			}()
		}

		for i := 0; i < 5; i++ {
			<-done
		}

		count, err := storage.client.Get(ctx, key).Int64()
		if err != nil {
			t.Fatalf("Failed to get final count: %v", err)
		}
		expectedCount := int64(5 * iterations)
		if count != expectedCount {
			t.Errorf("Expected count %d, got %d", expectedCount, count)
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		invalidStorage, err := NewRedisStorage("nonexistent", 6379, "", 0)
		if err == nil {
			t.Error("Expected error for invalid Redis connection")
			invalidStorage.Close()
		}

		storage.Close()
		_, err = storage.Increment(ctx, "test", time.Second)
		if err == nil {
			t.Error("Expected error when using closed client")
		}
	})

	t.Run("Multiple blocks and resets", func(t *testing.T) {
		storage, cleanup := setupTestRedis(t)
		defer cleanup()

		key := "multiple-blocks"
		blockDuration := 2 * time.Second

		for i := 0; i < 3; i++ {
			if err := storage.Block(ctx, key, blockDuration); err != nil {
				t.Fatalf("Failed to block on iteration %d: %v", i, err)
			}

			blocked, err := storage.IsBlocked(ctx, key)
			if err != nil {
				t.Fatalf("Failed to check blocked status: %v", err)
			}
			if !blocked {
				t.Errorf("Expected key to be blocked on iteration %d", i)
			}

			if i < 2 {
				if err := storage.Reset(ctx, key); err != nil {
					t.Fatalf("Failed to reset on iteration %d: %v", i, err)
				}

				blocked, err = storage.IsBlocked(ctx, key)
				if err != nil {
					t.Fatalf("Failed to check blocked status after reset: %v", err)
				}
				if blocked {
					t.Errorf("Expected key to be unblocked after reset on iteration %d", i)
				}
			}
		}
	})

	t.Run("Increment after expiration", func(t *testing.T) {
		storage, cleanup := setupTestRedis(t)
		defer cleanup()

		key := "expire-test"
		shortExpiration := 1 * time.Second

		count, err := storage.Increment(ctx, key, shortExpiration)
		if err != nil {
			t.Fatalf("Failed to increment: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}

		time.Sleep(shortExpiration + 100*time.Millisecond)

		count, err = storage.Increment(ctx, key, shortExpiration)
		if err != nil {
			t.Fatalf("Failed to increment after expiration: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count to reset to 1 after expiration, got %d", count)
		}
	})
}

func TestRedisStorage_BlockExpiration(t *testing.T) {
	storage, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	key := "expiration-test"
	shortDuration := 2 * time.Second

	if err := storage.Block(ctx, key, shortDuration); err != nil {
		t.Fatalf("Failed to block key: %v", err)
	}

	blocked, err := storage.IsBlocked(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check blocked status: %v", err)
	}
	if !blocked {
		t.Error("Expected key to be blocked")
	}

	time.Sleep(shortDuration + 100*time.Millisecond)

	blocked, err = storage.IsBlocked(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check blocked status: %v", err)
	}
	if blocked {
		t.Error("Expected key to be unblocked after expiration")
	}
}
