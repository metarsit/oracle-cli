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
