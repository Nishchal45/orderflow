package kafka

import (
	"context"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
)

// MessageHandler processes a consumed Kafka message.
type MessageHandler func(ctx context.Context, msg kafkago.Message) error

// Consumer wraps a Kafka reader for consuming events.
type Consumer struct {
	reader *kafkago.Reader
}

// NewConsumer creates a new Kafka consumer for a specific topic and group.
func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	return &Consumer{reader: reader}
}

// Consume starts reading messages and passes them to the handler.
// This blocks until the context is cancelled.
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("failed to read message: %w", err)
			}

			if err := handler(ctx, msg); err != nil {
				// Log error but continue consuming
				fmt.Printf("error handling message: %v\n", err)
			}
		}
	}
}

// Close gracefully shuts down the consumer.
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
