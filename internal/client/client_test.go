// internal/client/client_test.go
package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetSendsBearer(t *testing.T) {
	gotAuth := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok-abc", time.Second)
	var data struct{ Ok bool }
	if err := c.Get(t.Context(), "/v1/status", nil, &data); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer tok-abc" {
		t.Errorf("auth = %q", gotAuth)
	}
}

func TestGetNetworkErrorRetriesOnce(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		w.Write([]byte(`{"data":{},"meta":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", time.Second)
	var data struct{}
	if err := c.Get(t.Context(), "/x", nil, &data); err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

func TestPostNoRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", time.Second)
	var data struct{}
	err := c.Post(t.Context(), "/x", nil, &data)
	if err == nil {
		t.Fatal("want network error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("posts must not retry, calls = %d", got)
	}
	var nErr *ErrNetwork
	if !errors.As(err, &nErr) {
		t.Errorf("want ErrNetwork, got %T", err)
	}
}

func TestRedactBearerInError(t *testing.T) {
	c := New("http://127.0.0.1:1", "supersecret", 100*time.Millisecond)
	var data struct{}
	err := c.Get(t.Context(), "/x", nil, &data)
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), "supersecret") {
		t.Errorf("token leaked in error: %v", err)
	}
}

func TestBodyCapEnforced(t *testing.T) {
	huge := strings.Repeat("a", 11*1024*1024) // 11 MiB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":"` + huge + `","meta":{}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "tok", 5*time.Second)
	var data string
	if err := c.Get(t.Context(), "/x", nil, &data); err == nil {
		t.Error("want body-cap error")
	}
}
