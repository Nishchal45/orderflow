package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nishchal45/orderflow/pkg/events"
	"github.com/Nishchal45/orderflow/pkg/kafka"
	"github.com/Nishchal45/orderflow/pkg/logger"
	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	log := logger.New("notification-service")
	log.Info().Msg("starting notification service")

	brokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}

	// Create consumers for confirmed and cancelled orders
	confirmedConsumer := kafka.NewConsumer(brokers, events.TopicOrderConfirmed, "notification-service")
	defer confirmedConsumer.Close()

	cancelledConsumer := kafka.NewConsumer(brokers, events.TopicOrderCancelled, "notification-service")
	defer cancelledConsumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Health endpoint
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"ok","service":"notification-service"}`)
		})
		log.Info().Msg("notification health endpoint on :8085")
		http.ListenAndServe(":8085", mux)
	}()

	// Consume ORDER_CONFIRMED events
	go func() {
		log.Info().Msg("listening for ORDER_CONFIRMED events...")
		confirmedConsumer.Consume(ctx, func(ctx context.Context, msg kafkago.Message) error {
			evt, err := events.Unmarshal(msg.Value)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal event")
				return nil
			}
			log.Info().
				Str("order_id", evt.AggregateID).
				Str("event", evt.EventType).
				Msg("NOTIFICATION: Order confirmed! Email sent to customer.")
			return nil
		})
	}()

	// Consume ORDER_CANCELLED events
	go func() {
		log.Info().Msg("listening for ORDER_CANCELLED events...")
		cancelledConsumer.Consume(ctx, func(ctx context.Context, msg kafkago.Message) error {
			evt, err := events.Unmarshal(msg.Value)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal event")
				return nil
			}
			log.Info().
				Str("order_id", evt.AggregateID).
				Str("event", evt.EventType).
				Msg("NOTIFICATION: Order cancelled. Apology email sent to customer.")
			return nil
		})
	}()

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down notification service")
	cancel()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
