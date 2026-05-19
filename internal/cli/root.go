// internal/cli/root.go
package cli

import "github.com/spf13/cobra"

// NewRootCmd builds the oracle command tree. version is injected from main.
func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "oracle",
		Short:         "Deribit Oracle CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().String("base-url", "", "Oracle base URL (env ORACLE_BASE_URL)")
	cmd.PersistentFlags().String("token", "", "Bearer token (env ORACLE_API_TOKEN; prefer env or vault)")
	cmd.PersistentFlags().String("output", "", "Output format: json|table|yaml (env ORACLE_OUTPUT)")
	cmd.PersistentFlags().String("timeout", "", "HTTP timeout (e.g. 10s)")
	cmd.PersistentFlags().String("config", "", "Path to config.toml")
	cmd.PersistentFlags().String("vault", "", "Path to secrets.vault")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable debug logging on stderr")
	cmd.AddCommand(
		newVersionCmd(version),
		newVaultCmd(),
		newConfigCmd(),
		newHealthCmd(),
		newReadyCmd(),
	)
	return cmd
}
