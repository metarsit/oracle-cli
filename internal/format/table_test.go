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
