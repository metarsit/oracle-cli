package cli

import (
	"github.com/metarsit/oracle-cli/internal/deribit"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newDeribitCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "deribit", Short: "Read-only Deribit queries (separate creds)"}
	cmd.AddCommand(newDeribitBalanceCmd(), newDeribitPositionsCmd())
	return cmd
}

func newDeribitBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "GET /private/get_account_summary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			currency, _ := cmd.Flags().GetString("currency")
			c := deribit.New(cfg.DeribitBaseURL, cfg.DeribitClientID, cfg.DeribitClientSecret, cfg.Timeout)
			data, err := c.AccountSummary(cmd.Context(), currency)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("currency", "BTC", "BTC|ETH|USDC")
	return cmd
}

func newDeribitPositionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "positions",
		Short: "GET /private/get_positions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			currency, _ := cmd.Flags().GetString("currency")
			c := deribit.New(cfg.DeribitBaseURL, cfg.DeribitClientID, cfg.DeribitClientSecret, cfg.Timeout)
			data, err := c.Positions(cmd.Context(), currency)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("currency", "BTC", "BTC|ETH|USDC")
	return cmd
}
