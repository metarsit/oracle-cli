package cli

import (
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/config"
	"github.com/spf13/cobra"
)

func configPath(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("config"); v != "" {
		return v
	}
	if v := os.Getenv("ORACLE_CONFIG"); v != "" {
		return v
	}
	return config.Path()
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage CLI config (non-secret keys)"}
	cmd.PersistentFlags().String("config", "", "Path to config.toml")
	cmd.AddCommand(newConfigShowCmd(), newConfigGetCmd(), newConfigSetCmd(), newConfigRmCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use: "show", Short: "Print resolved non-secret config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			f, err := config.LoadFile(configPath(cmd))
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "base_url = %q\n", f.BaseURL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "deribit_base_url = %q\n", f.DeribitBaseURL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "output = %q\n", f.Output)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "timeout = %q\n", f.Timeout)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <key>", Short: "Print one config value",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.LoadFile(configPath(cmd))
			if err != nil {
				return err
			}
			v, err := configFieldGet(f, args[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), v)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "set <key> <value>", Short: "Set one config value",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isSecretKey(args[0]) {
				return fmt.Errorf("%q is a secret; use `oracle vault set` instead", args[0])
			}
			path := configPath(cmd)
			f, err := config.LoadFile(path)
			if err != nil {
				return err
			}
			if err := configFieldSet(&f, args[0], args[1]); err != nil {
				return err
			}
			if err := config.EnsureConfigDir(); err != nil {
				return err
			}
			return config.SaveFile(path, f)
		},
	}
}

func newConfigRmCmd() *cobra.Command {
	return &cobra.Command{
		Use: "rm <key>", Short: "Clear one config value",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isSecretKey(args[0]) {
				return fmt.Errorf("%q is a secret; use `oracle vault rm`", args[0])
			}
			path := configPath(cmd)
			f, err := config.LoadFile(path)
			if err != nil {
				return err
			}
			if err := configFieldSet(&f, args[0], ""); err != nil {
				return err
			}
			return config.SaveFile(path, f)
		},
	}
}

func isSecretKey(k string) bool {
	switch k {
	case "oracle_api_token", "deribit_client_id", "deribit_client_secret":
		return true
	}
	return false
}

func configFieldGet(f config.File, k string) (string, error) {
	switch k {
	case "base_url":
		return f.BaseURL, nil
	case "deribit_base_url":
		return f.DeribitBaseURL, nil
	case "output":
		return f.Output, nil
	case "timeout":
		return f.Timeout, nil
	}
	return "", fmt.Errorf("unknown config key %q", k)
}

func configFieldSet(f *config.File, k, v string) error {
	switch k {
	case "base_url":
		f.BaseURL = v
	case "deribit_base_url":
		f.DeribitBaseURL = v
	case "output":
		f.Output = v
	case "timeout":
		f.Timeout = v
	default:
		return fmt.Errorf("unknown config key %q", k)
	}
	return nil
}
