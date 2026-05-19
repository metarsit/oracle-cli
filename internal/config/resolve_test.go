// internal/config/resolve_test.go
package config

import (
	"math/rand"
	"strconv"
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

// TestResolvePropertyFlagAlwaysWins exercises the invariant: when a CLI flag
// value is non-empty, it must beat env, vault, and file regardless of what
// any of those carry. We feed 50 randomised inputs and assert the postcondition.
func TestResolvePropertyFlagAlwaysWins(t *testing.T) {
	rng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic seed is intentional for reproducible property tests

	// pool of values used across all sources to ensure they often collide
	pool := []string{"alpha", "beta", "gamma", "delta", "", "epsilon"}
	pick := func() string { return pool[rng.Intn(len(pool))] }

	for i := 0; i < 50; i++ {
		flagURL := "flag-" + strconv.Itoa(i) // always non-empty
		flagTok := "tok-" + strconv.Itoa(i)
		flagOut := "json" // always non-empty
		flagTO := "7s"

		t.Setenv("ORACLE_BASE_URL", pick())
		t.Setenv("ORACLE_API_TOKEN", pick())
		t.Setenv("ORACLE_OUTPUT", pick())
		t.Setenv("ORACLE_TIMEOUT", pick())

		in := Inputs{
			Flag: Flags{
				BaseURL: flagURL,
				Token:   flagTok,
				Output:  flagOut,
				Timeout: flagTO,
			},
			File:  File{BaseURL: pick(), Output: pick(), Timeout: pick()},
			Vault: map[string]string{"oracle_api_token": pick(), "base_url": pick()},
		}
		got := Resolve(in)
		if got.BaseURL != flagURL {
			t.Errorf("iter %d: BaseURL = %q, want flag %q", i, got.BaseURL, flagURL)
		}
		if got.Token != flagTok {
			t.Errorf("iter %d: Token = %q, want flag %q", i, got.Token, flagTok)
		}
		if got.Output != flagOut {
			t.Errorf("iter %d: Output = %q, want flag %q", i, got.Output, flagOut)
		}
		if got.Timeout != 7*time.Second {
			t.Errorf("iter %d: Timeout = %v, want 7s", i, got.Timeout)
		}
	}
}

// TestResolvePropertyDefaultsWhenAllEmpty: when no source supplies a value,
// the documented defaults must surface. Exercises the bottom-of-stack branch.
func TestResolvePropertyDefaultsWhenAllEmpty(t *testing.T) {
	t.Setenv("ORACLE_BASE_URL", "")
	t.Setenv("ORACLE_API_TOKEN", "")
	t.Setenv("ORACLE_OUTPUT", "")
	t.Setenv("ORACLE_TIMEOUT", "")
	t.Setenv("DERIBIT_BASE_URL", "")
	t.Setenv("DERIBIT_CLIENT_ID", "")
	t.Setenv("DERIBIT_CLIENT_SECRET", "")
	got := Resolve(Inputs{})
	if got.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL default = %q, want %q", got.BaseURL, defaultBaseURL)
	}
	if got.DeribitBaseURL != defaultDeribitBaseURL {
		t.Errorf("DeribitBaseURL default = %q, want %q", got.DeribitBaseURL, defaultDeribitBaseURL)
	}
	if got.Output != defaultOutput {
		t.Errorf("Output default = %q, want %q", got.Output, defaultOutput)
	}
	if got.Timeout != defaultTimeout {
		t.Errorf("Timeout default = %v, want %v", got.Timeout, defaultTimeout)
	}
}
