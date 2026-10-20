package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RobertoPatino1/url_shortener/store"
)

func doRequest(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// sequence devuelve un generador que recorre los códigos en bucle.
func sequence(codes ...string) func() (string, error) {
	i := 0
	return func() (string, error) {
		c := codes[i%len(codes)]
		i++
		return c, nil
	}
}

type failingStore struct{}

func (failingStore) Create(context.Context, string, string) error { return errors.New("boom") }
func (failingStore) Retrieve(context.Context, string) (string, error) {
	return "", errors.New("boom")
}
func (failingStore) Ping(context.Context) error { return errors.New("boom") }

func TestShortenAndRedirect(t *testing.T) {
	h := New(store.NewMemoryStore(), "http://short.test/").Routes()

	rec := doRequest(h, http.MethodPost, "/shorten", `{"url":"https://go.dev"}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp shortenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Code, codeLength)
	assert.Equal(t, "http://short.test/"+resp.Code, resp.ShortURL)

	rec = doRequest(h, http.MethodGet, "/"+resp.Code, "")
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "https://go.dev", rec.Header().Get("Location"))
}

func TestRedirectNotFound(t *testing.T) {
	h := New(store.NewMemoryStore(), "http://short.test").Routes()
	rec := doRequest(h, http.MethodGet, "/noexiste", "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestShortenInvalidInput(t *testing.T) {
	h := New(store.NewMemoryStore(), "http://short.test").Routes()

	tests := []struct{ name, body string }{
		{"malformed json", `{"url":`},
		{"empty body", ``},
		{"missing url", `{}`},
		{"javascript scheme", `{"url":"javascript:alert(1)"}`},
		{"ftp scheme", `{"url":"ftp://example.com/file"}`},
		{"no host", `{"url":"https://"}`},
		{"not a url", `{"url":"hola"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(h, http.MethodPost, "/shorten", tc.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestShortenRetriesOnCollision(t *testing.T) {
	st := store.NewMemoryStore()
	require.NoError(t, st.Create(context.Background(), "aaa", "https://existing.test"))

	srv := New(st, "http://short.test")
	srv.newCode = sequence("aaa", "aaa", "bbb") // 2 colisiones, luego éxito

	rec := doRequest(srv.Routes(), http.MethodPost, "/shorten", `{"url":"https://go.dev"}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp shortenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "bbb", resp.Code)

	got, err := st.Retrieve(context.Background(), "aaa")
	require.NoError(t, err)
	assert.Equal(t, "https://existing.test", got, "el link original no debe sobrescribirse")
}

func TestShortenGivesUpAfterMaxAttempts(t *testing.T) {
	st := store.NewMemoryStore()
	require.NoError(t, st.Create(context.Background(), "aaa", "https://existing.test"))

	srv := New(st, "http://short.test")
	srv.newCode = sequence("aaa") // siempre colisiona

	rec := doRequest(srv.Routes(), http.MethodPost, "/shorten", `{"url":"https://go.dev"}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestStoreFailuresReturn500(t *testing.T) {
	h := New(failingStore{}, "http://short.test").Routes()

	rec := doRequest(h, http.MethodPost, "/shorten", `{"url":"https://go.dev"}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	rec = doRequest(h, http.MethodGet, "/abc", "")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHealth(t *testing.T) {
	rec := doRequest(New(store.NewMemoryStore(), "http://short.test").Routes(), http.MethodGet, "/healthz", "")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(New(failingStore{}, "http://short.test").Routes(), http.MethodGet, "/healthz", "")
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
