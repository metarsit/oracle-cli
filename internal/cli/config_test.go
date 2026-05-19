// internal/cli/config_test.go
package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSetShowGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgPath := filepath.Join(dir, "oracle-cli", "config.toml")

	run := func(args ...string) (string, error) {
		root := NewRootCmd("t")
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		err := root.Execute()
		return out.String(), err
	}

	if _, err := run("config", "set", "base_url", "https://x"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := run("config", "get", "base_url")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(got) != "https://x" {
		t.Errorf("got %q want https://x", got)
	}
	got, _ = run("config", "show")
	if !strings.Contains(got, "https://x") {
		t.Errorf("show missing base_url: %q", got)
	}
	if _, err := run("config", "set", "oracle_api_token", "leak"); err == nil {
		t.Error("expected refusal to store secret in config")
	}
	_ = cfgPath
}
