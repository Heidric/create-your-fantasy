package config

import (
	"github.com/Heidric/create-your-fantasy/pkg/log"
	"github.com/Heidric/create-your-fantasy/pkg/pgx"
	"github.com/joho/godotenv"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	Logger        *log.Config
	DB            *pgx.Config
	ServerAddress string
}

func NewConfig() (*Config, error) {
	c := &Config{
		Logger: &log.Config{},
		DB:     &pgx.Config{},
	}

	_ = godotenv.Load()

	if err := envconfig.Init(c); err != nil {
		return nil, err
	}

	c.DB.SetDefault()
	c.Logger.SetDefault()

	return c, nil
}
