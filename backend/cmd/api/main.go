package main

import (
	"log"
	"log/slog"
	"os"

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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

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

	redisClient := storage.NewRedis(cfg)

	userRepo := user.NewRepository(pool)
	tokenStore := auth.NewRedisTokenStore(redisClient)
	authService := auth.NewService(userRepo, tokenStore, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

	listingRepo := listing.NewRepository(pool)
	listingCache := listing.NewRedisCache(redisClient)
	listingService := listing.NewService(listingRepo, producer, listingCache)
	listingHandler := listing.NewHandler(listingService)

	boostRepo := boost.NewRepository(pool)
	boostService := boost.NewService(boostRepo, producer, cfg.BoostDuration)
	boostHandler := boost.NewHandler(boostService)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.BodyLimit(1 << 20)) // 1 MiB

	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/refresh", authHandler.Refresh)
	r.POST("/auth/logout", authHandler.Logout)

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
