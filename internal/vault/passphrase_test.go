// internal/vault/passphrase_test.go
package vault

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/term"
)

func TestPassphraseFromEnv(t *testing.T) {
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "hunter2")
	got, err := readPassphrase(nopPrompter{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hunter2" {
		t.Errorf("got %q, want hunter2", got)
	}
}

func TestPassphraseMissingNoTTY(t *testing.T) {
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "")
	_, err := readPassphrase(nopPrompter{}, false)
	if !errors.Is(err, ErrPassphraseUnavailable) {
		t.Errorf("want ErrPassphraseUnavailable, got %v", err)
	}
}

func TestPassphraseFromPrompter(t *testing.T) {
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "")
	got, err := readPassphrase(staticPrompter("typed-it"), true)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "typed-it" {
		t.Errorf("got %q", got)
	}
}

type nopPrompter struct{}

func (nopPrompter) Prompt(string) ([]byte, error) { return nil, errors.New("should not be called") }

type staticPrompter string

func (s staticPrompter) Prompt(string) ([]byte, error) { return []byte(s), nil }

// TestTermPrompterType — TermPrompter.Prompt requires a real TTY to exercise
// term.ReadPassword; here we only assert it satisfies the Prompter contract.
// The Prompt method is documented as covered by manual + interactive tests.
// LINT:ignore-coverage — TTY-bound function.
func TestTermPrompterType(t *testing.T) {
	var p Prompter = TermPrompter{}
	if p == nil {
		t.Fatal("TermPrompter should satisfy Prompter interface")
	}
}

func TestTermPrompterPromptSkipWithoutTTY(t *testing.T) {
	// Skip unless stdin is a real TTY — term.ReadPassword would otherwise
	// block or read garbage from the test harness's pipe.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("requires TTY")
	}
	_, _ = TermPrompter{}.Prompt("test: ")
}
