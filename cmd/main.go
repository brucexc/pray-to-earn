package main

import (
	"context"
	"fmt"
	"os"

	"github.com/brucexc/pray-to-earn/internal/config"
	"github.com/brucexc/pray-to-earn/internal/service"
	"github.com/brucexc/pray-to-earn/internal/service/hub"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var command = cobra.Command{
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		server := service.NewServer(
			hub.Module,
			fx.Provide(hub.NewServer),
		)

		if err := server.Start(cmd.Context()); err != nil {
			return fmt.Errorf("start server: %w", err)
		}

		server.Wait()

		return nil
	},
}

func initializeLogger() {
	if os.Getenv(config.Environment) == config.EnvironmentDevelopment {
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	} else {
		zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	}
}

func init() {
	initializeLogger()

	command.PersistentFlags().String(config.KeyConfig, "./deploy/config.yaml", "config file path")
}

func main() {
	if err := command.ExecuteContext(context.Background()); err != nil {
		zap.L().Fatal("execute command", zap.Error(err))
	}
}
