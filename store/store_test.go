package store

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMemStore(t *testing.T) Store {
	t.Helper()
	return NewMemoryStore()
}

func newRedisStore(t *testing.T) Store {
	t.Helper()
	mr := miniredis.RunT(t)
	s, err := NewRedisStore(context.Background(), mr.Addr(), "", time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// Contract test: ambas implementaciones deben comportarse igual.
func TestStoreContract(t *testing.T) {
	impls := map[string]func(*testing.T) Store{
		"memory": newMemStore,
		"redis":  newRedisStore,
	}

	for name, newStore := range impls {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("create and retrieve", func(t *testing.T) {
				s := newStore(t)
				require.NoError(t, s.Create(ctx, "abc", "https://go.dev"))

				got, err := s.Retrieve(ctx, "abc")
				require.NoError(t, err)
				assert.Equal(t, "https://go.dev", got)
			})

			t.Run("retrieve missing", func(t *testing.T) {
				s := newStore(t)
				_, err := s.Retrieve(ctx, "nope")
				assert.ErrorIs(t, err, ErrNotFound)
			})

			t.Run("duplicate create keeps original", func(t *testing.T) {
				s := newStore(t)
				require.NoError(t, s.Create(ctx, "abc", "https://first.test"))

				err := s.Create(ctx, "abc", "https://second.test")
				assert.ErrorIs(t, err, ErrExists)

				got, err := s.Retrieve(ctx, "abc")
				require.NoError(t, err)
				assert.Equal(t, "https://first.test", got)
			})

			t.Run("concurrent create has one winner", func(t *testing.T) {
				s := newStore(t)

				var wins atomic.Int32
				var wg sync.WaitGroup
				for i := 0; i < 20; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						if err := s.Create(ctx, "race", "https://example.com"); err == nil {
							wins.Add(1)
						}
					}()
				}
				wg.Wait()

				assert.Equal(t, int32(1), wins.Load())
			})

			t.Run("ping", func(t *testing.T) {
				assert.NoError(t, newStore(t).Ping(ctx))
			})
		})
	}
}

func TestRedisStoreTTLExpires(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	s, err := NewRedisStore(ctx, mr.Addr(), "", time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	require.NoError(t, s.Create(ctx, "abc", "https://go.dev"))
	mr.FastForward(2 * time.Minute)

	_, err = s.Retrieve(ctx, "abc")
	assert.ErrorIs(t, err, ErrNotFound)
}
