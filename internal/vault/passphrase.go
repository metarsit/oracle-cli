// internal/vault/passphrase.go
package vault

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

const envPassphrase = "ORACLE_VAULT_PASSPHRASE"

// ErrPassphraseUnavailable is returned when no passphrase source is reachable.
var ErrPassphraseUnavailable = errors.New("vault: no passphrase available; set ORACLE_VAULT_PASSPHRASE or run interactively")

// Prompter abstracts TTY interaction for testability.
type Prompter interface {
	Prompt(label string) ([]byte, error)
}

// TermPrompter reads from /dev/tty with echo off.
type TermPrompter struct{}

func (TermPrompter) Prompt(label string) ([]byte, error) {
	fmt.Fprint(os.Stderr, label)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return pw, err
}

// ReadPassphrase is the public helper used by cli code.
func ReadPassphrase() ([]byte, error) {
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	return readPassphrase(TermPrompter{}, isTTY)
}

func readPassphrase(p Prompter, isTTY bool) ([]byte, error) {
	if v := os.Getenv(envPassphrase); v != "" {
		return []byte(v), nil
	}
	if !isTTY {
		return nil, ErrPassphraseUnavailable
	}
	return p.Prompt("Vault passphrase: ")
}
