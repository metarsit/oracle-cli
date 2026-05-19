package config

import (
	"os"
	"path/filepath"
)

const (
	appDir       = "oracle-cli"
	configFile   = "config.toml"
	vaultFile    = "secrets.vault"
	envXDGConfig = "XDG_CONFIG_HOME"
	envHome      = "HOME"
)

// configDir returns the directory where config and vault files live.
func configDir() string {
	if xdg := os.Getenv(envXDGConfig); xdg != "" {
		return filepath.Join(xdg, appDir)
	}
	return filepath.Join(os.Getenv(envHome), ".config", appDir)
}

// Path returns the absolute path of config.toml.
func Path() string { return filepath.Join(configDir(), configFile) }

// VaultPath returns the absolute path of secrets.vault.
func VaultPath() string { return filepath.Join(configDir(), vaultFile) }

// EnsureConfigDir creates the config directory with mode 0700.
func EnsureConfigDir() error {
	return os.MkdirAll(configDir(), 0o700)
}
