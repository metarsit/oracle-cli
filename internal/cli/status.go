// internal/cli/status.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "GET /v1/status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.Status(cmd.Context())
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
}
