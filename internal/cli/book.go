package cli

import (
	"github.com/metarsit/oracle-cli/internal/client"
	"github.com/metarsit/oracle-cli/internal/format"
	"github.com/spf13/cobra"
)

func newBookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "book",
		Short: "GET /v1/book/top",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := bootstrap(cmd, true)
			if err != nil {
				return err
			}
			inst, _ := cmd.Flags().GetString("instrument")
			c := client.New(cfg.BaseURL, cfg.Token, cfg.Timeout)
			data, err := c.BookTop(cmd.Context(), inst)
			if err != nil {
				return err
			}
			return format.NewRenderer(cfg.Output).Render(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().String("instrument", "", "instrument name")
	_ = cmd.MarkFlagRequired("instrument")
	return cmd
}
