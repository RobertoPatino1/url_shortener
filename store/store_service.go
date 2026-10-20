package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Falla en compilación si StorageService deja de cumplir Store.
var _ Store = (*StorageService)(nil)

type StorageService struct {
	client *redis.Client
	ttl    time.Duration // 0 = sin expiración
}

func NewStorageService(ctx context.Context, addr string, ttl time.Duration) (*StorageService, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &StorageService{client: client, ttl: ttl}, nil
}

func (s *StorageService) Save(ctx context.Context, code, originalURL string) error {
	if err := s.client.Set(ctx, code, originalURL, s.ttl).Err(); err != nil {
		return fmt.Errorf("save %q: %w", code, err)
	}
	return nil
}

func (s *StorageService) Retrieve(ctx context.Context, code string) (string, error) {
	val, err := s.client.Get(ctx, code).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("retrieve %q: %w", code, err)
	}
	return val, nil
}
