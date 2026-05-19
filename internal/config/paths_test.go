// internal/config/paths_test.go
package config

import (
	"path/filepath"
	"testing"
)

func TestConfigPath_XDGHomeOverridesHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	t.Setenv("HOME", "/home/u")
	got := Path()
	want := "/tmp/xdg/oracle-cli/config.toml"
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

func TestConfigPath_DefaultsToHomeDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/u")
	got := Path()
	want := filepath.Join("/home/u", ".config", "oracle-cli", "config.toml")
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

func TestVaultPath_DefaultsToHomeDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/u")
	got := VaultPath()
	want := filepath.Join("/home/u", ".config", "oracle-cli", "secrets.vault")
	if got != want {
		t.Errorf("VaultPath = %q, want %q", got, want)
	}
}
