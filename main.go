package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/RobertoPatino1/url_shortener/server"
	"github.com/RobertoPatino1/url_shortener/store"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	addr := getenv("ADDR", ":8080")
	baseURL := getenv("BASE_URL", "http://localhost:8080")

	srv := server.New(store.NewMemoryStore(), baseURL)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	log.Fatal(httpSrv.ListenAndServe())
}
