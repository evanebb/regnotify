package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/evanebb/regnotify/configuration"
	"github.com/evanebb/regnotify/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve <config>",
		Short: "`serve` runs the registry notification server",
		Long:  "`serve` runs the registry notification server",
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := buildConfiguration(args)
			if err != nil {
				return err
			}

			return server.Run(context.Background(), conf)
		},
	}
}

func buildConfiguration(args []string) (*configuration.Configuration, error) {
	if len(args) == 0 {
		return nil, errors.New("no configuration path given")
	}

	configurationFile := args[0]

	v := viper.New()

	configuration.SetDefaults(v)
	v.SetConfigFile(configurationFile)
	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %w", configurationFile, err)
	}

	conf := &configuration.Configuration{}

	err = v.Unmarshal(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return conf, err
}
