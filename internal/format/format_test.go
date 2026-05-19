// internal/format/format_test.go
package format

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestValidateFormat(t *testing.T) {
	cases := []struct {
		name    string
		wantErr error
	}{
		{"json", nil},
		{"yaml", nil},
		{"table", nil},
		{"xml", ErrUnknownFormat},
		{"", ErrUnknownFormat},
		{"JSON", ErrUnknownFormat}, // case-sensitive
		{" json", ErrUnknownFormat},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateFormat(c.name)
			if c.wantErr == nil && err != nil {
				t.Errorf("ValidateFormat(%q) = %v, want nil", c.name, err)
			}
			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Errorf("ValidateFormat(%q) = %v, want %v", c.name, err, c.wantErr)
			}
		})
	}
}

func TestNewRendererFallsBackToJSON(t *testing.T) {
	// Document the safety-net behaviour: unknown name -> JSON renderer.
	var buf bytes.Buffer
	if err := NewRenderer("xml").Render(&buf, map[string]int{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"a": 1`) {
		t.Errorf("expected JSON output, got %q", buf.String())
	}
}

func TestYAMLRenderError(t *testing.T) {
	var buf bytes.Buffer
	if err := (yamlRenderer{}).RenderError(&buf, "BAD", "nope"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "error:") || !strings.Contains(s, "BAD") || !strings.Contains(s, "nope") {
		t.Errorf("yaml render-error missing fields: %q", s)
	}
	if !strings.Contains(s, "meta:") {
		t.Errorf("yaml render-error missing meta block: %q", s)
	}
}
