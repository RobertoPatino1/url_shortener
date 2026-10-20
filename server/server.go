package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RobertoPatino1/url_shortener/shortener"
	"github.com/RobertoPatino1/url_shortener/store"
)

const (
	codeLength  = 7
	maxAttempts = 3
)

var errNoUniqueCode = errors.New("could not allocate a unique code")

type Server struct {
	store   store.Store
	baseURL string
	newCode func() (string, error) // inyectable para tests
}

func New(s store.Store, baseURL string) *Server {
	return &Server{
		store:   s,
		baseURL: strings.TrimRight(baseURL, "/"),
		newCode: func() (string, error) { return shortener.NewCode(codeLength) },
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /shorten", s.handleShorten)
	mux.HandleFunc("GET /{code}", s.handleRedirect)
	return mux
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		log.Printf("health: %v", err)
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
		return
	}
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleShorten(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if err := validateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code, err := s.createWithRetry(r.Context(), req.URL)
	if err != nil {
		log.Printf("shorten: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(shortenResponse{
		Code:     code,
		ShortURL: s.baseURL + "/" + code,
	})
}

// createWithRetry reintenta con un código nuevo solo si hubo colisión.
func (s *Server) createWithRetry(ctx context.Context, original string) (string, error) {
	for range maxAttempts {
		code, err := s.newCode()
		if err != nil {
			return "", fmt.Errorf("generate code: %w", err)
		}
		err = s.store.Create(ctx, code, original)
		if err == nil {
			return code, nil
		}
		if !errors.Is(err, store.ErrExists) {
			return "", err
		}
	}
	return "", errNoUniqueCode
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	original, err := s.store.Retrieve(r.Context(), r.PathValue("code"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("retrieve: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, original, http.StatusFound) // 302: no se cachea, permite analytics futuro
}

func validateURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return errors.New("invalid url")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("url must be http or https with a host")
	}
	return nil
}
