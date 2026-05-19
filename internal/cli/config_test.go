// internal/cli/config_test.go
package cli

import (
	"bytes"
	"os"
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

// driveCmd executes and returns output captured *after* execution. Mirrors
// the runCmd helper in commands_test.go but propagates the error so callers
// can assert on failure modes.
func driveCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd("t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestConfigSetAndGetAllValidKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cases := []struct {
		key, val string
	}{
		{"base_url", "https://oracle.example.com"},
		{"deribit_base_url", "https://test.deribit.com/api/v2"},
		{"output", "json"},
		{"timeout", "15s"},
	}
	for _, c := range cases {
		if _, err := driveCmd(t, "config", "set", c.key, c.val); err != nil {
			t.Fatalf("set %s: %v", c.key, err)
		}
	}
	for _, c := range cases {
		got, err := driveCmd(t, "config", "get", c.key)
		if err != nil {
			t.Fatalf("get %s: %v", c.key, err)
		}
		if strings.TrimSpace(got) != c.val {
			t.Errorf("get %s = %q, want %q", c.key, strings.TrimSpace(got), c.val)
		}
	}
}

func TestConfigRmClearsEachKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	keys := []struct {
		key, val string
	}{
		{"base_url", "https://x"},
		{"deribit_base_url", "https://y"},
		{"output", "yaml"},
		{"timeout", "30s"},
	}
	for _, k := range keys {
		if _, err := driveCmd(t, "config", "set", k.key, k.val); err != nil {
			t.Fatalf("set %s: %v", k.key, err)
		}
	}
	for _, k := range keys {
		if _, err := driveCmd(t, "config", "rm", k.key); err != nil {
			t.Fatalf("rm %s: %v", k.key, err)
		}
		got, err := driveCmd(t, "config", "get", k.key)
		if err != nil {
			t.Fatalf("get-after-rm %s: %v", k.key, err)
		}
		if strings.TrimSpace(got) != "" {
			t.Errorf("after rm %s got %q", k.key, got)
		}
	}
}

func TestConfigGetUnknownKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	_, err := driveCmd(t, "config", "get", "no_such_key")
	if err == nil {
		t.Fatal("expected unknown-key error")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("err = %v", err)
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	_, err := driveCmd(t, "config", "set", "no_such_key", "val")
	if err == nil {
		t.Fatal("expected unknown-key error")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("err = %v", err)
	}
}

func TestConfigPathFlagOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	flagPath := filepath.Join(dir, "flag.toml")
	envPath := filepath.Join(dir, "env.toml")
	t.Setenv("ORACLE_CONFIG", envPath)

	// write via flag path
	if _, err := driveCmd(t, "config", "--config", flagPath, "set", "base_url", "https://from-flag"); err != nil {
		t.Fatalf("set: %v", err)
	}
	// flag-path file must exist; env path must NOT
	if _, err := os.Stat(flagPath); err != nil {
		t.Errorf("flag path not created: %v", err)
	}
	if _, err := os.Stat(envPath); err == nil {
		t.Errorf("env path should NOT have been used")
	}
}

func TestConfigRmRejectsSecretKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	_, err := driveCmd(t, "config", "rm", "oracle_api_token")
	if err == nil {
		t.Fatal("expected refusal")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Errorf("err = %v", err)
	}
}
