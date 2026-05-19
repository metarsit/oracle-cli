// internal/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// File is the on-disk config representation. Non-secret keys only.
// OracleAPIToken / DeribitClientID / DeribitClientSecret never persist here;
// they live in the vault. The struct tag-less field exists to detect accidental
// writes and reject them in SaveFile.
type File struct {
	BaseURL        string `toml:"base_url,omitempty"`
	DeribitBaseURL string `toml:"deribit_base_url,omitempty"`
	Output         string `toml:"output,omitempty"`
	Timeout        string `toml:"timeout,omitempty"`

	// Guard fields — must remain empty in persisted state.
	OracleAPIToken      string `toml:"-"`
	DeribitClientID     string `toml:"-"`
	DeribitClientSecret string `toml:"-"`
}

// ErrSecretInConfig is returned when SaveFile is asked to persist a secret.
var ErrSecretInConfig = errors.New("config file refuses to store secrets; use vault")

// LoadFile reads path; missing file returns zero-value File and nil error.
func LoadFile(path string) (File, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is config-controlled
	if errors.Is(err, os.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read config: %w", err)
	}
	var f File
	if err := toml.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("parse config: %w", err)
	}
	return f, nil
}

// SaveFile writes f to path with mode 0600. Refuses to write secret-bearing fields.
func SaveFile(path string, f File) error {
	if f.OracleAPIToken != "" || f.DeribitClientID != "" || f.DeribitClientSecret != "" {
		return ErrSecretInConfig
	}
	b, err := toml.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, b, 0o600)
}
