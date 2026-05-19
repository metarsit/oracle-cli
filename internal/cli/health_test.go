// internal/cli/health_test.go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthCommandJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
	}))
	defer srv.Close()
	t.Setenv("ORACLE_BASE_URL", srv.URL)
	t.Setenv("ORACLE_OUTPUT", "json")

	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"health"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"ok"`) {
		t.Errorf("out = %q", out.String())
	}
}
