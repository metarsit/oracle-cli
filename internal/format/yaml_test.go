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
