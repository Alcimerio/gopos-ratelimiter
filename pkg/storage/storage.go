package storage

import (
	"context"
	"time"
)

// Storage defines the interface for rate limiter storage implementations
type Storage interface {
	// Increment increments the counter for a key and returns the current count
	Increment(ctx context.Context, key string, expiration time.Duration) (int64, error)
	
	// IsBlocked checks if a key is currently blocked
	IsBlocked(ctx context.Context, key string) (bool, error)
	
	// Block sets a block on a key for the specified duration
	Block(ctx context.Context, key string, duration time.Duration) error
	
	// Reset resets the counter for a key
	Reset(ctx context.Context, key string) error
	
	// Close closes the storage connection
	Close() error
}
