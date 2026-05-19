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
		Use:   "plan",
		Short: "Aggregated snapshot for Hermes (status+suggest+hedge+positions, JSON-only)",
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

const planTotalCalls = 6 // status + 2 sugg + 2 hedge + positions

func runPlan(ctx context.Context, out io.Writer, c *client.Client) error {
	start := time.Now()
	var (
		mu     sync.Mutex
		status json.RawMessage
		sugMap = map[string]any{"BTC": nil, "ETH": nil}
		hedMap = map[string]any{"BTC": nil, "ETH": nil}
		pos    client.Positions
		posSet bool
		errs   []planErr
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
			"ts":          time.Now().UTC().Format(time.RFC3339Nano),
			"duration_ms": time.Since(start).Milliseconds(),
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
	if len(errs) >= planTotalCalls {
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
