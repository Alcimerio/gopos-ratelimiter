package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/alcimerio/gopos-ratelimiter/pkg/storage"
)

type Config struct {
	IPLimit       int
	TokenLimit    int
	BlockDuration time.Duration
}

type RateLimiter struct {
	storage storage.Storage
	config  Config
}

func NewRateLimiter(storage storage.Storage, config Config) *RateLimiter {
	return &RateLimiter{
		storage: storage,
		config:  config,
	}
}

func (rl *RateLimiter) CheckLimit(ctx context.Context, ip, token string) error {
	if token != "" {
		if blocked, err := rl.storage.IsBlocked(ctx, token); err != nil {
			return fmt.Errorf("failed to check token block status: %v", err)
		} else if blocked {
			return fmt.Errorf("token rate limit exceeded")
		}
	}

	if blocked, err := rl.storage.IsBlocked(ctx, ip); err != nil {
		return fmt.Errorf("failed to check IP block status: %v", err)
	} else if blocked {
		return fmt.Errorf("IP rate limit exceeded")
	}

	if token != "" {
		count, err := rl.storage.Increment(ctx, token, time.Second)
		if err != nil {
			return fmt.Errorf("failed to increment token counter: %v", err)
		}

		if count > int64(rl.config.TokenLimit) {
			if err := rl.storage.Block(ctx, token, rl.config.BlockDuration); err != nil {
				return fmt.Errorf("failed to block token: %v", err)
			}

			if err := rl.storage.Reset(ctx, token); err != nil {
				return fmt.Errorf("failed to reset token counter: %v", err)
			}
			return fmt.Errorf("token rate limit exceeded")
		}
		return nil
	}

	count, err := rl.storage.Increment(ctx, ip, time.Second)
	if err != nil {
		return fmt.Errorf("failed to increment IP counter: %v", err)
	}

	if count > int64(rl.config.IPLimit) {
		if err := rl.storage.Block(ctx, ip, rl.config.BlockDuration); err != nil {
			return fmt.Errorf("failed to block IP: %v", err)
		}

		if err := rl.storage.Reset(ctx, ip); err != nil {
			return fmt.Errorf("failed to reset IP counter: %v", err)
		}
		return fmt.Errorf("IP rate limit exceeded")
	}

	return nil
}
