package config

import (
	"context"
	"ssuspy-common/types"

	"github.com/sethvargo/go-envconfig"
)

var Config StructConfig

func NewConfig() (StructConfig, error) {
	var config StructConfig

	if err := envconfig.Process(context.Background(), &config); err != nil {
		return StructConfig{}, err
	}

	Config = config
	return config, nil
}

type StructConfig struct {
	Mongo       *types.MongoConfig `env:", prefix=MONGO_"`
	Redis       *types.RedisConfig `env:", prefix=REDIS_"`
	TelegramBot *BotConfig         `env:", prefix=TELEGRAM_"`
	JWTSecret   string             `env:"JWT_SECRET, required"`
	DevMode     bool               `env:"DEV_MODE, default=false"`
}

type BotConfig struct {
	ApiURL string `env:"API_URL, required"`
}
