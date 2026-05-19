// internal/cli/commands_test.go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeOracle returns canned envelopes per path.
func fakeOracle(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/v1/status"):
			_, _ = w.Write([]byte(`{"data":{"status":"ok"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/instruments"):
			_, _ = w.Write([]byte(`{"data":[{"name":"BTC-PERPETUAL","base":"BTC","kind":"perp"}],"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/prices/latest"):
			_, _ = w.Write([]byte(`{"data":{"instrument":"BTC-PERPETUAL","mark":"65000","ts":"2026-01-01T00:00:00Z"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/book/top"):
			_, _ = w.Write([]byte(`{"data":{"instrument":"BTC-PERPETUAL","bid_px":"65000","bid_sz":"1","ask_px":"65001","ask_sz":"1","ts":"2026-01-01T00:00:00Z"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/suggestions/latest"):
			_, _ = w.Write([]byte(`{"data":{"id":1,"asset":"BTC","chosen_expiry":"T+1"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/engine/run"):
			_, _ = w.Write([]byte(`{"data":{"started":true},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/settlements/latest"):
			_, _ = w.Write([]byte(`{"data":{"id":1,"asset":"BTC","expiry_ts":"2026-01-01T08:00:00Z","spot_settle":"65500"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/hedges/latest"):
			_, _ = w.Write([]byte(`{"data":{"id":1,"settlement_id":1,"side":"buy","qty_usd":"100","mark_price_at":"65000","proposed_at":"2026-01-01T00:00:00Z"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/positions/current"):
			_, _ = w.Write([]byte(`{"data":{"positions":[{"base":"BTC","net_qty":"0.5"}]},"meta":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func runCmd(t *testing.T, args ...string) string {
	t.Helper()
	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, out.String())
	}
	return out.String()
}

func TestEveryReadCommandJSON(t *testing.T) {
	srv := fakeOracle(t)
	defer srv.Close()
	t.Setenv("ORACLE_BASE_URL", srv.URL)
	t.Setenv("ORACLE_API_TOKEN", "tok")
	t.Setenv("ORACLE_OUTPUT", "json")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"status"}, `"status"`},
		{[]string{"instruments", "--base", "BTC"}, `"BTC-PERPETUAL"`},
		{[]string{"price", "--instrument", "BTC-PERPETUAL"}, `"BTC-PERPETUAL"`},
		{[]string{"book", "--instrument", "BTC-PERPETUAL"}, `"bid_px"`},
		{[]string{"suggest", "--asset", "BTC"}, `"chosen_expiry"`},
		{[]string{"engine", "run"}, `"started"`},
		{[]string{"settle", "--asset", "BTC"}, `"spot_settle"`},
		{[]string{"hedge", "--asset", "BTC"}, `"side"`},
		{[]string{"positions"}, `"BTC"`},
	}
	for _, c := range cases {
		c := c
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			out := runCmd(t, c.args...)
			if !strings.Contains(out, c.want) {
				t.Errorf("missing %q in:\n%s", c.want, out)
			}
		})
	}
}
