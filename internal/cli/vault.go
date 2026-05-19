// internal/cli/vault.go
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/config"
	"github.com/metarsit/oracle-cli/internal/vault"
	"github.com/spf13/cobra"
)

// vaultPath resolves --vault flag -> ORACLE_VAULT env -> default XDG path.
func vaultPath(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("vault"); v != "" {
		return v
	}
	if v := os.Getenv("ORACLE_VAULT"); v != "" {
		return v
	}
	return config.VaultPath()
}

func newVaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Manage the encrypted secret store",
	}
	cmd.PersistentFlags().String("vault", "", "Path to vault file (default $XDG_CONFIG_HOME/oracle-cli/secrets.vault)")
	cmd.AddCommand(
		newVaultInitCmd(),
		newVaultSetCmd(),
		newVaultGetCmd(),
		newVaultListCmd(),
		newVaultRmCmd(),
		newVaultRotateCmd(),
		newVaultExportCmd(),
	)
	return cmd
}

func newVaultInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create an empty vault",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := vaultPath(cmd)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("vault already exists at %s", path)
			}
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			if err := config.EnsureConfigDir(); err != nil {
				return err
			}
			return vault.Save(path, vault.NewEmpty(), pw)
		},
	}
}

func newVaultSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a secret",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			v.Set(args[0], args[1])
			return vault.Save(path, v, pw)
		},
	}
}

func newVaultGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a secret to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			val, ok := v.Get(args[0])
			if !ok {
				return fmt.Errorf("key %q not in vault", args[0])
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), val)
			return err
		},
	}
}

func newVaultListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List secret keys (values never shown)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			for _, k := range v.Keys() {
				fmt.Fprintln(cmd.OutOrStdout(), k)
			}
			return nil
		},
	}
}

func newVaultRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <key>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			v.Delete(args[0])
			return vault.Save(path, v, pw)
		},
	}
}

func newVaultRotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate",
		Short: "Rotate vault passphrase (uses TTY prompts for old + new)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := vaultPath(cmd)
			oldPw, err := vault.TermPrompter{}.Prompt("Current passphrase: ")
			if err != nil {
				return err
			}
			newPw, err := vault.TermPrompter{}.Prompt("New passphrase: ")
			if err != nil {
				return err
			}
			if len(newPw) == 0 {
				return errors.New("new passphrase empty")
			}
			return vault.Rotate(path, oldPw, newPw)
		},
	}
}

func newVaultExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Print decrypted vault to stdout (requires --confirm)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			confirm, _ := cmd.Flags().GetBool("confirm")
			if !confirm {
				return errors.New("refusing to export without --confirm")
			}
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			for _, k := range v.Keys() {
				val, _ := v.Get(k)
				fmt.Fprintf(cmd.OutOrStdout(), "%s = %q\n", k, val)
			}
			return nil
		},
	}
	cmd.Flags().Bool("confirm", false, "Acknowledge that secrets will be printed in plaintext")
	return cmd
}
