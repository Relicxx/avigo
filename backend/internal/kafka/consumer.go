package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// Handler обрабатывает одно сообщение топика.
type Handler func(ctx context.Context, topic string, payload []byte) error

// Consumer читает один топик в составе consumer group.
type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers, groupID, topic string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  strings.Split(brokers, ","),
			GroupID:  groupID,
			Topic:    topic,
			MinBytes: 1,
			MaxBytes: 10 << 20, // 10 MiB
			MaxWait:  time.Second,
		}),
	}
}

// Run читает сообщения до отмены контекста или закрытия консьюмера.
// Семантика at-least-once: offset коммитится только после обработки.
// Ошибка обработчика логируется, а сообщение пропускается (коммитится),
// чтобы одно битое событие не блокировало партицию навсегда.
func (c *Consumer) Run(ctx context.Context, handle Handler) error {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		if err := handle(ctx, m.Topic, m.Value); err != nil {
			slog.Error("handle message failed",
				"topic", m.Topic,
				"partition", m.Partition,
				"offset", m.Offset,
				"error", err,
			)
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.ErrClosedPipe) {
				return nil
			}
			return fmt.Errorf("commit offset: %w", err)
		}
	}
}

// Close останавливает чтение и разблокирует Run.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
