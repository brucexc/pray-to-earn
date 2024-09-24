package provider

import (
	"github.com/brucexc/pray-to-earn/internal/config"
	"github.com/spf13/viper"
)

func ProvideConfig() (*config.File, error) {
	return config.Setup(viper.GetString(config.KeyConfig))
}
