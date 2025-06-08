package config

import (
	"context"
	// "fmt" // Not needed if GrpcConfig.String() is not used locally or common one is sufficient

	commonConfig "github.com/example/current-repo/common/config"
	"github.com/sethvargo/go-envconfig" // Import envconfig here
)

var Config StructConfig

type StructConfig struct {
	commonConfig.BaseConfig `env:",squash"` // Use squash to make BaseConfig fields act as if they are part of StructConfig
	CreatorGithubURL        string          `env:"CREATOR_GITHUB_URL"`
	MaxBotsByUser           int64           `env:"MAX_BOTS_BY_USER, default=10"`
	// If GRPC_SERVER_HOST is strictly required for creator_bot, it must be ensured by the environment.
	// Alternatively, add a specific field here and validate in NewConfig or use a custom processor.
	// For example:
	// RequiredGrpcHost string `env:"GRPC_SERVER_HOST,required"`
	// Then in NewConfig: config.BaseConfig.Grpc.Host = config.RequiredGrpcHost
}

func NewConfig() (StructConfig, error) {
	var config StructConfig
	// envconfig.Process will initialize nil pointer fields in BaseConfig
	// (e.g., Mongo, Redis, TelegramBot, Grpc) if `env:",squash"` is used.

	if err := envconfig.Process(context.Background(), &config); err != nil {
		return StructConfig{}, err
	}

	// Example of custom validation or logic after loading:
	// If creator_bot absolutely requires GRPC host and port to be set:
	// if config.BaseConfig.Grpc == nil || config.BaseConfig.Grpc.Host == "" {
	// 	return StructConfig{}, fmt.Errorf("GRPC_SERVER_HOST is required for creator_bot")
	// }
	// if config.BaseConfig.Grpc.Port == 0 {
	//	return StructConfig{}, fmt.Errorf("GRPC_SERVER_PORT is required for creator_bot")
	// }
	// This enforcement should align with deployment practices. For now, assume env vars handle requirements.

	Config = config
	return config, nil
}
