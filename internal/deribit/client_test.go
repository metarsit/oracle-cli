// internal/deribit/client_test.go
package deribit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthFailureReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"code":13004,"message":"invalid_credentials"}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "cid", "wrong", time.Second)
	_, err := c.AccountSummary(context.Background(), "BTC")
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), "invalid_credentials") {
		t.Errorf("err = %v", err)
	}
}

func TestAuthRequiresCredentials(t *testing.T) {
	c := New("http://unused", "", "", time.Second)
	_, err := c.AccountSummary(context.Background(), "BTC")
	if err == nil {
		t.Fatal("expected missing-credentials error")
	}
	if !strings.Contains(err.Error(), "client_id") {
		t.Errorf("err = %v", err)
	}
}

func TestPrivateGet401ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/public/auth"):
			_, _ = w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900}}`))
		default:
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`unauthorized`))
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "cid", "csec", time.Second)
	_, err := c.AccountSummary(context.Background(), "BTC")
	if err == nil {
		t.Fatal("expected 401 wrapped error")
	}
	if !strings.Contains(err.Error(), "deribit auth failed") {
		t.Errorf("err = %v", err)
	}
}

func TestPositionsCurrencyPassedAsQuery(t *testing.T) {
	gotCurrency := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/public/auth"):
			_, _ = w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900}}`))
		case strings.HasSuffix(r.URL.Path, "/private/get_positions"):
			gotCurrency = r.URL.Query().Get("currency")
			_, _ = w.Write([]byte(`{"result":[]}`))
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "cid", "csec", time.Second)
	pos, err := c.Positions(context.Background(), "ETH")
	if err != nil {
		t.Fatal(err)
	}
	if gotCurrency != "ETH" {
		t.Errorf("currency query = %q, want ETH", gotCurrency)
	}
	if len(pos) != 0 {
		t.Errorf("empty positions expected, got %d", len(pos))
	}
}

func TestPositionsMalformedResultErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/public/auth"):
			_, _ = w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900}}`))
		case strings.HasSuffix(r.URL.Path, "/private/get_positions"):
			// Result is a string instead of a list — second-level Unmarshal must error.
			_, _ = w.Write([]byte(`{"result":"not-a-list"}`))
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "cid", "csec", time.Second)
	_, err := c.Positions(context.Background(), "BTC")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestAuthCachesToken(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/public/auth"):
			calls++
			_, _ = w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900,"token_type":"bearer"}}`))
		case strings.HasSuffix(r.URL.Path, "/private/get_account_summary"):
			if r.Header.Get("Authorization") != "Bearer abc" {
				t.Errorf("auth = %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"result":{"currency":"BTC","equity":1.5,"available_funds":1.0}}`))
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "cid", "csec", time.Second)
	if _, err := c.AccountSummary(context.Background(), "BTC"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AccountSummary(context.Background(), "BTC"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("auth calls = %d, want 1 (cached)", calls)
	}
}
