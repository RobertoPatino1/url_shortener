package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RobertoPatino1/url_shortener/server"
	"github.com/RobertoPatino1/url_shortener/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := getenv("ADDR", ":8080")
	baseURL := getenv("BASE_URL", "http://localhost:8080")

	st, closeStore, err := newStore(ctx)
	if err != nil {
		return err
	}
	defer closeStore()

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           server.New(st, baseURL).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s (base url %s)", addr, baseURL)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		log.Println("shutting down...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx)
}

// newStore usa Redis si REDIS_ADDR está definido; si no, cae a memoria.
func newStore(ctx context.Context) (store.Store, func(), error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Println("REDIS_ADDR not set: using in-memory store (data is lost on restart)")
		return store.NewMemoryStore(), func() {}, nil
	}

	ttl, err := parseTTL(os.Getenv("LINK_TTL"))
	if err != nil {
		return nil, nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rs, err := store.NewRedisStore(pingCtx, redisAddr, os.Getenv("REDIS_PASSWORD"), ttl)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("using redis at %s (link ttl: %v, 0 = no expiry)", redisAddr, ttl)
	return rs, func() { _ = rs.Close() }, nil
}

func parseTTL(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid LINK_TTL %q: %w", raw, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("LINK_TTL must be >= 0, got %v", d)
	}
	return d, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
