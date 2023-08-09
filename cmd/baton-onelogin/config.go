package main

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/spf13/cobra"
)

// config defines the external configuration required for the connector to run.
type config struct {
	cli.BaseConfig `mapstructure:",squash"` // Puts the base config options in the same place as the connector options

	ClientId     string `mapstructure:"onelogin-client-id"`
	ClientSecret string `mapstructure:"onelogin-client-secret"`
	Subdomain    string `mapstructure:"subdomain"`
}

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *config) error {
	if cfg.ClientId == "" || cfg.ClientSecret == "" || cfg.Subdomain == "" {
		return fmt.Errorf("onelogin-client-id, onelogin-client-secret and subdomain must be provided")
	}

	return nil
}

// cmdFlags sets the cmdFlags required for the connector.
func cmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("onelogin-client-id", "", "OneLogin client ID used to generate the access token. ($BATON_ONELOGIN_CLIENT_ID)")
	cmd.PersistentFlags().String("onelogin-client-secret", "", "OneLogin client secret used to generate the access token. ($BATON_ONELOGIN_CLIENT_SECRET)")
	cmd.PersistentFlags().String("subdomain", "", "OneLogin subdomain to connect to. ($BATON_SUBDOMAIN)")
}
