// internal/config/resolve_test.go
package config

import (
	"testing"
	"time"
)

func TestResolvePrecedence(t *testing.T) {
	t.Setenv("ORACLE_BASE_URL", "env-url")
	t.Setenv("ORACLE_API_TOKEN", "env-tok")
	t.Setenv("ORACLE_OUTPUT", "")
	t.Setenv("ORACLE_TIMEOUT", "")

	in := Inputs{
		Flag:  Flags{}, // no flag override
		File:  File{BaseURL: "file-url", Output: "json"},
		Vault: map[string]string{"oracle_api_token": "vault-tok"},
	}
	got := Resolve(in)
	if got.BaseURL != "env-url" {
		t.Errorf("BaseURL: env should beat file: got %q", got.BaseURL)
	}
	if got.Token != "env-tok" {
		t.Errorf("Token: env should beat vault: got %q", got.Token)
	}
	if got.Output != "json" {
		t.Errorf("Output: file should fill missing env: got %q", got.Output)
	}
	if got.Timeout != 10*time.Second {
		t.Errorf("Timeout default missing: got %v", got.Timeout)
	}
}

func TestResolveFlagBeatsAll(t *testing.T) {
	t.Setenv("ORACLE_BASE_URL", "env")
	in := Inputs{
		Flag:  Flags{BaseURL: "flag"},
		File:  File{BaseURL: "file"},
		Vault: map[string]string{},
	}
	if got := Resolve(in); got.BaseURL != "flag" {
		t.Errorf("got %q want flag", got.BaseURL)
	}
}
