// internal/client/errors_test.go
package client

import (
	"errors"
	"io"
	"testing"
)

func TestErrorStringers(t *testing.T) {
	netCause := errors.New("dial tcp: refused")
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"auth", &ErrAuth{Status: 401, Msg: "bad token"}, "auth failed (401): bad token"},
		{"auth_forbidden", &ErrAuth{Status: 403, Msg: ""}, "auth failed (403): "},
		{"notfound", &ErrNotFound{Msg: "missing"}, "not found: missing"},
		{"notfound_empty", &ErrNotFound{Msg: ""}, "not found: "},
		{"network", &ErrNetwork{Err: netCause}, "network: dial tcp: refused"},
		{"degraded", &ErrDegraded{Status: 503, Msg: "warmup"}, "degraded (503): warmup"},
		{"api", &ErrAPI{Code: "RATE", Msg: "slow down", Status: 429}, "RATE (429): slow down"},
		{"api_empty_code", &ErrAPI{Code: "", Msg: "boom", Status: 500}, " (500): boom"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.err.Error(); got != c.want {
				t.Errorf("Error() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestErrNetworkUnwrap(t *testing.T) {
	cause := io.EOF
	wrapped := &ErrNetwork{Err: cause}
	if !errors.Is(wrapped, io.EOF) {
		t.Errorf("errors.Is should walk Unwrap chain to io.EOF")
	}
	if u := errors.Unwrap(wrapped); u != cause {
		t.Errorf("Unwrap() = %v, want io.EOF", u)
	}
}

func TestErrorsAsChain(t *testing.T) {
	// Wrap a typed client error inside a generic fmt error and ensure
	// errors.As still extracts the typed value.
	original := &ErrAuth{Status: 401, Msg: "x"}
	wrapped := errors.Join(errors.New("ctx"), original)
	var got *ErrAuth
	if !errors.As(wrapped, &got) {
		t.Fatalf("As did not find *ErrAuth in chain")
	}
	if got.Status != 401 {
		t.Errorf("recovered Status = %d", got.Status)
	}
}
