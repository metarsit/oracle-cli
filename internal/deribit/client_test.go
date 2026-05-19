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
