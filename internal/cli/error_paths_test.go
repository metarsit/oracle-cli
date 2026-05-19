// internal/cli/error_paths_test.go
package cli

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/metarsit/oracle-cli/internal/client"
)

// failingOracle returns the given status + envelope on every request.
func failingOracle(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestStatusReturnsAuthErr(t *testing.T) {
	srv := failingOracle(401, `{"error":{"code":"AUTH_FAILED","message":"bad"},"meta":{}}`)
	defer srv.Close()
	t.Setenv("ORACLE_BASE_URL", srv.URL)
	t.Setenv("ORACLE_API_TOKEN", "tok")

	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"status"})
	err := root.Execute()
	var authErr *client.ErrAuth
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *ErrAuth, got %T: %v", err, err)
	}
}

func TestSuggestReturnsNotFound(t *testing.T) {
	srv := failingOracle(404, `{"error":{"code":"NOT_FOUND","message":"x"},"meta":{}}`)
	defer srv.Close()
	t.Setenv("ORACLE_BASE_URL", srv.URL)
	t.Setenv("ORACLE_API_TOKEN", "tok")

	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"suggest", "--asset", "BTC"})
	err := root.Execute()
	var nfErr *client.ErrNotFound
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *ErrNotFound, got %T: %v", err, err)
	}
}

func TestInstrumentsMissingRequiredFlag(t *testing.T) {
	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"instruments"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected required-flag error")
	}
}

func TestReadyAlsoCallsHealth(t *testing.T) {
	// exercise newReadyCmd happy path which earlier showed 11% cov
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"ready":true},"meta":{}}`))
	}))
	defer srv.Close()
	t.Setenv("ORACLE_BASE_URL", srv.URL)
	t.Setenv("ORACLE_OUTPUT", "json")

	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"ready"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ready") {
		t.Errorf("ready output: %s", out.String())
	}
}

func TestDeribitPositionsHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/public/auth"):
			_, _ = w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900}}`))
		case strings.HasSuffix(r.URL.Path, "/private/get_positions"):
			_, _ = w.Write([]byte(`{"result":[{"instrument_name":"BTC-PERPETUAL","size":0.5,"direction":"buy"}]}`))
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
	root.SetArgs([]string{"deribit", "positions", "--currency", "BTC"})
	if err := root.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "BTC-PERPETUAL") {
		t.Errorf("deribit positions out: %s", out.String())
	}
}

func TestClassifyAllErrorTypes(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{nil, ""},
		{&client.ErrAuth{Msg: "a"}, "AUTH_FAILED"},
		{&client.ErrNotFound{Msg: "n"}, "NOT_FOUND"},
		{&client.ErrNetwork{Err: errors.New("net")}, "NETWORK"},
		{&client.ErrDegraded{Msg: "d"}, "DEGRADED"},
		{&client.ErrAPI{Code: "X", Msg: "y"}, "X"},
		{errors.New("plain"), "ERROR"},
	}
	for _, c := range cases {
		got, _ := classify(c.err)
		if got != c.code {
			t.Errorf("classify(%T) = %q, want %q", c.err, got, c.code)
		}
	}
}

func TestPlanAllFailedError(t *testing.T) {
	e := planAllFailed{errs: []planErr{{Endpoint: "x"}, {Endpoint: "y"}}}
	if !strings.Contains(e.Error(), "2 plan calls failed") {
		t.Errorf("error string: %s", e.Error())
	}
}

func TestNilIfUnset(t *testing.T) {
	if v := nilIfUnset(client.Positions{}, false); v != nil {
		t.Errorf("unset should be nil, got %v", v)
	}
	if v := nilIfUnset(client.Positions{}, true); v == nil {
		t.Error("set should be non-nil")
	}
}
