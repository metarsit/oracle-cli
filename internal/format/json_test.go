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
