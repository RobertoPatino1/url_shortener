package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "url:"

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration // 0 = sin expiración
}

var _ Store = (*RedisStore)(nil)

func NewRedisStore(ctx context.Context, addr, password string, ttl time.Duration) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisStore{client: client, ttl: ttl}, nil
}

func (s *RedisStore) Create(ctx context.Context, code, originalURL string) error {
	ok, err := s.client.SetNX(ctx, keyPrefix+code, originalURL, s.ttl).Result()
	if err != nil {
		return fmt.Errorf("create %q: %w", code, err)
	}
	if !ok {
		return ErrExists
	}
	return nil
}

func (s *RedisStore) Retrieve(ctx context.Context, code string) (string, error) {
	val, err := s.client.Get(ctx, keyPrefix+code).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("retrieve %q: %w", code, err)
	}
	return val, nil
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) Close() error { return s.client.Close() }
