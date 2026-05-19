// internal/client/endpoints_test.go
package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newFake(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL, "tok", time.Second)
}

func TestHealthSkipsAuth(t *testing.T) {
	c := newFake(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected auth header on /healthz")
		}
		w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
	}))
	if _, err := c.Health(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestSuggestionLatestPassesQuery(t *testing.T) {
	c := newFake(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("asset"); got != "BTC" {
			t.Errorf("asset = %q", got)
		}
		w.Write([]byte(`{"data":{"id":1,"asset":"BTC","chosen_expiry":"T+1"},"meta":{}}`))
	}))
	got, err := c.SuggestionLatest(t.Context(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset != "BTC" {
		t.Errorf("asset = %q", got.Asset)
	}
}

func TestEngineRunPosts(t *testing.T) {
	c := newFake(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/engine/run") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"data":{"started":true},"meta":{}}`))
	}))
	if err := c.EngineRun(t.Context()); err != nil {
		t.Fatal(err)
	}
}
