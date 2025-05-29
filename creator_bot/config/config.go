package config

import (
	"context"
	"fmt"
	"net/url"

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
	Mongo            *MongoConfig `env:", prefix=MONGO_"`
	Redis            *RedisConfig `env:", prefix=REDIS_"`
	TelegramBot      *BotConfig   `env:", prefix=TELEGRAM_"`
	Grpc             *GrpcConfig  `env:", prefix=GRPC_SERVER_"`
	CreatorGithubURL string       `env:"CREATOR_GITHUB_URL"`
	MaxBotsByUser    int64        `env:"MAX_BOTS_BY_USER, default=10"`
	DevMode          bool         `env:"DEV_MODE, default=false"`
}

type GrpcConfig struct {
	Host string `env:"HOST, required"`
	Port int    `env:"PORT, default=50051"`
}

func (g *GrpcConfig) String() string {
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}

type MongoConfig struct {
	Host     string            `env:"HOST, required"`
	Port     int               `env:"PORT, required"`
	Database string            `env:"DB, default=ssuspy"`
	Username string            `env:"USERNAME"`
	Password string            `env:"PASSWORD"`
	Options  map[string]string `env:"OPTIONS, separator=|"`
}

func (m MongoConfig) BuildMongoURI() string {
	var auth = ""
	if m.Username != "" && m.Password != "" {
		auth = fmt.Sprintf("%s:%s@", url.QueryEscape(m.Username), url.QueryEscape(m.Password))
	}

	var query = ""
	if len(m.Options) > 0 {
		q := url.Values{}
		for key, value := range m.Options {
			q.Add(key, value)
		}
		query = "?" + q.Encode()
	}

	return fmt.Sprintf("mongodb://%s%s:%d/%s%s", auth, m.Host, m.Port, m.Database, query)
}

type RedisConfig struct {
	Host     string `env:"HOST, default=localhost"`
	Port     int    `env:"PORT, default=6379"`
	Password string `env:"PASSWORD"`
	Database int    `env:"DBNAME, default=0"`
}

type BotConfig struct {
	Token  string `env:"TOKEN, required"`
	ApiURL string `env:"API_URL, required"`
}
