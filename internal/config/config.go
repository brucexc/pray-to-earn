package config

import (
	"fmt"
	"os"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

const (
	Environment            = "environment"
	EnvironmentDevelopment = "development"

	KeyConfig = "config"
)

type File struct {
	Environment string     `yaml:"environment" validate:"required" default:"development"`
	Database    *Database  `yaml:"database"`
	Redis       *Redis     `yaml:"redis"`
	RSS3Chain   *RSS3Chain `yaml:"rss3_chain"`
	AdminKey    string     `yaml:"admin_key"`
}

type Database struct {
	URI string `mapstructure:"uri"`
}

type Redis struct {
	URI string `mapstructure:"uri" validate:"required" default:"redis://localhost:6379/0"`
}

type RSS3Chain struct {
	Endpoint string `yaml:"endpoint" validate:"required" default:" https://rpc.testnet.rss3.io"`
}

func Setup(configFilePath string) (*File, error) {
	config, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var configFile File
	if err := yaml.Unmarshal(config, &configFile); err != nil {
		return nil, fmt.Errorf("unmarshal config file: %w", err)
	}

	if err := defaults.Set(&configFile); err != nil {
		return nil, fmt.Errorf("set default values.yaml: %w", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(&configFile); err != nil {
		return nil, fmt.Errorf("validate config file: %w", err)
	}

	return &configFile, nil
}
