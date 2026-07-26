package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Relicxx/avigo/config"
	"github.com/Relicxx/avigo/internal/auth"
	"github.com/Relicxx/avigo/internal/boost"
	"github.com/Relicxx/avigo/internal/chat"
	"github.com/Relicxx/avigo/internal/health"
	"github.com/Relicxx/avigo/internal/kafka"
	"github.com/Relicxx/avigo/internal/listing"
	"github.com/Relicxx/avigo/internal/storage"
	"github.com/Relicxx/avigo/internal/user"
	"github.com/Relicxx/avigo/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is not set: refusing to start without a signing secret")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := storage.NewPostgres(ctx, cfg)
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
	r.Use(middleware.Metrics())
	r.Use(middleware.BodyLimit(1 << 20)) // 1 MiB

	// Liveness: процесс жив. Readiness: зависимости отвечают.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", health.Readiness(2*time.Second,
		health.Check{Name: "postgres", Probe: pool.Ping},
		health.Check{Name: "redis", Probe: func(ctx context.Context) error {
			return redisClient.Ping(ctx).Err()
		}},
	))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Auth-эндпоинты ограничены по частоте: login/register делают дорогой
	// bcrypt и являются мишенью перебора паролей.
	authLimit := middleware.RateLimit(storage.NewRateLimitStore(redisClient), "auth", 10, time.Minute)
	r.POST("/auth/register", authLimit, authHandler.Register)
	r.POST("/auth/login", authLimit, authHandler.Login)
	r.POST("/auth/refresh", authLimit, authHandler.Refresh)
	r.POST("/auth/logout", authLimit, authHandler.Logout)

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

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	// defer pool.Close() и producer.Close() выполняются после выхода из main.
}
