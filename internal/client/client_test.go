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
		_, _ = w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		_, _ = w.Write([]byte(`{"data":{},"meta":{}}`))
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		_, _ = w.Write([]byte(`{"data":"` + huge + `","meta":{}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "tok", 5*time.Second)
	var data string
	if err := c.Get(t.Context(), "/x", nil, &data); err == nil {
		t.Error("want body-cap error")
	}
}

// TestBodyCapBoundaryExceeded — exactly one byte over the cap must error.
func TestBodyCapBoundaryExceeded(t *testing.T) {
	// craft a body that is exactly maxResponseBytes + 1
	const target = maxResponseBytes + 1
	prefix := `{"data":"`
	suffix := `","meta":{}}`
	pad := target - len(prefix) - len(suffix)
	if pad < 0 {
		t.Fatalf("envelope wrappers larger than cap (%d)", pad)
	}
	huge := strings.Repeat("a", pad)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(prefix + huge + suffix))
	}))
	defer srv.Close()
	c := New(srv.URL, "tok", 5*time.Second)
	var data string
	err := c.Get(t.Context(), "/x", nil, &data)
	if err == nil {
		t.Fatal("want cap error at +1 byte")
	}
	if !strings.Contains(err.Error(), "exceeded") {
		t.Errorf("error should mention cap: %v", err)
	}
}

// TestBodyCapBoundaryAccepted — exactly maxResponseBytes succeeds.
func TestBodyCapBoundaryAccepted(t *testing.T) {
	const target = maxResponseBytes
	prefix := `{"data":"`
	suffix := `","meta":{}}`
	pad := target - len(prefix) - len(suffix)
	if pad < 0 {
		t.Fatalf("envelope wrappers larger than cap (%d)", pad)
	}
	fill := strings.Repeat("a", pad)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(prefix + fill + suffix))
	}))
	defer srv.Close()
	c := New(srv.URL, "tok", 30*time.Second)
	var data string
	if err := c.Get(t.Context(), "/x", nil, &data); err != nil {
		t.Fatalf("exact-cap body should succeed: %v", err)
	}
	if len(data) != pad {
		t.Errorf("decoded data len = %d, want %d", len(data), pad)
	}
}

// TestBuildURLBadBase — propagates url.Parse error to the caller.
func TestBuildURLBadBase(t *testing.T) {
	c := New("://bad", "tok", time.Second)
	var data struct{}
	err := c.Get(t.Context(), "/x", nil, &data)
	if err == nil {
		t.Fatal("want parse error")
	}
	if !strings.Contains(err.Error(), "parse base") {
		t.Errorf("error should wrap parse base: %v", err)
	}
}

// TestPostMarshalFailure — channels cannot be JSON-marshalled.
func TestPostMarshalFailure(t *testing.T) {
	c := New("http://127.0.0.1:1", "", time.Second)
	body := make(chan int)
	var out struct{}
	err := c.Post(t.Context(), "/x", body, &out)
	if err == nil {
		t.Fatal("want marshal error")
	}
	if !strings.Contains(err.Error(), "marshal body") {
		t.Errorf("error should wrap marshal body: %v", err)
	}
}

// TestRedactReplacesTokenInError — token appearing in error string is replaced
// with REDACTED via the internal replace helper. Constructed via a server URL
// that embeds the token in the host so a network error surfaces it.
func TestRedactReplacesTokenInError(t *testing.T) {
	const token = "secrettokenABC"
	// Force url.Parse to fail by giving an unreachable host containing the
	// token. The retry path falls into ErrNetwork wrapping; redactErr then
	// triggers the replace branch only when the underlying error text
	// contains the token. We craft a request whose http client error
	// message will mention the host (containing the token).
	c := New("http://"+token+".invalid.nonexistent.test:1", token, 100*time.Millisecond)
	var data struct{}
	err := c.Get(t.Context(), "/x", nil, &data)
	if err == nil {
		t.Fatal("want network error")
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("token must be redacted, got %v", err)
	}
	if !strings.Contains(err.Error(), "REDACTED") {
		t.Errorf("expected REDACTED marker, got %v", err)
	}
}

// TestServer500MapsToAPI — generic 5xx (not 503) becomes ErrAPI, not ErrDegraded.
func TestServer500MapsToAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL","message":"boom"},"meta":{}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "tok", time.Second)
	var data struct{}
	err := c.Get(t.Context(), "/x", nil, &data)
	var apiErr *ErrAPI
	if !errors.As(err, &apiErr) {
		t.Fatalf("want ErrAPI, got %T %v", err, err)
	}
	var deg *ErrDegraded
	if errors.As(err, &deg) {
		t.Errorf("500 must not classify as ErrDegraded")
	}
	if apiErr.Status != 500 || apiErr.Code != "INTERNAL" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}
