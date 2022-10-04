package config

import (
	"time"

	"github.com/pkg/errors"
	"github.com/vrischmann/envconfig"
)

// Config struct
type Config struct {
	HTTPBindPort    int           `envconfig:"default=8001"`
	GracefulTimeout time.Duration `envconfig:"default=10s"`
	SentryDSN       string        `envconfig:"optional"`
	Env             string        `envconfig:"default=local"`
}

// InitConfig func
func InitConfig(prefix string) (*Config, error) {
	config := &Config{}
	if err := envconfig.InitWithPrefix(config, prefix); err != nil {
		return nil, errors.Wrap(err, "init config failed")
	}

	return config, nil
}
