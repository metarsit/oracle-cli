// internal/config/config_test.go
package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	in := File{
		BaseURL:        "https://oracle.example",
		DeribitBaseURL: "https://www.deribit.com/api/v2",
		Output:         "json",
		Timeout:        "15s",
	}
	if err := SaveFile(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != in {
		t.Errorf("roundtrip mismatch: got %+v want %+v", got, in)
	}
}

func TestLoadFileMissingReturnsZeroValue(t *testing.T) {
	got, err := LoadFile(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if (got != File{}) {
		t.Errorf("expected zero File, got %+v", got)
	}
}

func TestSaveFileRejectsSecretKeys(t *testing.T) {
	dir := t.TempDir()
	in := File{BaseURL: "x", OracleAPIToken: "leak"}
	err := SaveFile(filepath.Join(dir, "c.toml"), in)
	if err == nil {
		t.Fatal("expected error rejecting secret in config file")
	}
}
