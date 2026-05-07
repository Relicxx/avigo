package main

import (
	"log"

	"github.com/Relicxx/avigo/config"
	"github.com/Relicxx/avigo/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	pool, err := storage.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	log.Printf("db connected")
	log.Printf("starting on port %s", cfg.AppPort)
}
