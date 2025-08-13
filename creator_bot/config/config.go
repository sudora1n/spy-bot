package config

import (
	"context"
	"fmt"
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
	Mongo            *types.MongoConfig `env:", prefix=MONGO_"`
	Redis            *types.RedisConfig `env:", prefix=REDIS_"`
	TelegramBot      *BotConfig         `env:", prefix=TELEGRAM_"`
	Grpc             *GrpcConfig        `env:", prefix=GRPC_SERVER_"`
	CreatorGithubURL string             `env:"CREATOR_GITHUB_URL"`
	MaxBotsByUser    int64              `env:"MAX_BOTS_BY_USER, default=10"`
	BusinessURL      string             `env:"BUSINESS_URL, default=http://business-bot:8080"`
	DevMode          bool               `env:"DEV_MODE, default=false"`
}

type GrpcConfig struct {
	Host string `env:"HOST, required"`
	Port int    `env:"PORT, default=50051"`
}

func (g *GrpcConfig) String() string {
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}

type BotConfig struct {
	Token  string `env:"TOKEN, required"`
	ApiURL string `env:"API_URL, required"`
}
