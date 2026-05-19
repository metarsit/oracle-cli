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
