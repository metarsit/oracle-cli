# oracle-cli Phase 01 — HTTP Client + Output Formatters

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the oracle HTTP client (envelope decoder, typed errors, bearer middleware, retry) and the json/yaml/table output formatters with golden-file tests.

**Architecture:** `internal/client` wraps `net/http`. Bearer header added by a `RoundTripper` that redacts in errors. Responses go through a generic envelope decoder before unmarshalling into typed structs. `internal/format` exposes a `Renderer` interface with three implementations selected by `--output`.

**Tech Stack:** stdlib `net/http`, `encoding/json`, `gopkg.in/yaml.v3`, `jedib0t/go-pretty/v6`.

**Spec:** `docs/superpowers/specs/2026-05-19-oracle-cli-design.md`

---

## Task 1: Envelope + typed errors

**Files:**
- Create: `internal/client/envelope.go`
- Create: `internal/client/envelope_test.go`
- Create: `internal/client/errors.go`

- [ ] **Step 1: Write failing test**

```go
// internal/client/envelope_test.go
package client

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDecodeData(t *testing.T) {
	body := []byte(`{"data":{"x":1},"meta":{"ts":"2026-01-01T00:00:00Z"}}`)
	var data struct{ X int }
	if err := DecodeEnvelope(body, 200, &data); err != nil {
		t.Fatal(err)
	}
	if data.X != 1 {
		t.Errorf("X = %d, want 1", data.X)
	}
}

func TestDecodeAPIError(t *testing.T) {
	body := []byte(`{"error":{"code":"BAD","message":"nope"},"meta":{"ts":"..."}}`)
	var data struct{}
	err := DecodeEnvelope(body, 400, &data)
	var apiErr *ErrAPI
	if !errors.As(err, &apiErr) {
		t.Fatalf("want ErrAPI, got %T %v", err, err)
	}
	if apiErr.Code != "BAD" || apiErr.Msg != "nope" || apiErr.Status != 400 {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestDecodeAuthError(t *testing.T) {
	body := []byte(`{"error":{"code":"AUTH","message":"x"},"meta":{}}`)
	var data struct{}
	err := DecodeEnvelope(body, 401, &data)
	var authErr *ErrAuth
	if !errors.As(err, &authErr) {
		t.Fatalf("want ErrAuth, got %T %v", err, err)
	}
}

func TestDecodeNotFound(t *testing.T) {
	body := []byte(`{"error":{"code":"NOT_FOUND","message":"x"},"meta":{}}`)
	var data struct{}
	err := DecodeEnvelope(body, 404, &data)
	var nfErr *ErrNotFound
	if !errors.As(err, &nfErr) {
		t.Fatalf("want ErrNotFound, got %T %v", err, err)
	}
}

func TestDecodeDegraded(t *testing.T) {
	body := []byte(`{"error":{"code":"DEGRADED","message":"warmup"},"meta":{}}`)
	var data struct{}
	err := DecodeEnvelope(body, 503, &data)
	var d *ErrDegraded
	if !errors.As(err, &d) {
		t.Fatalf("want ErrDegraded, got %T %v", err, err)
	}
}

func TestDecodeMalformed(t *testing.T) {
	var data struct{}
	if err := DecodeEnvelope([]byte("not-json"), 200, &data); err == nil {
		t.Fatal("expected json error")
	}
}

var _ = json.Marshal // keep import
```

- [ ] **Step 2: Implement errors.go**

```go
// internal/client/errors.go
package client

import "fmt"

// ErrAuth is returned for HTTP 401/403.
type ErrAuth struct{ Status int; Msg string }

func (e *ErrAuth) Error() string { return fmt.Sprintf("auth failed (%d): %s", e.Status, e.Msg) }

// ErrNotFound is returned for HTTP 404.
type ErrNotFound struct{ Msg string }

func (e *ErrNotFound) Error() string { return "not found: " + e.Msg }

// ErrNetwork wraps transport-level failures (dial, EOF, timeout).
type ErrNetwork struct{ Err error }

func (e *ErrNetwork) Error() string { return "network: " + e.Err.Error() }
func (e *ErrNetwork) Unwrap() error { return e.Err }

// ErrDegraded is returned for HTTP 503 or non-ready /readyz payload.
type ErrDegraded struct{ Status int; Msg string }

func (e *ErrDegraded) Error() string { return fmt.Sprintf("degraded (%d): %s", e.Status, e.Msg) }

// ErrAPI is any other error envelope (4xx/5xx with .error).
type ErrAPI struct{ Code, Msg string; Status int }

func (e *ErrAPI) Error() string { return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Msg) }
```

- [ ] **Step 3: Implement envelope.go**

```go
// internal/client/envelope.go
package client

import (
	"encoding/json"
	"fmt"
)

// rawEnvelope mirrors the oracle's wire shape.
type rawEnvelope struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Meta json.RawMessage `json:"meta,omitempty"`
}

// DecodeEnvelope parses body, classifies by status + envelope.error, and
// either unmarshals .data into out or returns a typed error.
func DecodeEnvelope(body []byte, status int, out any) error {
	var env rawEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if env.Error != nil || status >= 400 {
		code, msg := "", ""
		if env.Error != nil {
			code, msg = env.Error.Code, env.Error.Message
		}
		switch {
		case status == 401 || status == 403:
			return &ErrAuth{Status: status, Msg: msg}
		case status == 404:
			return &ErrNotFound{Msg: msg}
		case status == 503:
			return &ErrDegraded{Status: status, Msg: msg}
		}
		return &ErrAPI{Code: code, Msg: msg, Status: status}
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/client/... -v
git add internal/client
git commit -m "feat(client): envelope decoder + typed error taxonomy"
```

---

## Task 2: HTTP client with bearer middleware + retry

**Files:**
- Create: `internal/client/client.go`
- Create: `internal/client/client_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/client/client_test.go
package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetSendsBearer(t *testing.T) {
	gotAuth := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok-abc", time.Second)
	var data struct{ Ok bool }
	if err := c.Get(t.Context(), "/v1/status", nil, &data); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer tok-abc" {
		t.Errorf("auth = %q", gotAuth)
	}
}

func TestGetNetworkErrorRetriesOnce(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		w.Write([]byte(`{"data":{},"meta":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", time.Second)
	var data struct{}
	if err := c.Get(t.Context(), "/x", nil, &data); err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestPostNoRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", time.Second)
	var data struct{}
	err := c.Post(t.Context(), "/x", nil, &data)
	if err == nil {
		t.Fatal("want network error")
	}
	if calls != 1 {
		t.Errorf("posts must not retry, calls = %d", calls)
	}
	var nErr *ErrNetwork
	if !errors.As(err, &nErr) {
		t.Errorf("want ErrNetwork, got %T", err)
	}
}

func TestRedactBearerInError(t *testing.T) {
	c := New("http://127.0.0.1:1", "supersecret", 100*time.Millisecond)
	var data struct{}
	err := c.Get(t.Context(), "/x", nil, &data)
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), "supersecret") {
		t.Errorf("token leaked in error: %v", err)
	}
}

func TestBodyCapEnforced(t *testing.T) {
	huge := strings.Repeat("a", 11*1024*1024) // 11 MiB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":"` + huge + `","meta":{}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "tok", 5*time.Second)
	var data string
	if err := c.Get(t.Context(), "/x", nil, &data); err == nil {
		t.Error("want body-cap error")
	}
}
```

- [ ] **Step 2: Implement client.go**

```go
// internal/client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const maxResponseBytes = 10 * 1024 * 1024

// Client talks to the oracle HTTP API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New builds a client with bearer token and timeout.
func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

// Get issues GET path?query and decodes envelope into out.
func (c *Client) Get(ctx context.Context, path string, query map[string]string, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, out, true)
}

// Post issues POST path with optional JSON body.
func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, out, false)
}

func (c *Client) do(ctx context.Context, method, path string, query map[string]string, body, out any, retry bool) error {
	u, err := buildURL(c.baseURL, path, query)
	if err != nil {
		return err
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		if c.token != "" && path != "/healthz" && path != "/readyz" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if retry && attempt == 0 {
				time.Sleep(250 * time.Millisecond)
				continue
			}
			return &ErrNetwork{Err: redactErr(err, c.token)}
		}
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		_ = resp.Body.Close()
		if readErr != nil {
			return &ErrNetwork{Err: redactErr(readErr, c.token)}
		}
		if len(respBody) > maxResponseBytes {
			return fmt.Errorf("response exceeded %d byte cap", maxResponseBytes)
		}
		return DecodeEnvelope(respBody, resp.StatusCode, out)
	}
	return errors.New("unreachable")
}

func buildURL(base, path string, query map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse base: %w", err)
	}
	u = u.JoinPath(path)
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func redactErr(err error, token string) error {
	if token == "" {
		return err
	}
	msg := err.Error()
	if !contains(msg, token) {
		return err
	}
	return errors.New(replace(msg, token, "REDACTED"))
}

func contains(s, sub string) bool { return len(sub) > 0 && bytes.Contains([]byte(s), []byte(sub)) }
func replace(s, old, new string) string {
	return string(bytes.ReplaceAll([]byte(s), []byte(old), []byte(new)))
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/client/... -v -race
git add internal/client
git commit -m "feat(client): bearer middleware, retry-once GET, body cap, redaction"
```

---

## Task 3: Typed response structs

**Files:**
- Create: `internal/client/types.go`
- Create: `internal/client/types_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/client/types_test.go
package client

import (
	"encoding/json"
	"testing"
)

func TestSuggestionUnmarshal(t *testing.T) {
	b := []byte(`{
		"id":1,"run_at":"2026-05-18T22:33:56+07:00","asset":"BTC",
		"chosen_expiry":"T+1","expiry_ts":"2026-05-19T08:00:00Z",
		"put_strike":74000,"call_strike":78500,
		"put_bid":0.0023,"call_bid":0.0009,
		"net_premium_usd":243.74,"annualised_roi":0.693271,
		"skew_reason":"flat"
	}`)
	var s Suggestion
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	if s.Asset != "BTC" || s.ChosenExpiry != "T+1" || s.SkewReason != "flat" {
		t.Errorf("bad parse: %+v", s)
	}
}

func TestPositionsUnmarshal(t *testing.T) {
	b := []byte(`{"positions":[{"base":"BTC","net_qty":0.5},{"base":"ETH","net_qty":-1.25}]}`)
	var p Positions
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Positions) != 2 || p.Positions[0].Base != "BTC" {
		t.Errorf("bad parse: %+v", p)
	}
}
```

- [ ] **Step 2: Implement types.go**

```go
// internal/client/types.go
package client

import (
	"encoding/json"
	"time"
)

// Instrument is one row of GET /v1/instruments.
type Instrument struct {
	Name        string `json:"name"`
	Base        string `json:"base"`
	Kind        string `json:"kind"`
	OptionType  string `json:"option_type,omitempty"`
	Strike      json.Number `json:"strike,omitempty"`
	ExpiryTS    *time.Time  `json:"expiry_ts,omitempty"`
}

// Price is GET /v1/prices/latest.
type Price struct {
	Instrument string      `json:"instrument"`
	Mark       json.Number `json:"mark"`
	Index      json.Number `json:"index,omitempty"`
	TS         time.Time   `json:"ts"`
}

// BookTop is GET /v1/book/top.
type BookTop struct {
	Instrument string      `json:"instrument"`
	BidPx      json.Number `json:"bid_px"`
	BidSz      json.Number `json:"bid_sz"`
	AskPx      json.Number `json:"ask_px"`
	AskSz      json.Number `json:"ask_sz"`
	TS         time.Time   `json:"ts"`
}

// Suggestion is GET /v1/suggestions/latest.
type Suggestion struct {
	ID            int64       `json:"id"`
	RunAt         time.Time   `json:"run_at"`
	Asset         string      `json:"asset"`
	ChosenExpiry  string      `json:"chosen_expiry"`
	ExpiryTS      *time.Time  `json:"expiry_ts"`
	PutStrike     json.Number `json:"put_strike"`
	CallStrike    json.Number `json:"call_strike"`
	PutBid        json.Number `json:"put_bid"`
	CallBid       json.Number `json:"call_bid"`
	NetPremiumUSD json.Number `json:"net_premium_usd"`
	AnnualisedROI json.Number `json:"annualised_roi"`
	SkewReason    string      `json:"skew_reason"`
}

// Hedge is GET /v1/hedges/latest.
type Hedge struct {
	ID            int64       `json:"id"`
	SettlementID  int64       `json:"settlement_id"`
	Side          string      `json:"side"`
	QtyUSD        json.Number `json:"qty_usd"`
	MarkPriceAt   json.Number `json:"mark_price_at"`
	ProposedAt    time.Time   `json:"proposed_at"`
}

// Settlement is GET /v1/settlements/latest.
type Settlement struct {
	ID          int64       `json:"id"`
	ExpiryTS    time.Time   `json:"expiry_ts"`
	Asset       string      `json:"asset"`
	SpotSettle  json.Number `json:"spot_settle"`
	PutITM      bool        `json:"put_itm"`
	CallITM     bool        `json:"call_itm"`
	ResidualQty json.Number `json:"residual_qty"`
	PNLUSD      json.Number `json:"pnl_usd"`
}

// Position is one row inside Positions.
type Position struct {
	Base   string      `json:"base"`
	NetQty json.Number `json:"net_qty"`
}

// Positions is GET /v1/positions/current.
type Positions struct {
	Positions []Position `json:"positions"`
}

// Status is GET /v1/status — schema is service-defined; keep as raw json.
type Status = json.RawMessage
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/client/... -v
git add internal/client
git commit -m "feat(client): typed response structs (Suggestion, Hedge, Settlement, etc.)"
```

---

## Task 4: Endpoint methods on Client

**Files:**
- Create: `internal/client/endpoints.go`
- Create: `internal/client/endpoints_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/client/endpoints_test.go
package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newFake(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL, "tok", time.Second)
}

func TestHealthSkipsAuth(t *testing.T) {
	c := newFake(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected auth header on /healthz")
		}
		w.Write([]byte(`{"data":{"ok":true},"meta":{}}`))
	}))
	if _, err := c.Health(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestSuggestionLatestPassesQuery(t *testing.T) {
	c := newFake(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("asset"); got != "BTC" {
			t.Errorf("asset = %q", got)
		}
		w.Write([]byte(`{"data":{"id":1,"asset":"BTC","chosen_expiry":"T+1"},"meta":{}}`))
	}))
	got, err := c.SuggestionLatest(t.Context(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset != "BTC" {
		t.Errorf("asset = %q", got.Asset)
	}
}

func TestEngineRunPosts(t *testing.T) {
	c := newFake(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/engine/run") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"data":{"started":true},"meta":{}}`))
	}))
	if err := c.EngineRun(t.Context()); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Implement endpoints.go**

```go
// internal/client/endpoints.go
package client

import (
	"context"
	"encoding/json"
)

// Health returns the raw /healthz envelope payload.
func (c *Client) Health(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	return out, c.Get(ctx, "/healthz", nil, &out)
}

// Ready returns the raw /readyz envelope payload.
func (c *Client) Ready(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	return out, c.Get(ctx, "/readyz", nil, &out)
}

// Status returns the raw /v1/status payload.
func (c *Client) Status(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	return out, c.Get(ctx, "/v1/status", nil, &out)
}

func (c *Client) Instruments(ctx context.Context, base, kind string) ([]Instrument, error) {
	var out []Instrument
	return out, c.Get(ctx, "/v1/instruments", map[string]string{"base": base, "kind": kind}, &out)
}

func (c *Client) PriceLatest(ctx context.Context, instrument string) (Price, error) {
	var out Price
	return out, c.Get(ctx, "/v1/prices/latest", map[string]string{"instrument": instrument}, &out)
}

func (c *Client) BookTop(ctx context.Context, instrument string) (BookTop, error) {
	var out BookTop
	return out, c.Get(ctx, "/v1/book/top", map[string]string{"instrument": instrument}, &out)
}

func (c *Client) SuggestionLatest(ctx context.Context, asset string) (Suggestion, error) {
	var out Suggestion
	return out, c.Get(ctx, "/v1/suggestions/latest", map[string]string{"asset": asset}, &out)
}

func (c *Client) EngineRun(ctx context.Context) error {
	var out json.RawMessage
	return c.Post(ctx, "/v1/engine/run", struct{}{}, &out)
}

func (c *Client) SettlementLatest(ctx context.Context, asset string) (Settlement, error) {
	var out Settlement
	return out, c.Get(ctx, "/v1/settlements/latest", map[string]string{"asset": asset}, &out)
}

func (c *Client) HedgeLatest(ctx context.Context, asset string) (Hedge, error) {
	var out Hedge
	return out, c.Get(ctx, "/v1/hedges/latest", map[string]string{"asset": asset}, &out)
}

func (c *Client) PositionsCurrent(ctx context.Context) (Positions, error) {
	var out Positions
	return out, c.Get(ctx, "/v1/positions/current", nil, &out)
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/client/... -v -race
git add internal/client
git commit -m "feat(client): endpoint methods for all v1 oracle routes"
```

---

## Task 5: Renderer interface + JSON renderer

**Files:**
- Create: `internal/format/format.go`
- Create: `internal/format/json.go`
- Create: `internal/format/json_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/format/json_test.go
package format

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONRendererPretty(t *testing.T) {
	r := NewRenderer("json")
	var buf bytes.Buffer
	if err := r.Render(&buf, map[string]any{"a": 1, "b": "x"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\n  ") {
		t.Errorf("expected indent: %q", buf.String())
	}
}

func TestJSONRendererErrorEnvelope(t *testing.T) {
	r := NewRenderer("json")
	var buf bytes.Buffer
	if err := r.RenderError(&buf, "AUTH_FAILED", "401 unauthorized"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{`"error"`, `"AUTH_FAILED"`, `"meta"`, `"ts"`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %q", want, s)
		}
	}
}
```

- [ ] **Step 2: Implement format.go + json.go**

```go
// internal/format/format.go
package format

import (
	"errors"
	"io"
)

// Renderer turns a typed value (or error) into the chosen output format.
type Renderer interface {
	Render(w io.Writer, v any) error
	RenderError(w io.Writer, code, msg string) error
}

// NewRenderer returns the renderer for the given format name. Unknown
// names fall back to JSON (callers should validate first; this is a safety net).
func NewRenderer(name string) Renderer {
	switch name {
	case "json":
		return jsonRenderer{}
	case "yaml":
		return yamlRenderer{}
	case "table":
		return tableRenderer{}
	}
	return jsonRenderer{}
}

// ErrUnknownFormat is returned by ValidateFormat for invalid names.
var ErrUnknownFormat = errors.New("unknown output format")

// ValidateFormat reports whether name is one of json/yaml/table.
func ValidateFormat(name string) error {
	switch name {
	case "json", "yaml", "table":
		return nil
	}
	return ErrUnknownFormat
}
```

```go
// internal/format/json.go
package format

import (
	"encoding/json"
	"io"
	"time"
)

type jsonRenderer struct{}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Meta struct {
		TS string `json:"ts"`
	} `json:"meta"`
}

func (jsonRenderer) Render(w io.Writer, v any) error {
	env := struct {
		Data any `json:"data"`
		Meta struct {
			TS string `json:"ts"`
		} `json:"meta"`
	}{Data: v}
	env.Meta.TS = time.Now().UTC().Format(time.RFC3339Nano)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

func (jsonRenderer) RenderError(w io.Writer, code, msg string) error {
	var env errorEnvelope
	env.Error.Code = code
	env.Error.Message = msg
	env.Meta.TS = time.Now().UTC().Format(time.RFC3339Nano)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/format/... -v
git add internal/format
git commit -m "feat(format): renderer interface + JSON implementation"
```

---

## Task 6: YAML renderer

**Files:**
- Create: `internal/format/yaml.go`
- Create: `internal/format/yaml_test.go`

- [ ] **Step 1: Add dep + write failing test**

```bash
go get gopkg.in/yaml.v3@latest
```

```go
// internal/format/yaml_test.go
package format

import (
	"bytes"
	"strings"
	"testing"
)

func TestYAMLRender(t *testing.T) {
	r := NewRenderer("yaml")
	var buf bytes.Buffer
	if err := r.Render(&buf, map[string]any{"a": 1, "b": "x"}); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "data:") || !strings.Contains(s, "a: 1") {
		t.Errorf("bad yaml: %q", s)
	}
}
```

- [ ] **Step 2: Implement yaml.go**

```go
// internal/format/yaml.go
package format

import (
	"io"
	"time"

	"gopkg.in/yaml.v3"
)

type yamlRenderer struct{}

func (yamlRenderer) Render(w io.Writer, v any) error {
	env := map[string]any{
		"data": v,
		"meta": map[string]string{"ts": time.Now().UTC().Format(time.RFC3339Nano)},
	}
	return yaml.NewEncoder(w).Encode(env)
}

func (yamlRenderer) RenderError(w io.Writer, code, msg string) error {
	env := map[string]any{
		"error": map[string]string{"code": code, "message": msg},
		"meta":  map[string]string{"ts": time.Now().UTC().Format(time.RFC3339Nano)},
	}
	return yaml.NewEncoder(w).Encode(env)
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/format/... -v
git add internal/format go.mod go.sum
git commit -m "feat(format): YAML renderer"
```

---

## Task 7: Table renderer + golden files

**Files:**
- Create: `internal/format/table.go`
- Create: `internal/format/table_test.go`
- Create: `internal/format/testdata/suggestion.golden`
- Create: `internal/format/testdata/positions.golden`

- [ ] **Step 1: Add dep + write failing test**

```bash
go get github.com/jedib0t/go-pretty/v6/table@latest
```

```go
// internal/format/table_test.go
package format

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metarsit/oracle-cli/internal/client"
)

var update = flag.Bool("update", false, "update golden files")

func TestTableSuggestion(t *testing.T) {
	s := client.Suggestion{
		ID: 1, Asset: "BTC", ChosenExpiry: "T+1",
		PutStrike: "74000", CallStrike: "78500",
		PutBid: "0.0023", CallBid: "0.0009",
		NetPremiumUSD: "243.74", AnnualisedROI: "0.693271",
		SkewReason: "flat",
	}
	r := NewRenderer("table")
	var buf bytes.Buffer
	if err := r.Render(&buf, s); err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "suggestion.golden", buf.Bytes())
}

func TestTablePositions(t *testing.T) {
	p := client.Positions{Positions: []client.Position{
		{Base: "BTC", NetQty: "0.5"},
		{Base: "ETH", NetQty: "-1.25"},
	}}
	r := NewRenderer("table")
	var buf bytes.Buffer
	if err := r.Render(&buf, p); err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "positions.golden", buf.Bytes())
}

func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update)", name, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch %s:\n--- got ---\n%s\n--- want ---\n%s", name, strings.TrimSpace(string(got)), strings.TrimSpace(string(want)))
	}
}
```

- [ ] **Step 2: Implement table.go**

```go
// internal/format/table.go
package format

import (
	"fmt"
	"io"
	"reflect"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/metarsit/oracle-cli/internal/client"
)

type tableRenderer struct{}

func (tableRenderer) Render(w io.Writer, v any) error {
	switch val := v.(type) {
	case client.Suggestion:
		return renderSuggestion(w, val)
	case client.Positions:
		return renderPositions(w, val)
	case client.Hedge:
		return renderHedge(w, val)
	case client.Settlement:
		return renderSettlement(w, val)
	case client.Price:
		return renderPrice(w, val)
	case client.BookTop:
		return renderBook(w, val)
	case []client.Instrument:
		return renderInstruments(w, val)
	}
	return renderReflect(w, v)
}

func (tableRenderer) RenderError(w io.Writer, code, msg string) error {
	_, err := fmt.Fprintf(w, "ERROR %s: %s\n", code, msg)
	return err
}

func newTable(w io.Writer) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleLight)
	return t
}

func renderSuggestion(w io.Writer, s client.Suggestion) error {
	t := newTable(w)
	t.AppendHeader(table.Row{"field", "value"})
	t.AppendRows([]table.Row{
		{"asset", s.Asset}, {"chosen_expiry", s.ChosenExpiry},
		{"put_strike", s.PutStrike}, {"call_strike", s.CallStrike},
		{"put_bid", s.PutBid}, {"call_bid", s.CallBid},
		{"net_premium_usd", s.NetPremiumUSD}, {"annualised_roi", s.AnnualisedROI},
		{"skew_reason", s.SkewReason},
	})
	t.Render()
	return nil
}

func renderPositions(w io.Writer, p client.Positions) error {
	t := newTable(w)
	t.AppendHeader(table.Row{"base", "net_qty"})
	for _, row := range p.Positions {
		t.AppendRow(table.Row{row.Base, row.NetQty})
	}
	t.Render()
	return nil
}

func renderHedge(w io.Writer, h client.Hedge) error {
	t := newTable(w)
	t.AppendHeader(table.Row{"field", "value"})
	t.AppendRows([]table.Row{
		{"id", h.ID}, {"settlement_id", h.SettlementID},
		{"side", h.Side}, {"qty_usd", h.QtyUSD},
		{"mark_price_at", h.MarkPriceAt}, {"proposed_at", h.ProposedAt},
	})
	t.Render()
	return nil
}

func renderSettlement(w io.Writer, s client.Settlement) error {
	t := newTable(w)
	t.AppendHeader(table.Row{"field", "value"})
	t.AppendRows([]table.Row{
		{"id", s.ID}, {"asset", s.Asset}, {"expiry_ts", s.ExpiryTS},
		{"spot_settle", s.SpotSettle}, {"put_itm", s.PutITM}, {"call_itm", s.CallITM},
		{"residual_qty", s.ResidualQty}, {"pnl_usd", s.PNLUSD},
	})
	t.Render()
	return nil
}

func renderPrice(w io.Writer, p client.Price) error {
	t := newTable(w)
	t.AppendHeader(table.Row{"field", "value"})
	t.AppendRows([]table.Row{
		{"instrument", p.Instrument}, {"mark", p.Mark}, {"index", p.Index}, {"ts", p.TS},
	})
	t.Render()
	return nil
}

func renderBook(w io.Writer, b client.BookTop) error {
	t := newTable(w)
	t.AppendHeader(table.Row{"field", "value"})
	t.AppendRows([]table.Row{
		{"instrument", b.Instrument},
		{"bid_px", b.BidPx}, {"bid_sz", b.BidSz},
		{"ask_px", b.AskPx}, {"ask_sz", b.AskSz},
		{"ts", b.TS},
	})
	t.Render()
	return nil
}

func renderInstruments(w io.Writer, ins []client.Instrument) error {
	t := newTable(w)
	t.AppendHeader(table.Row{"name", "base", "kind", "option_type", "strike", "expiry_ts"})
	for _, i := range ins {
		exp := ""
		if i.ExpiryTS != nil {
			exp = i.ExpiryTS.UTC().Format("2006-01-02T15:04Z")
		}
		t.AppendRow(table.Row{i.Name, i.Base, i.Kind, i.OptionType, i.Strike, exp})
	}
	t.Render()
	return nil
}

// renderReflect is the fallback for raw maps / raw JSON.
func renderReflect(w io.Writer, v any) error {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		_, err := fmt.Fprintln(w, "(no data)")
		return err
	}
	_, err := fmt.Fprintf(w, "%+v\n", v)
	return err
}
```

- [ ] **Step 3: Generate goldens + verify**

```bash
go test ./internal/format/... -update
go test ./internal/format/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/format go.mod go.sum
git commit -m "feat(format): table renderer with golden-file tests"
```

---

## Phase 01 Done — proceed to Phase 02 (`2026-05-19-oracle-cli-02-commands-release.md`).
