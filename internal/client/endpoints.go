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

// Instruments lists tradeable instruments filtered by base currency and kind.
func (c *Client) Instruments(ctx context.Context, base, kind string) ([]Instrument, error) {
	var out []Instrument
	return out, c.Get(ctx, "/v1/instruments", map[string]string{"base": base, "kind": kind}, &out)
}

// PriceLatest returns the most recent price for the given instrument.
func (c *Client) PriceLatest(ctx context.Context, instrument string) (Price, error) {
	var out Price
	return out, c.Get(ctx, "/v1/prices/latest", map[string]string{"instrument": instrument}, &out)
}

// BookTop returns the top-of-book quote for the given instrument.
func (c *Client) BookTop(ctx context.Context, instrument string) (BookTop, error) {
	var out BookTop
	return out, c.Get(ctx, "/v1/book/top", map[string]string{"instrument": instrument}, &out)
}

// SuggestionLatest returns the most recent hedge suggestion for the given asset.
func (c *Client) SuggestionLatest(ctx context.Context, asset string) (Suggestion, error) {
	var out Suggestion
	return out, c.Get(ctx, "/v1/suggestions/latest", map[string]string{"asset": asset}, &out)
}

// EngineRun triggers a pricing engine cycle via POST /v1/engine/run.
func (c *Client) EngineRun(ctx context.Context) error {
	var out json.RawMessage
	return c.Post(ctx, "/v1/engine/run", struct{}{}, &out)
}

// SettlementLatest returns the most recent settlement record for the given asset.
func (c *Client) SettlementLatest(ctx context.Context, asset string) (Settlement, error) {
	var out Settlement
	return out, c.Get(ctx, "/v1/settlements/latest", map[string]string{"asset": asset}, &out)
}

// HedgeLatest returns the most recent hedge record for the given asset.
func (c *Client) HedgeLatest(ctx context.Context, asset string) (Hedge, error) {
	var out Hedge
	return out, c.Get(ctx, "/v1/hedges/latest", map[string]string{"asset": asset}, &out)
}

// PositionsCurrent returns the current open positions.
func (c *Client) PositionsCurrent(ctx context.Context) (Positions, error) {
	var out Positions
	return out, c.Get(ctx, "/v1/positions/current", nil, &out)
}
