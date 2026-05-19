# oracle-cli Phase 02 — Subcommands, Deribit, Release

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire all oracle subcommands, the `plan` aggregator, read-only Deribit OAuth2 client + commands, exit-code mapping, README, and release artifacts.

**Architecture:** Each oracle command is a thin shell: parse flags → resolve config (calling vault if needed) → instantiate `client.Client` → call endpoint → render. `plan` fans out via `errgroup`. Deribit uses a separate `internal/deribit` package with its own OAuth2 token cache. `main.go` maps typed errors to exit codes.

**Tech Stack:** stdlib `golang.org/x/sync/errgroup`, stdlib net/http (Deribit), gh-cli for release.

**Spec:** `docs/superpowers/specs/2026-05-19-oracle-cli-design.md`
**Depends on:** Phase 00 and Phase 01 complete.

---

## Task 1: Shared command bootstrap (config resolution helper)

**Files:**
- Create: `internal/cli/bootstrap.go`
- Create: `internal/cli/bootstrap_test.go`
- Modify: `internal/cli/root.go` (add global flags)

- [ ] **Step 1: Write failing test**

```go
// internal/cli/bootstrap_test.go
package cli

import (
	"testing"
)

func TestBootstrapEnvOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ORACLE_BASE_URL", "https://env.example")
	t.Setenv("ORACLE_API_TOKEN", "env-tok")

	root := NewRootCmd("t")
	root.SetArgs([]string{"status"}) // pick any cmd so PersistentFlags parse
	// Don't execute; just resolve flags + bootstrap on the chosen cmd.
	if err := root.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	cfg, err := bootstrap(root, false)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if cfg.BaseURL != "https://env.example" || cfg.Token != "env-tok" {
		t.Errorf("bad cfg: %+v", cfg)
	}
}
```

- [ ] **Step 2: Implement bootstrap.go**

```go
// internal/cli/bootstrap.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/config"
	"github.com/metarsit/oracle-cli/internal/vault"
	"github.com/spf13/cobra"
)

type resolvedConfig struct {
	config.Resolved
}

// bootstrap turns the parsed cobra command + global flags into a Resolved
// config. needsAuth=true triggers lazy vault open when the token is missing
// from flag+env.
func bootstrap(cmd *cobra.Command, needsAuth bool) (resolvedConfig, error) {
	flags := config.Flags{}
	flags.BaseURL, _ = cmd.Flags().GetString("base-url")
	flags.Token, _ = cmd.Flags().GetString("token")
	flags.Output, _ = cmd.Flags().GetString("output")
	flags.Timeout, _ = cmd.Flags().GetString("timeout")

	file, err := config.LoadFile(configPath(cmd))
	if err != nil {
		return resolvedConfig{}, err
	}

	vaultMap := map[string]string{}
	if needsAuth {
		preview := config.Resolve(config.Inputs{Flag: flags, File: file, Vault: nil})
		if preview.Token == "" || preview.DeribitClientSecret == "" {
			pw, err := vault.ReadPassphrase()
			if err == nil {
				if v, vErr := vault.Open(vaultPath(cmd), pw); vErr == nil {
					vaultMap = v.Secrets
				}
			}
		}
	}

	r := config.Resolve(config.Inputs{Flag: flags, File: file, Vault: vaultMap})
	return resolvedConfig{Resolved: r}, nil
}
```

- [ ] **Step 3: Modify root.go to add global flags**

```go
// internal/cli/root.go
package cli

import "github.com/spf13/cobra"

func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "oracle",
		Short:         "Deribit Oracle CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().String("base-url", "", "Oracle base URL (env ORACLE_BASE_URL)")
	cmd.PersistentFlags().String("token", "", "Bearer token (env ORACLE_API_TOKEN; prefer env or vault)")
	cmd.PersistentFlags().String("output", "", "Output format: json|table|yaml (env ORACLE_OUTPUT)")
	cmd.PersistentFlags().String("timeout", "", "HTTP timeout (e.g. 10s)")
	cmd.PersistentFlags().String("config", "", "Path to config.toml")
	cmd.PersistentFlags().String("vault", "", "Path to secrets.vault")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable debug logging on stderr")
	cmd.AddCommand(
		newVersionCmd(version),
		newVaultCmd(),
		newConfigCmd(),
		newHealthCmd(),
		newReadyCmd(),
		newStatusCmd(),
		newInstrumentsCmd(),
		newPriceCmd(),
		newBookCmd(),
		newSuggestCmd(),
		newEngineCmd(),
		newSettleCmd(),
		newHedgeCmd(),
		newPositionsCmd(),
		newPlanCmd(),
		newDeribitCmd(),
	)
	return cmd
}
```

- [ ] **Step 4: Verify + commit**

```bash
go build ./...
git add internal/cli
git commit -m "feat(cli): shared bootstrap + global flags on root"
```

---

## Task 2: `oracle health` and `oracle ready`

**Files:**
- Create: `internal/cli/health.go`
- Create: `internal/cli/ready.go`
- Create: `internal/cli/health_test.go`

- [ ] **Step 1: Write failing test**

```go
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
		w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
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
```

- [ ] **Step 2: Implement health.go + ready.go**

```go
// internal/cli/health.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use: "health", Short: "GET /healthz (no auth)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, false)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, "", cfg.Timeout)
			data, err := c.Health(cmd.Context())
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
}
```

```go
// internal/cli/ready.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newReadyCmd() *cobra.Command {
	return &cobra.Command{
		Use: "ready", Short: "GET /readyz (no auth)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, false)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, "", cfg.Timeout)
			data, err := c.Ready(cmd.Context())
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/cli/... -run TestHealth -v
git add internal/cli
git commit -m "feat(cli): health + ready subcommands"
```

---

## Task 3: Remaining oracle subcommands (status, instruments, price, book, suggest, engine, settle, hedge, positions)

**Files:**
- Create: `internal/cli/status.go`
- Create: `internal/cli/instruments.go`
- Create: `internal/cli/price.go`
- Create: `internal/cli/book.go`
- Create: `internal/cli/suggest.go`
- Create: `internal/cli/engine.go`
- Create: `internal/cli/settle.go`
- Create: `internal/cli/hedge.go`
- Create: `internal/cli/positions.go`
- Create: `internal/cli/commands_test.go`

- [ ] **Step 1: Write failing matrix test**

```go
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
			w.Write([]byte(`{"data":{"status":"ok"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/instruments"):
			w.Write([]byte(`{"data":[{"name":"BTC-PERPETUAL","base":"BTC","kind":"perp"}],"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/prices/latest"):
			w.Write([]byte(`{"data":{"instrument":"BTC-PERPETUAL","mark":"65000","ts":"2026-01-01T00:00:00Z"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/book/top"):
			w.Write([]byte(`{"data":{"instrument":"BTC-PERPETUAL","bid_px":"65000","bid_sz":"1","ask_px":"65001","ask_sz":"1","ts":"2026-01-01T00:00:00Z"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/suggestions/latest"):
			w.Write([]byte(`{"data":{"id":1,"asset":"BTC","chosen_expiry":"T+1"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/engine/run"):
			w.Write([]byte(`{"data":{"started":true},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/settlements/latest"):
			w.Write([]byte(`{"data":{"id":1,"asset":"BTC","expiry_ts":"2026-01-01T08:00:00Z","spot_settle":"65500"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/hedges/latest"):
			w.Write([]byte(`{"data":{"id":1,"settlement_id":1,"side":"buy","qty_usd":"100","mark_price_at":"65000","proposed_at":"2026-01-01T00:00:00Z"},"meta":{}}`))
		case strings.HasSuffix(r.URL.Path, "/v1/positions/current"):
			w.Write([]byte(`{"data":{"positions":[{"base":"BTC","net_qty":"0.5"}]},"meta":{}}`))
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
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			out := runCmd(t, c.args...)
			if !strings.Contains(out, c.want) {
				t.Errorf("missing %q in:\n%s", c.want, out)
			}
		})
	}
}
```

- [ ] **Step 2: Implement each command file**

Each follows the same pattern. Provide full files:

```go
// internal/cli/status.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use: "status", Short: "GET /v1/status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.Status(cmd.Context())
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
}
```

```go
// internal/cli/instruments.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newInstrumentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "instruments", Short: "GET /v1/instruments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			base, _ := cmd.Flags().GetString("base")
			kind, _ := cmd.Flags().GetString("kind")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.Instruments(cmd.Context(), base, kind)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("base", "", "BTC|ETH")
	cmd.Flags().String("kind", "", "option|perp|future")
	_ = cmd.MarkFlagRequired("base")
	return cmd
}
```

```go
// internal/cli/price.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newPriceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "price", Short: "GET /v1/prices/latest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			inst, _ := cmd.Flags().GetString("instrument")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.PriceLatest(cmd.Context(), inst)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("instrument", "", "instrument name e.g. BTC-PERPETUAL")
	_ = cmd.MarkFlagRequired("instrument")
	return cmd
}
```

```go
// internal/cli/book.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newBookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "book", Short: "GET /v1/book/top",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			inst, _ := cmd.Flags().GetString("instrument")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.BookTop(cmd.Context(), inst)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("instrument", "", "instrument name")
	_ = cmd.MarkFlagRequired("instrument")
	return cmd
}
```

```go
// internal/cli/suggest.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newSuggestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "suggest", Short: "GET /v1/suggestions/latest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			asset, _ := cmd.Flags().GetString("asset")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.SuggestionLatest(cmd.Context(), asset)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("asset", "", "BTC|ETH")
	_ = cmd.MarkFlagRequired("asset")
	return cmd
}
```

```go
// internal/cli/engine.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newEngineCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "engine", Short: "Engine controls"}
	cmd.AddCommand(&cobra.Command{
		Use: "run", Short: "POST /v1/engine/run",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			if err := c.EngineRun(cmd.Context()); err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), map[string]bool{"started": true})
		},
	})
	return cmd
}
```

```go
// internal/cli/settle.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newSettleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "settle", Short: "GET /v1/settlements/latest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			asset, _ := cmd.Flags().GetString("asset")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.SettlementLatest(cmd.Context(), asset)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("asset", "", "BTC|ETH")
	_ = cmd.MarkFlagRequired("asset")
	return cmd
}
```

```go
// internal/cli/hedge.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newHedgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "hedge", Short: "GET /v1/hedges/latest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			asset, _ := cmd.Flags().GetString("asset")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.HedgeLatest(cmd.Context(), asset)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("asset", "", "BTC|ETH")
	_ = cmd.MarkFlagRequired("asset")
	return cmd
}
```

```go
// internal/cli/positions.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newPositionsCmd() *cobra.Command {
	return &cobra.Command{
		Use: "positions", Short: "GET /v1/positions/current",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.PositionsCurrent(cmd.Context())
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/cli/... -v -race
git add internal/cli
git commit -m "feat(cli): status/instruments/price/book/suggest/engine/settle/hedge/positions"
```

---

## Task 4: `oracle plan` aggregator

**Files:**
- Create: `internal/cli/plan.go`
- Create: `internal/cli/plan_test.go`

- [ ] **Step 1: Add errgroup dep + write failing test**

```bash
go get golang.org/x/sync/errgroup@latest
```

```go
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
		w.Write([]byte(`{"data":{},"meta":{}}`))
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
```

- [ ] **Step 2: Implement plan.go**

```go
// internal/cli/plan.go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newPlanCmd() *cobra.Command {
	return &cobra.Command{
		Use: "plan", Short: "Aggregated snapshot for Hermes (status+suggest+hedge+positions, JSON-only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			return runPlan(cmd.Context(), cmd.OutOrStdout(), c)
		},
	}
}

type planErr struct {
	Endpoint string `json:"endpoint"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

func runPlan(ctx context.Context, out io.Writer, c *client.Client) error {
	start := time.Now()
	var (
		mu      sync.Mutex
		status  json.RawMessage
		sugMap  = map[string]any{"BTC": nil, "ETH": nil}
		hedMap  = map[string]any{"BTC": nil, "ETH": nil}
		pos     client.Positions
		posSet  bool
		errs    []planErr
	)
	record := func(endpoint string, err error) {
		mu.Lock()
		defer mu.Unlock()
		code, msg := classify(err)
		errs = append(errs, planErr{Endpoint: endpoint, Code: code, Message: msg})
	}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		s, err := c.Status(gctx)
		if err != nil {
			record("status", err)
			return nil
		}
		mu.Lock()
		status = s
		mu.Unlock()
		return nil
	})
	for _, asset := range []string{"BTC", "ETH"} {
		asset := asset
		g.Go(func() error {
			s, err := c.SuggestionLatest(gctx, asset)
			if err != nil {
				record("suggest:"+asset, err)
				return nil
			}
			mu.Lock()
			sugMap[asset] = s
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			h, err := c.HedgeLatest(gctx, asset)
			if err != nil {
				record("hedge:"+asset, err)
				return nil
			}
			mu.Lock()
			hedMap[asset] = h
			mu.Unlock()
			return nil
		})
	}
	g.Go(func() error {
		p, err := c.PositionsCurrent(gctx)
		if err != nil {
			record("positions", err)
			return nil
		}
		mu.Lock()
		pos = p
		posSet = true
		mu.Unlock()
		return nil
	})
	_ = g.Wait()

	doc := map[string]any{
		"data": map[string]any{
			"status":      status,
			"suggestions": sugMap,
			"hedges":      hedMap,
			"positions":   nilIfUnset(pos, posSet),
		},
		"errors": errs,
		"meta": map[string]any{
			"ts":           time.Now().UTC().Format(time.RFC3339Nano),
			"duration_ms":  time.Since(start).Milliseconds(),
		},
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if _, err := out.Write(append(b, '\n')); err != nil {
		return err
	}

	// Exit-code semantics per spec §6 enforced by main.go via classify of
	// the highest-severity uniform failure. Returning nil here = exit 0;
	// `oracle plan` always succeeds when at least one call succeeded.
	if len(errs) >= 8 { // all six calls failed
		return planAllFailed{errs: errs}
	}
	return nil
}

type planAllFailed struct{ errs []planErr }

func (p planAllFailed) Error() string { return fmt.Sprintf("all %d plan calls failed", len(p.errs)) }

func nilIfUnset(p client.Positions, set bool) any {
	if !set {
		return nil
	}
	return p
}

// classify returns (code, message) for a client error.
func classify(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	type coder interface{ Error() string }
	_ = coder(nil)
	// stable codes used by spec
	switch e := err.(type) {
	case *client.ErrAuth:
		return "AUTH_FAILED", e.Msg
	case *client.ErrNotFound:
		return "NOT_FOUND", e.Msg
	case *client.ErrNetwork:
		return "NETWORK", e.Err.Error()
	case *client.ErrDegraded:
		return "DEGRADED", e.Msg
	case *client.ErrAPI:
		return e.Code, e.Msg
	}
	return "ERROR", err.Error()
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/cli/... -run TestPlan -v -race
git add internal/cli go.mod go.sum
git commit -m "feat(cli): plan aggregator with concurrent fan-out"
```

---

## Task 5: Deribit OAuth2 client + read-only endpoints

**Files:**
- Create: `internal/deribit/client.go`
- Create: `internal/deribit/client_test.go`
- Create: `internal/deribit/account.go`

- [ ] **Step 1: Write failing test**

```go
// internal/deribit/client_test.go
package deribit

import (
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
			w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900,"token_type":"bearer"}}`))
		case strings.HasSuffix(r.URL.Path, "/private/get_account_summary"):
			if r.Header.Get("Authorization") != "Bearer abc" {
				t.Errorf("auth = %q", r.Header.Get("Authorization"))
			}
			w.Write([]byte(`{"result":{"currency":"BTC","equity":1.5,"available_funds":1.0}}`))
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "cid", "csec", time.Second)
	if _, err := c.AccountSummary(t.Context(), "BTC"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AccountSummary(t.Context(), "BTC"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("auth calls = %d, want 1 (cached)", calls)
	}
}
```

- [ ] **Step 2: Implement client.go**

```go
// internal/deribit/client.go
package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Client wraps the Deribit OAuth2 client_credentials flow.
type Client struct {
	baseURL  string
	clientID string
	secret   string
	http     *http.Client

	mu        sync.Mutex
	token     string
	tokenExp  time.Time
}

// New returns an OAuth2 client. Pass the API base URL (typically
// https://www.deribit.com/api/v2 or the testnet equivalent).
func New(baseURL, clientID, secret string, timeout time.Duration) *Client {
	return &Client{
		baseURL:  baseURL,
		clientID: clientID,
		secret:   secret,
		http:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) authToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	if c.clientID == "" || c.secret == "" {
		return "", errors.New("deribit: client_id and client_secret required")
	}
	q := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.secret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/public/auth?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		Result struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("auth parse: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("deribit auth error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}
	c.token = parsed.Result.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(parsed.Result.ExpiresIn-30) * time.Second)
	return c.token, nil
}

func (c *Client) privateGet(ctx context.Context, path string, query url.Values, out any) error {
	tok, err := c.authToken(ctx)
	if err != nil {
		return err
	}
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("deribit get: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("deribit auth failed: %s", string(body))
	}
	return json.Unmarshal(body, out)
}
```

- [ ] **Step 3: Implement account.go**

```go
// internal/deribit/account.go
package deribit

import (
	"context"
	"encoding/json"
	"net/url"
)

// AccountSummary mirrors /private/get_account_summary result.
type AccountSummary struct {
	Currency       string  `json:"currency"`
	Equity         float64 `json:"equity"`
	AvailableFunds float64 `json:"available_funds"`
}

// AccountSummary returns the account snapshot for currency (BTC|ETH|USDC).
func (c *Client) AccountSummary(ctx context.Context, currency string) (AccountSummary, error) {
	var wrap struct {
		Result AccountSummary `json:"result"`
	}
	q := url.Values{"currency": {currency}}
	if err := c.privateGet(ctx, "/private/get_account_summary", q, &wrap); err != nil {
		return AccountSummary{}, err
	}
	return wrap.Result, nil
}

// Position mirrors a row of /private/get_positions result.
type Position struct {
	InstrumentName string  `json:"instrument_name"`
	Size           float64 `json:"size"`
	Direction      string  `json:"direction"`
	AveragePrice   float64 `json:"average_price"`
	MarkPrice      float64 `json:"mark_price"`
}

// Positions returns open positions for currency.
func (c *Client) Positions(ctx context.Context, currency string) ([]Position, error) {
	var wrap struct {
		Result json.RawMessage `json:"result"`
	}
	q := url.Values{"currency": {currency}}
	if err := c.privateGet(ctx, "/private/get_positions", q, &wrap); err != nil {
		return nil, err
	}
	var out []Position
	if err := json.Unmarshal(wrap.Result, &out); err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/deribit/... -v -race
git add internal/deribit
git commit -m "feat(deribit): OAuth2 client + account_summary + positions"
```

---

## Task 6: `oracle deribit` subcommand group

**Files:**
- Create: `internal/cli/deribit.go`
- Create: `internal/cli/deribit_test.go`

- [ ] **Step 1: Write failing test**

```go
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
			w.Write([]byte(`{"result":{"access_token":"abc","expires_in":900}}`))
		case strings.HasSuffix(r.URL.Path, "/private/get_account_summary"):
			w.Write([]byte(`{"result":{"currency":"BTC","equity":1.5,"available_funds":1.0}}`))
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
```

- [ ] **Step 2: Implement deribit.go**

```go
// internal/cli/deribit.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/deribit"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newDeribitCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "deribit", Short: "Read-only Deribit queries (separate creds)"}
	cmd.AddCommand(newDeribitBalanceCmd(), newDeribitPositionsCmd())
	return cmd
}

func newDeribitBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "balance", Short: "GET /private/get_account_summary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			currency, _ := cmd.Flags().GetString("currency")
			c := deribit.New(cfg.DeribitBaseURL, cfg.DeribitClientID, cfg.DeribitClientSecret, cfg.Timeout)
			data, err := c.AccountSummary(cmd.Context(), currency)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("currency", "BTC", "BTC|ETH|USDC")
	return cmd
}

func newDeribitPositionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "positions", Short: "GET /private/get_positions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			currency, _ := cmd.Flags().GetString("currency")
			c := deribit.New(cfg.DeribitBaseURL, cfg.DeribitClientID, cfg.DeribitClientSecret, cfg.Timeout)
			data, err := c.Positions(cmd.Context(), currency)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("currency", "BTC", "BTC|ETH|USDC")
	return cmd
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/cli/... -run TestDeribit -v -race
git add internal/cli
git commit -m "feat(cli): deribit balance + positions subcommands"
```

---

## Task 7: Exit-code mapping in `main.go`

**Files:**
- Modify: `cmd/oracle/main.go`
- Create: `cmd/oracle/main_test.go`

- [ ] **Step 1: Write failing test**

```go
// cmd/oracle/main_test.go
package main

import (
	"errors"
	"testing"

	"github.com/metarsit/oracle-cli/internal/client"
)

func TestExitCodeMapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{&client.ErrAuth{Status: 401, Msg: ""}, 2},
		{&client.ErrNotFound{Msg: ""}, 3},
		{&client.ErrNetwork{Err: errors.New("x")}, 4},
		{&client.ErrDegraded{Status: 503, Msg: ""}, 5},
		{errors.New("anything"), 1},
	}
	for _, c := range cases {
		if got := exitCode(c.err); got != c.want {
			t.Errorf("exitCode(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Modify main.go**

```go
// cmd/oracle/main.go
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/cli"
	"github.com/metarsit/oracle-cli/internal/client"
)

var version = "dev"

func main() {
	err := cli.NewRootCmd(version).Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(exitCode(err))
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var authErr *client.ErrAuth
	if errors.As(err, &authErr) {
		return 2
	}
	var nfErr *client.ErrNotFound
	if errors.As(err, &nfErr) {
		return 3
	}
	var netErr *client.ErrNetwork
	if errors.As(err, &netErr) {
		return 4
	}
	var degErr *client.ErrDegraded
	if errors.As(err, &degErr) {
		return 5
	}
	return 1
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./cmd/oracle/... -v
git add cmd/oracle
git commit -m "feat(cli): exit-code mapping per spec"
```

---

## Task 8: README + LICENSE + CHANGELOG + SECURITY

**Files:**
- Create: `README.md`
- Create: `LICENSE`
- Create: `CHANGELOG.md`
- Create: `SECURITY.md`

- [ ] **Step 1: Write README.md**

```markdown
# oracle-cli

Command-line client for the Deribit Oracle HTTP API. Used directly by humans
and as a subprocess by the Hermes Agent.

## Install

```bash
go install github.com/metarsit/oracle-cli/cmd/oracle@latest
```

Or grab a binary from [Releases](https://github.com/metarsit/oracle-cli/releases).

## Quickstart

```bash
# One-time
oracle vault init                       # prompts for vault passphrase
oracle vault set oracle_api_token "<token-from-oracle-deploy>"
oracle config set base_url "https://oracle.up.railway.app"

# Daily use
oracle suggest --asset BTC
oracle plan                             # JSON aggregate for Hermes
oracle deribit balance --currency BTC
```

## Configuration

Resolution order: flag → env → vault → `~/.config/oracle-cli/config.toml` → default.

| Key                     | Env                       | Default                                    |
|-------------------------|---------------------------|--------------------------------------------|
| `base_url`              | `ORACLE_BASE_URL`         | `http://localhost:8080`                    |
| `oracle_api_token`      | `ORACLE_API_TOKEN`        | _(vault or env only)_                      |
| `deribit_client_id`     | `DERIBIT_CLIENT_ID`       | _(vault or env only)_                      |
| `deribit_client_secret` | `DERIBIT_CLIENT_SECRET`   | _(vault or env only)_                      |
| `deribit_base_url`      | `DERIBIT_BASE_URL`        | `https://www.deribit.com/api/v2`           |
| `output`                | `ORACLE_OUTPUT`           | `table`                                    |
| `timeout`               | `ORACLE_TIMEOUT`          | `10s`                                      |

**Secrets never persist to `config.toml`.** Use `oracle vault set` instead.

## Exit codes

| Code | Meaning                                                    |
|------|------------------------------------------------------------|
| 0    | success                                                    |
| 1    | generic error                                              |
| 2    | auth failed (HTTP 401/403)                                 |
| 3    | not found (HTTP 404)                                       |
| 4    | oracle unreachable (network)                               |
| 5    | oracle degraded (HTTP 503 / non-ready)                     |

## Hermes integration

```bash
ORACLE_VAULT_PASSPHRASE=$VAULT_PW \
  oracle plan --output json > /tmp/oracle-plan.json
```

Stdout = single JSON document. Stderr = logs / progress / errors.

## Vault security

AES-256-GCM with argon2id KDF (time=3, memory=64 MiB, threads=4). Fresh nonce
per write. Refuses to operate when file mode is looser than `0600` or the
parent directory is world-writable. See `SECURITY.md` for the full threat model.
```

- [ ] **Step 2: Write LICENSE (MIT)**

```text
MIT License

Copyright (c) 2026 Nicholas Metarsit

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 3: Write CHANGELOG.md**

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-19
### Added
- Initial release: oracle wrapping subcommands, `plan` aggregator, encrypted vault,
  read-only Deribit OAuth2 client, json/yaml/table output, exit-code mapping.
```

- [ ] **Step 4: Write SECURITY.md**

```markdown
# Security policy

## Reporting a vulnerability

Email `nicholas@alcoholnic.com`. Please do not file public issues for security
problems.

## Vault threat model

In scope:
- Attacker reads `secrets.vault` from a stolen laptop, leaked backup, or
  accidental git commit. Brute-force must be infeasible.

Out of scope (v0.1.0):
- In-process memory disclosure (cleared key in memory).
- Root-on-the-box, malicious modified binary, side-channel attacks.

Crypto:
- AES-256-GCM with random 12-byte nonce per write.
- argon2id KDF: time=3, memory=64 MiB, threads=4, 16-byte salt.
- Header binds AAD → tamper-evident.

The CLI refuses to operate when:
- vault file mode is looser than `0600`.
- the parent directory is world-writable.
```

- [ ] **Step 5: Commit**

```bash
git add README.md LICENSE CHANGELOG.md SECURITY.md
git commit -m "docs: README, LICENSE (MIT), CHANGELOG, SECURITY"
```

---

## Task 9: Release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write workflow**

```yaml
# .github/workflows/release.yml
name: release
on:
  push:
    tags: ["v*"]
jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - name: build matrix
        run: make release
      - name: release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          gh release create "$GITHUB_REF_NAME" \
            --title "$GITHUB_REF_NAME" \
            --generate-notes \
            dist/*
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: GitHub Release workflow for tag v*"
```

---

## Task 10: Final verification

- [ ] **Step 1: Full local sweep**

```bash
make fmt
make vet
make lint || true   # may warn; address blockers
go test -race -coverprofile=cover.out ./...
go tool cover -func cover.out | tail -1
```

Expected: total coverage ≥ 80%.

- [ ] **Step 2: Manual smoke test**

```bash
make build
./bin/oracle version
./bin/oracle vault init
ORACLE_VAULT_PASSPHRASE=test ./bin/oracle vault set oracle_api_token testtok
./bin/oracle config set base_url http://localhost:8080
./bin/oracle config show
```

- [ ] **Step 3: Tag + push**

```bash
git tag v0.1.0
# git push origin main --tags   # actual push deferred to user
```

---

## Phase 02 Done — v0.1.0 ready to tag.
