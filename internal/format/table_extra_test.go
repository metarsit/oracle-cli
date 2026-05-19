// internal/format/table_extra_test.go
package format

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/metarsit/oracle-cli/internal/client"
)

func TestTableHedge(t *testing.T) {
	h := client.Hedge{
		ID: 1, SettlementID: 2, Side: "buy",
		QtyUSD: "100", MarkPriceAt: "65000",
		ProposedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := NewRenderer("table").Render(&buf, h); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "buy") || !strings.Contains(buf.String(), "settlement_id") {
		t.Errorf("hedge table missing fields: %s", buf.String())
	}
}

func TestTableSettlement(t *testing.T) {
	s := client.Settlement{
		ID: 1, Asset: "BTC",
		ExpiryTS:   time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC),
		SpotSettle: "65500", PutITM: true, CallITM: false,
		ResidualQty: "0", PNLUSD: "12.34",
	}
	var buf bytes.Buffer
	if err := NewRenderer("table").Render(&buf, s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "spot_settle") || !strings.Contains(buf.String(), "BTC") {
		t.Errorf("settlement table missing: %s", buf.String())
	}
}

func TestTablePrice(t *testing.T) {
	p := client.Price{
		Instrument: "BTC-PERPETUAL", Mark: "65000", Index: "65010",
		TS: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := NewRenderer("table").Render(&buf, p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "BTC-PERPETUAL") {
		t.Errorf("price table missing: %s", buf.String())
	}
}

func TestTableBook(t *testing.T) {
	b := client.BookTop{
		Instrument: "BTC-PERPETUAL",
		BidPx:      "65000", BidSz: "1", AskPx: "65001", AskSz: "1",
		TS: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := NewRenderer("table").Render(&buf, b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "bid_px") || !strings.Contains(buf.String(), "ask_px") {
		t.Errorf("book table missing: %s", buf.String())
	}
}

func TestTableInstruments(t *testing.T) {
	exp := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	ins := []client.Instrument{
		{Name: "BTC-PERPETUAL", Base: "BTC", Kind: "perp"},
		{Name: "BTC-1JAN26-65000-C", Base: "BTC", Kind: "option", OptionType: "C", Strike: "65000", ExpiryTS: &exp},
	}
	var buf bytes.Buffer
	if err := NewRenderer("table").Render(&buf, ins); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "BTC-PERPETUAL") || !strings.Contains(buf.String(), "65000") {
		t.Errorf("instruments table missing: %s", buf.String())
	}
}

func TestTableRenderReflectFallback(t *testing.T) {
	// raw map -> renderReflect fallback
	var buf bytes.Buffer
	if err := NewRenderer("table").Render(&buf, map[string]int{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("expected fallback output")
	}
	// nil value path
	var buf2 bytes.Buffer
	if err := NewRenderer("table").Render(&buf2, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf2.String(), "no data") {
		t.Errorf("nil should fall through to (no data): %s", buf2.String())
	}
}

func TestTableRenderError(t *testing.T) {
	var buf bytes.Buffer
	if err := NewRenderer("table").RenderError(&buf, "AUTH_FAILED", "bad token"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "AUTH_FAILED") || !strings.Contains(buf.String(), "bad token") {
		t.Errorf("RenderError missing fields: %s", buf.String())
	}
}
