// internal/cli/bootstrap_test.go
package cli

import (
	"testing"
)

func TestBootstrapEnvOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ORACLE_BASE_URL", "https://env.example")
	t.Setenv("ORACLE_API_TOKEN", "env-tok")

	root := NewRootCmd("t")
	root.SetArgs([]string{"status"}) // pick any cmd so PersistentFlags parse
	// Don't execute; just resolve flags + bootstrap on the chosen cmd.
	if err := root.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	cfg, err := bootstrap(root, false)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if cfg.BaseURL != "https://env.example" || cfg.Token != "env-tok" {
		t.Errorf("bad cfg: %+v", cfg)
	}
}
