package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/RobertoPatino1/url_shortener/shortener"
	"github.com/RobertoPatino1/url_shortener/store"
)

const codeLength = 7

type Server struct {
	store   store.Store
	baseURL string
}

func New(s store.Store, baseURL string) *Server {
	return &Server{store: s, baseURL: baseURL}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
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

	code, err := shortener.NewCode(codeLength)
	if err != nil {
		log.Printf("generate code: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.Save(r.Context(), code, req.URL); err != nil {
		log.Printf("save: %v", err)
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
	http.Redirect(w, r, original, http.StatusFound) // 302
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
