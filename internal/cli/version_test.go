// internal/cli/version_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	root := NewRootCmd("test-version")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "test-version") {
		t.Errorf("got %q, want substring 'test-version'", out.String())
	}
}
