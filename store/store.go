package store

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("short url not found")
	ErrExists   = errors.New("short url already exists")
)

type Store interface {
	// Create guarda el mapping solo si el code no existe; si existe devuelve ErrExists.
	Create(ctx context.Context, code, originalURL string) error
	// Retrieve devuelve la URL original o ErrNotFound.
	Retrieve(ctx context.Context, code string) (string, error)
	// Ping verifica que el backend esté disponible.
	Ping(ctx context.Context) error
}
