package store

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("short url not found")

type Store interface {
	Save(ctx context.Context, code, originalURL string) error
	Retrieve(ctx context.Context, code string) (string, error)
}
