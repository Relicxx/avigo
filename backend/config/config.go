package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort      string
	PostgresDSN  string
	RedisAddr    string
	KafkaBrokers string
}

func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		AppPort:      os.Getenv("APP_PORT"),
		PostgresDSN:  os.Getenv("POSTGRES_DSN"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
		KafkaBrokers: os.Getenv("KAFKA_BROKERS"),
	}

	if cfg.AppPort == "" {
		cfg.AppPort = "8080"
	}

	if cfg.PostgresDSN == "" {
		cfg.PostgresDSN = "host=localhost user=avigo password=avigo dbname=avigo port=5432 sslmode=disable"
	}

	if cfg.RedisAddr == "" {
		cfg.RedisAddr = "localhost:6379"
	}

	if cfg.KafkaBrokers == "" {
		cfg.KafkaBrokers = "localhost:9092"
	}

	return cfg, nil
}
