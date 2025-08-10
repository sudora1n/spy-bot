package types

import (
	"fmt"
	"net/url"
)

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
