// internal/cli/instruments.go
package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newInstrumentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instruments",
		Short: "GET /v1/instruments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			base, _ := cmd.Flags().GetString("base")
			kind, _ := cmd.Flags().GetString("kind")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.Instruments(cmd.Context(), base, kind)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("base", "", "BTC|ETH")
	cmd.Flags().String("kind", "", "option|perp|future")
	_ = cmd.MarkFlagRequired("base")
	return cmd
}
