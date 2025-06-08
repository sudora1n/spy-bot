package config

import (
	"context"

	commonConfig "github.com/example/current-repo/common/config"
	"github.com/sethvargo/go-envconfig" // Import envconfig here
)

var Config StructConfig

type StructConfig struct {
	commonConfig.BaseConfig `env:",squash"` // Use squash
	BusinessGithubURL       string          `env:"BUSINESS_GITHUB_URL"`
	FilesWorkers            int             `env:"FILES_WORKERS,default=5"`
}

func NewConfig() (StructConfig, error) {
	var config StructConfig
	// envconfig.Process will initialize nil pointer fields in BaseConfig.

	if err := envconfig.Process(context.Background(), &config); err != nil {
		return StructConfig{}, err
	}

	// Business_bot specific considerations:
	// - It does not use gRPC, so BaseConfig.Grpc might be populated from env but will be ignored by the bot.
	// - TelegramBotConfig.Token in BaseConfig is optional. If business_bot uses Telegram notifications
	//   and requires a token, its environment must provide TELEGRAM_TOKEN.
	//   If TELEGRAM_TOKEN is not set, config.BaseConfig.TelegramBot.Token will be empty.
	//   The application logic must handle this (e.g., disable notifications if token is missing).

	Config = config
	return config, nil
}
