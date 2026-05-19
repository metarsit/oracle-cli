// internal/cli/engine.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newEngineCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "engine", Short: "Engine controls"}
	cmd.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "POST /v1/engine/run",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			if err := c.EngineRun(cmd.Context()); err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), map[string]bool{"started": true})
		},
	})
	return cmd
}
