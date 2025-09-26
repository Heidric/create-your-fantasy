package jwt

import (
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	Issuer   string
	Audience string
	Secret   string
}

func NewConfig() (*Config, error) {
	c := &Config{}

	_ = godotenv.Load()

	if err := envconfig.Init(c); err != nil {
		return nil, errors.Wrap(err, "init config")
	}

	return c, nil
}
