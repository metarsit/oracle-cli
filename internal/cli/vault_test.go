// internal/cli/vault_test.go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultInitSetGetListRm(t *testing.T) {
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "secrets.vault")
	t.Setenv("ORACLE_VAULT", vaultPath)
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "hunter2")
	t.Setenv("XDG_CONFIG_HOME", dir)

	run := func(args ...string) (string, string, error) {
		root := NewRootCmd("test")
		var out, errBuf bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errBuf)
		root.SetArgs(args)
		err := root.Execute()
		return out.String(), errBuf.String(), err
	}

	if _, _, err := run("vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := os.Stat(vaultPath); err != nil {
		t.Fatalf("vault file not created: %v", err)
	}
	if _, _, err := run("vault", "set", "oracle_api_token", "tok-xyz"); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, _, err := run("vault", "get", "oracle_api_token")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(out) != "tok-xyz" {
		t.Errorf("got %q want tok-xyz", out)
	}
	out, _, err = run("vault", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "oracle_api_token") {
		t.Errorf("list missing key: %q", out)
	}
	if _, _, err := run("vault", "rm", "oracle_api_token"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	out, _, _ = run("vault", "list")
	if strings.Contains(out, "oracle_api_token") {
		t.Errorf("rm failed: %q", out)
	}
}

// vaultRun is a small helper local to vault tests that mirrors the inline
// `run` closure above but returns (stdout, stderr, err) without forcing the
// caller to redeclare the boilerplate.
func vaultRun(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := NewRootCmd("t")
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errBuf.String(), err
}

func TestVaultExportRequiresConfirm(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "secrets.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	t.Setenv("XDG_CONFIG_HOME", dir)

	if _, _, err := vaultRun(t, "vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	_, _, err := vaultRun(t, "vault", "export")
	if err == nil {
		t.Fatal("expected refusal without --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("err = %v", err)
	}
}

func TestVaultExportWithConfirmPrintsTOMLLines(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "secrets.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	t.Setenv("XDG_CONFIG_HOME", dir)

	if _, _, err := vaultRun(t, "vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, _, err := vaultRun(t, "vault", "set", "oracle_api_token", "tok-1"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, _, err := vaultRun(t, "vault", "set", "deribit_client_id", "cid-1"); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, _, err := vaultRun(t, "vault", "export", "--confirm")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(out, `oracle_api_token = "tok-1"`) {
		t.Errorf("export missing token line: %q", out)
	}
	if !strings.Contains(out, `deribit_client_id = "cid-1"`) {
		t.Errorf("export missing cid line: %q", out)
	}
}

func TestVaultInitRefusesIfExists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "secrets.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	t.Setenv("XDG_CONFIG_HOME", dir)

	if _, _, err := vaultRun(t, "vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	_, _, err := vaultRun(t, "vault", "init")
	if err == nil {
		t.Fatal("expected init to refuse second time")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("err = %v", err)
	}
}

func TestVaultGetUnknownKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "secrets.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	t.Setenv("XDG_CONFIG_HOME", dir)

	if _, _, err := vaultRun(t, "vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	_, _, err := vaultRun(t, "vault", "get", "nope")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "not in vault") {
		t.Errorf("err = %v", err)
	}
}

func TestVaultListErrorWhenMissingVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "does-not-exist.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	_, _, err := vaultRun(t, "vault", "list")
	if err == nil {
		t.Fatal("expected error opening missing vault")
	}
}

func TestVaultSetErrorWhenMissingVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "does-not-exist.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	_, _, err := vaultRun(t, "vault", "set", "k", "v")
	if err == nil {
		t.Fatal("expected error opening missing vault")
	}
}

func TestVaultRmErrorWhenMissingVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "does-not-exist.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	_, _, err := vaultRun(t, "vault", "rm", "k")
	if err == nil {
		t.Fatal("expected error opening missing vault")
	}
}

func TestVaultExportErrorWhenMissingVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "does-not-exist.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	_, _, err := vaultRun(t, "vault", "export", "--confirm")
	if err == nil {
		t.Fatal("expected error opening missing vault")
	}
}

func TestVaultPathFlagOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	flagPath := filepath.Join(dir, "flag.vault")
	envPath := filepath.Join(dir, "env.vault")
	t.Setenv("ORACLE_VAULT", envPath)
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	t.Setenv("XDG_CONFIG_HOME", dir)
	if _, _, err := vaultRun(t, "vault", "--vault", flagPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := os.Stat(flagPath); err != nil {
		t.Errorf("flag vault not created: %v", err)
	}
	if _, err := os.Stat(envPath); err == nil {
		t.Errorf("env vault should not have been used")
	}
}

func TestVaultRotateRequiresTTY(t *testing.T) {
	// rotate uses TermPrompter directly with no env override. In non-TTY
	// runs term.ReadPassword fails immediately; assert the documented
	// behaviour rather than fake stdin (per task brief).
	dir := t.TempDir()
	t.Setenv("ORACLE_VAULT", filepath.Join(dir, "secrets.vault"))
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "pw")
	t.Setenv("XDG_CONFIG_HOME", dir)
	if _, _, err := vaultRun(t, "vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	_, _, err := vaultRun(t, "vault", "rotate")
	if err == nil {
		t.Fatal("expected rotate to fail in non-TTY environment")
	}
}
