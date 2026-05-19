// internal/cli/hedge.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newHedgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hedge",
		Short: "GET /v1/hedges/latest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			asset, _ := cmd.Flags().GetString("asset")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.HedgeLatest(cmd.Context(), asset)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("asset", "", "BTC|ETH")
	_ = cmd.MarkFlagRequired("asset")
	return cmd
}
