// cmd/oracle/integration_test.go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binPath is set by TestMain after building the CLI binary into a temp dir.
var binPath string

func TestMain(m *testing.M) {
	// flag.Parse so testing.Short() is safe to read here.
	if !flag.Parsed() {
		flag.Parse()
	}
	if testing.Short() {
		// Skip the heavy build when -short; integration tests below also skip.
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "oracle-int-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	binPath = filepath.Join(dir, "oracle_test_bin")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// integrationOracle is a fake oracle that switches handlers by env var so
// individual integration tests can replay specific status codes.
func integrationOracle(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/healthz"):
			_, _ = w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/suggestions/latest"):
			_, _ = w.Write([]byte(`{"data":{"id":1,"asset":"BTC","chosen_expiry":"T+1"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/status"):
			if status != 0 {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"status":"ok"},"meta":{}}`))
		default:
			// plan aggregator hits multiple endpoints; satisfy them minimally
			_, _ = w.Write([]byte(`{"data":{},"meta":{}}`))
		}
	}))
}

func runBinary(t *testing.T, env []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("spawn failed: %v", err)
		}
	}
	return stdout.String(), stderr.String(), code
}

func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test (binary spawn)")
	}
}

func TestIntegrationHealth(t *testing.T) {
	skipIfShort(t)
	srv := integrationOracle(t, 0, "")
	defer srv.Close()
	_, stderr, code := runBinary(t, []string{
		"ORACLE_BASE_URL=" + srv.URL,
		"ORACLE_OUTPUT=json",
	}, "health")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
}

func TestIntegrationSuggestJSONContainsChosenExpiry(t *testing.T) {
	skipIfShort(t)
	srv := integrationOracle(t, 0, "")
	defer srv.Close()
	stdout, stderr, code := runBinary(t, []string{
		"ORACLE_BASE_URL=" + srv.URL,
		"ORACLE_API_TOKEN=tok",
		"ORACLE_OUTPUT=json",
	}, "suggest", "--asset", "BTC", "--output", "json")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
	// envelope wraps data in {"data": ..., "meta": ...}; assert chosen_expiry visible
	if !strings.Contains(stdout, "chosen_expiry") {
		t.Errorf("stdout missing chosen_expiry: %s", stdout)
	}
	// Must parse as JSON
	var dec map[string]any
	if err := json.Unmarshal([]byte(stdout), &dec); err != nil {
		t.Errorf("not valid json: %v\n%s", err, stdout)
	}
}

func TestIntegrationStatusExitCodes(t *testing.T) {
	skipIfShort(t)
	cases := []struct {
		name    string
		status  int
		body    string
		want    int
		wantErr string
	}{
		{"auth_401", 401, `{"error":{"code":"AUTH","message":"bad token"},"meta":{}}`, 2, "auth"},
		{"notfound_404", 404, `{"error":{"code":"NF","message":"x"},"meta":{}}`, 3, "not found"},
		{"degraded_503", 503, `{"error":{"code":"DEG","message":"warm"},"meta":{}}`, 5, "degraded"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := integrationOracle(t, c.status, c.body)
			defer srv.Close()
			_, stderr, code := runBinary(t, []string{
				"ORACLE_BASE_URL=" + srv.URL,
				"ORACLE_API_TOKEN=tok",
			}, "status")
			if code != c.want {
				t.Errorf("exit = %d, want %d\nstderr=%s", code, c.want, stderr)
			}
			if c.wantErr != "" && !strings.Contains(stderr, c.wantErr) {
				t.Errorf("stderr missing %q: %q", c.wantErr, stderr)
			}
		})
	}
}

func TestIntegrationStatusUnreachable(t *testing.T) {
	skipIfShort(t)
	_, _, code := runBinary(t, []string{
		// loopback port no listener
		"ORACLE_BASE_URL=http://127.0.0.1:1",
		"ORACLE_API_TOKEN=tok",
		"ORACLE_TIMEOUT=500ms",
	}, "status")
	if code != 4 {
		t.Errorf("exit = %d, want 4 (network)", code)
	}
}

func TestIntegrationPlanJSONShape(t *testing.T) {
	skipIfShort(t)
	srv := integrationOracle(t, 0, "")
	defer srv.Close()
	stdout, stderr, code := runBinary(t, []string{
		"ORACLE_BASE_URL=" + srv.URL,
		"ORACLE_API_TOKEN=tok",
		"ORACLE_OUTPUT=json",
	}, "plan")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("plan output not json: %v\n%s", err, stdout)
	}
	data, ok := doc["data"].(map[string]any)
	if !ok {
		t.Fatalf("plan.data missing: %v", doc)
	}
	for _, k := range []string{"status", "suggestions", "hedges", "positions"} {
		if _, ok := data[k]; !ok {
			t.Errorf("plan.data missing key %q: %v", k, data)
		}
	}
	if _, ok := doc["meta"]; !ok {
		t.Error("plan.meta missing")
	}
}

func TestIntegrationVersionFlag(t *testing.T) {
	skipIfShort(t)
	stdout, stderr, code := runBinary(t, nil, "version")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
	if stdout == "" {
		t.Error("version printed no output")
	}
}
