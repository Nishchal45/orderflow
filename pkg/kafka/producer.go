package kafka

import (
	"context"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer wraps a Kafka writer for publishing events.
type Producer struct {
	writer *kafkago.Writer
}

// NewProducer creates a new Kafka producer.
func NewProducer(brokers []string) *Producer {
	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Balancer:     &kafkago.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafkago.RequireOne,
	}

	return &Producer{writer: writer}
}

// Publish sends a message to the specified Kafka topic.
func (p *Producer) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	msg := kafkago.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", topic, err)
	}

	return nil
}

// Close gracefully shuts down the producer.
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
