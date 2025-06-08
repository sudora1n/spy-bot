package config

import (
	"context"
	"fmt"
	"net/url"

	"github.com/sethvargo/go-envconfig"
)

// MongoConfig holds MongoDB connection parameters.
type MongoConfig struct {
	Host     string            `env:"HOST,required"`
	Port     int               `env:"PORT,required"`
	Database string            `env:"DB,default=ssuspy"`
	Username string            `env:"USERNAME"`
	Password string            `env:"PASSWORD"`
	Options  map[string]string `env:"OPTIONS,separator=|"`
}

// BuildMongoURI constructs the MongoDB connection string.
func (m *MongoConfig) BuildMongoURI() string {
	var auth string
	if m.Username != "" && m.Password != "" {
		auth = fmt.Sprintf("%s:%s@", url.QueryEscape(m.Username), url.QueryEscape(m.Password))
	}

	var query string
	if len(m.Options) > 0 {
		q := url.Values{}
		for key, value := range m.Options {
			q.Add(key, value)
		}
		query = "?" + q.Encode()
	}

	return fmt.Sprintf("mongodb://%s%s:%d/%s%s", auth, m.Host, m.Port, m.Database, query)
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Host     string `env:"HOST,default=localhost"`
	Port     int    `env:"PORT,default=6379"`
	Password string `env:"PASSWORD"`
	Database int    `env:"DBNAME,default=0"`
}

// TelegramBotConfig holds Telegram bot parameters.
// Token is intentionally made optional here; individual bots can enforce requirement if needed.
type TelegramBotConfig struct {
	Token  string `env:"TOKEN"` // Was "TOKEN,required"
	ApiURL string `env:"API_URL,required"`
}

// GrpcConfig holds gRPC server parameters.
// Host and Port are optional at the base level.
// Specific services (like creator_bot) must ensure their required env vars are set.
type GrpcConfig struct {
	Host string `env:"HOST"`
	Port int    `env:"PORT"`
}

func (g *GrpcConfig) String() string {
	if g.Host == "" || g.Port == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}

// BaseConfig contains common configuration settings.
type BaseConfig struct {
	Mongo       *MongoConfig       `env:",prefix=MONGO_"`
	Redis       *RedisConfig       `env:",prefix=REDIS_"`
	TelegramBot *TelegramBotConfig `env:",prefix=TELEGRAM_"`
	Grpc        *GrpcConfig        `env:",prefix=GRPC_SERVER_"`
	DevMode     bool               `env:"DEV_MODE,default=false"`
}

// LoadBaseConfig loads configuration from environment variables into BaseConfig.
// This function might not be strictly necessary if embedding structs call envconfig.Process on themselves.
// However, it can be a utility if a module ONLY needs the BaseConfig.
func LoadBaseConfig(ctx context.Context, cfg *BaseConfig) error {
	// Ensure pointer fields are initialized before processing.
	// envconfig.Process typically handles this for top-level fields,
	// but when calling for a sub-struct, it's good practice.
	if cfg.Mongo == nil {
		cfg.Mongo = &MongoConfig{}
	}
	if cfg.Redis == nil {
		cfg.Redis = &RedisConfig{}
	}
	if cfg.TelegramBot == nil {
		cfg.TelegramBot = &TelegramBotConfig{}
	}
	if cfg.Grpc == nil {
		cfg.Grpc = &GrpcConfig{}
	}
	return envconfig.Process(ctx, cfg)
}
