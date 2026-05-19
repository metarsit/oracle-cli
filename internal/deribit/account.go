// Package deribit provides an OAuth2 client for the Deribit API.
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
