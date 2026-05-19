package client

import (
	"encoding/json"
	"time"
)

// Instrument is one row of GET /v1/instruments.
type Instrument struct {
	Name       string      `json:"name"`
	Base       string      `json:"base"`
	Kind       string      `json:"kind"`
	OptionType string      `json:"option_type,omitempty"`
	Strike     json.Number `json:"strike,omitempty"`
	ExpiryTS   *time.Time  `json:"expiry_ts,omitempty"`
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
	ID           int64       `json:"id"`
	SettlementID int64       `json:"settlement_id"`
	Side         string      `json:"side"`
	QtyUSD       json.Number `json:"qty_usd"`
	MarkPriceAt  json.Number `json:"mark_price_at"`
	ProposedAt   time.Time   `json:"proposed_at"`
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
