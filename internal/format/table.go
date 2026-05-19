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
