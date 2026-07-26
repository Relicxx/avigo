// Worker — консьюмер доменных событий Kafka. Агрегирует
// listing.created и boost.created в суточные счётчики в Redis.
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Relicxx/avigo/config"
	"github.com/Relicxx/avigo/internal/analytics"
	"github.com/Relicxx/avigo/internal/kafka"
	"github.com/Relicxx/avigo/internal/storage"
	"golang.org/x/sync/errgroup"
)

const consumerGroup = "avigo-analytics"

var topics = []string{"listing.created", "boost.created"}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisClient := storage.NewRedis(cfg)
	processor := analytics.NewProcessor(analytics.NewRedisCounter(redisClient))

	g, gctx := errgroup.WithContext(ctx)
	consumers := make([]*kafka.Consumer, 0, len(topics))
	for _, topic := range topics {
		c := kafka.NewConsumer(cfg.KafkaBrokers, consumerGroup, topic)
		consumers = append(consumers, c)
		g.Go(func() error {
			return c.Run(gctx, processor.Handle)
		})
	}

	slog.Info("worker started", "group", consumerGroup, "topics", topics, "brokers", cfg.KafkaBrokers)

	// Ждём сигнал остановки или фатальную ошибку консьюмера.
	<-gctx.Done()
	slog.Info("shutting down")

	for _, c := range consumers {
		if err := c.Close(); err != nil {
			slog.Error("close consumer failed", "error", err)
		}
	}
	if err := g.Wait(); err != nil {
		slog.Error("worker stopped with error", "error", err)
		os.Exit(1)
	}
	slog.Info("worker stopped")
}
