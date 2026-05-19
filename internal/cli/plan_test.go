// internal/cli/plan_test.go
package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlanAggregatorJSON(t *testing.T) {
	srv := fakeOracle(t)
	defer srv.Close()
	t.Setenv("ORACLE_BASE_URL", srv.URL)
	t.Setenv("ORACLE_API_TOKEN", "tok")

	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan"})
	if err := root.Execute(); err != nil {
		t.Fatalf("plan: %v\n%s", err, out.String())
	}

	var parsed struct {
		Data struct {
			Status      json.RawMessage            `json:"status"`
			Suggestions map[string]json.RawMessage `json:"suggestions"`
			Hedges      map[string]json.RawMessage `json:"hedges"`
			Positions   json.RawMessage            `json:"positions"`
		} `json:"data"`
		Meta struct {
			DurationMS int `json:"duration_ms"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("decode: %v\n%s", err, out.String())
	}
	if len(parsed.Data.Status) == 0 {
		t.Error("missing status")
	}
	if _, ok := parsed.Data.Suggestions["BTC"]; !ok {
		t.Error("missing BTC suggestion")
	}
	if _, ok := parsed.Data.Hedges["ETH"]; !ok {
		t.Error("missing ETH hedge")
	}
}

func TestPlanPartialFailureExits0(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "suggestions") && r.URL.Query().Get("asset") == "ETH" {
			http.Error(w, `{"error":{"code":"NOT_FOUND","message":"x"},"meta":{}}`, 404)
			return
		}
		_, _ = w.Write([]byte(`{"data":{},"meta":{}}`))
	}))
	defer srv.Close()
	t.Setenv("ORACLE_BASE_URL", srv.URL)
	t.Setenv("ORACLE_API_TOKEN", "tok")

	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan"})
	if err := root.Execute(); err != nil {
		t.Fatalf("plan should succeed on partial failure: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"errors"`) {
		t.Errorf("missing errors array: %s", out.String())
	}
}
