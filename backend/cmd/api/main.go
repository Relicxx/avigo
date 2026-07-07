package main

import (
	"log"

	"github.com/Relicxx/avigo/config"
	"github.com/Relicxx/avigo/internal/auth"
	"github.com/Relicxx/avigo/internal/boost"
	"github.com/Relicxx/avigo/internal/chat"
	"github.com/Relicxx/avigo/internal/kafka"
	"github.com/Relicxx/avigo/internal/listing"
	"github.com/Relicxx/avigo/internal/storage"
	"github.com/Relicxx/avigo/internal/user"
	"github.com/Relicxx/avigo/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pool, err := storage.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	userRepo := user.NewRepository(pool)
	authService := auth.NewService(userRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

	listingRepo := listing.NewRepository(pool)
	redisClient := storage.NewRedis(cfg)
	listingService := listing.NewService(listingRepo, producer, redisClient)
	listingHandler := listing.NewHandler(listingService)

	boostRepo := boost.NewRepository(pool)
	boostService := boost.NewService(boostRepo, producer)
	boostHandler := boost.NewHandler(boostService)

	r := gin.Default()
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/refresh", authHandler.Refresh)

	listings := r.Group("/listings")
	listings.GET("", listingHandler.List)
	listings.GET("/:id", listingHandler.GetByID)

	protected := listings.Group("")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	protected.POST("", listingHandler.Create)
	protected.PUT("/:id", listingHandler.Update)
	protected.DELETE("/:id", listingHandler.Delete)
	protected.POST("/:id/boost", boostHandler.Boost)

	chatRepo := chat.NewRepository(pool)
	chatService := chat.NewService(chatRepo)
	chatHandler := chat.NewHandler(chatService)

	// Чат доступен только аутентифицированным пользователям:
	// историю видят лишь участники переписки.
	messages := r.Group("")
	messages.Use(middleware.Auth(cfg.JWTSecret))
	messages.POST("/messages", chatHandler.Send)
	messages.GET("/listings/:id/messages", chatHandler.GetByListing)

	log.Printf("starting on port %s", cfg.AppPort)
	r.Run(":" + cfg.AppPort)
}
