// internal/cli/deribit_test.go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeribitBalanceJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/public/auth"):
			_, _ = w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900}}`))
		case strings.HasSuffix(r.URL.Path, "/private/get_account_summary"):
			_, _ = w.Write([]byte(`{"result":{"currency":"BTC","equity":1.5,"available_funds":1.0}}`))
		}
	}))
	defer srv.Close()
	t.Setenv("DERIBIT_BASE_URL", srv.URL)
	t.Setenv("DERIBIT_CLIENT_ID", "cid")
	t.Setenv("DERIBIT_CLIENT_SECRET", "csec")
	t.Setenv("ORACLE_OUTPUT", "json")

	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"deribit", "balance", "--currency", "BTC"})
	if err := root.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"equity"`) {
		t.Errorf("out = %s", out.String())
	}
}
