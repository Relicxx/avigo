package main

import (
	"log"

	"github.com/Relicxx/avigo/config"
	"github.com/Relicxx/avigo/internal/auth"
	"github.com/Relicxx/avigo/internal/storage"
	"github.com/Relicxx/avigo/internal/user"
	"github.com/gin-gonic/gin"
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

	userRepo := user.NewRepository(pool)
	authService := auth.NewService(userRepo, "secret")
	authHandler := auth.NewHandler(authService)

	r := gin.Default()
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)
	r.Run(":" + cfg.AppPort)
}
