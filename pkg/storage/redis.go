package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(host string, port int, password string, db int) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisStorage{client: client}, nil
}

func (r *RedisStorage) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiration)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment key: %v", err)
	}
	
	return incr.Val(), nil
}

func (r *RedisStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	blockedKey := fmt.Sprintf("blocked:%s", key)
	exists, err := r.client.Exists(ctx, blockedKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blocked status: %v", err)
	}
	return exists == 1, nil
}

func (r *RedisStorage) Block(ctx context.Context, key string, duration time.Duration) error {
	blockedKey := fmt.Sprintf("blocked:%s", key)
	err := r.client.Set(ctx, blockedKey, "1", duration).Err()
	if err != nil {
		return fmt.Errorf("failed to set block: %v", err)
	}
	return nil
}

func (r *RedisStorage) Reset(ctx context.Context, key string) error {
	pipe := r.client.Pipeline()
	
	// Delete both the counter key and the blocked key
	pipe.Del(ctx, key)
	pipe.Del(ctx, fmt.Sprintf("blocked:%s", key))
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset keys: %v", err)
	}
	return nil
}

func (r *RedisStorage) Close() error {
	return r.client.Close()
}
