package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort       string
	PostgresDSN   string
	RedisAddr     string
	KafkaBrokers  string
	JWTSecret     string
	BoostDuration time.Duration
}

func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		AppPort:      os.Getenv("APP_PORT"),
		PostgresDSN:  os.Getenv("POSTGRES_DSN"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
		KafkaBrokers: os.Getenv("KAFKA_BROKERS"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
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

	cfg.BoostDuration = 24 * time.Hour
	if raw := os.Getenv("BOOST_DURATION"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("invalid BOOST_DURATION %q: expected positive Go duration, e.g. 24h", raw)
		}
		cfg.BoostDuration = d
	}

	return cfg, nil
}
