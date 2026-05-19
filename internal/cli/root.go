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
	cmd.AddCommand(
		newVersionCmd(version),
		newVaultCmd(),
	)
	return cmd
}
