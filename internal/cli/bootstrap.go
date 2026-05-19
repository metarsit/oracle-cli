// Package cli implements the oracle command-line interface.
package cli

import (
	"github.com/metarsit/oracle-cli/internal/config"
	"github.com/metarsit/oracle-cli/internal/vault"
	"github.com/spf13/cobra"
)

type resolvedConfig struct {
	config.Resolved
}

// bootstrap turns the parsed cobra command + global flags into a Resolved
// config. needsAuth=true triggers lazy vault open when the token is missing
// from flag+env.
func bootstrap(cmd *cobra.Command, needsAuth bool) (resolvedConfig, error) {
	flags := config.Flags{}
	flags.BaseURL, _ = cmd.Flags().GetString("base-url")
	flags.Token, _ = cmd.Flags().GetString("token")
	flags.Output, _ = cmd.Flags().GetString("output")
	flags.Timeout, _ = cmd.Flags().GetString("timeout")

	file, err := config.LoadFile(configPath(cmd))
	if err != nil {
		return resolvedConfig{}, err
	}

	vaultMap := map[string]string{}
	if needsAuth {
		preview := config.Resolve(config.Inputs{Flag: flags, File: file, Vault: nil})
		if preview.Token == "" || preview.DeribitClientSecret == "" {
			pw, err := vault.ReadPassphrase()
			if err == nil {
				if v, vErr := vault.Open(vaultPath(cmd), pw); vErr == nil {
					vaultMap = v.Secrets
				}
			}
		}
	}

	r := config.Resolve(config.Inputs{Flag: flags, File: file, Vault: vaultMap})
	return resolvedConfig{Resolved: r}, nil
}
