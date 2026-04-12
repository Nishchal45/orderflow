package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nishchal45/orderflow/pkg/config"
	"github.com/Nishchal45/orderflow/pkg/database"
	"github.com/Nishchal45/orderflow/pkg/events"
	"github.com/Nishchal45/orderflow/pkg/kafka"
	"github.com/Nishchal45/orderflow/pkg/logger"
	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/engine"
	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/handler"
	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/model"
	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/repository"
	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	log := logger.New("saga-orchestrator")
	log.Info().Msg("starting saga orchestrator")

	// Load config
	dbCfg := config.LoadDatabaseConfig("orderflow_sagas")
	kafkaCfg := config.LoadKafkaConfig()

	// Connect to database
	db, err := database.New(database.Config{DSN: dbCfg.DSN()})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("connected to PostgreSQL")

	// Create Kafka producer (for publishing order.confirmed / order.cancelled)
	producer := kafka.NewProducer(kafkaCfg.Brokers)
	defer producer.Close()

	// Service URLs
	orderURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8081")
	inventoryURL := getEnv("INVENTORY_SERVICE_URL", "http://localhost:8082")
	paymentURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8083")

	// Create saga engine
	repo := repository.NewSagaRepository(db)
	sagaEngine := engine.NewSagaEngine(repo, producer, log, orderURL, inventoryURL, paymentURL)
	sagaHandler := handler.NewSagaHandler(repo, log)

	// Create Kafka consumer for ORDER_CREATED events
	consumer := kafka.NewConsumer(kafkaCfg.Brokers, events.TopicOrderCreated, "saga-orchestrator")
	defer consumer.Close()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Health check endpoint
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /api/v1/saga/{order_id}", sagaHandler.GetSagaByOrderID)
		mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"ok","service":"saga-orchestrator"}`)
		})
		log.Info().Msg("saga health endpoint on :8084")
		http.ListenAndServe(":8084", mux)
	}()

	// Start consuming ORDER_CREATED events
	log.Info().Msg("listening for ORDER_CREATED events on Kafka...")

	go func() {
		err := consumer.Consume(ctx, func(ctx context.Context, msg kafkago.Message) error {
			log.Info().Str("topic", msg.Topic).Msg("received event")

			// Parse the event envelope
			evt, err := events.Unmarshal(msg.Value)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal event")
				return nil // don't retry bad messages
			}

			// Extract order data from the event payload
			var orderData model.OrderData
			if err := evt.DecodePayload(&orderData); err != nil {
				log.Error().Err(err).Msg("failed to decode order data")
				return nil
			}

			log.Info().
				Str("order_id", orderData.ID).
				Str("customer", orderData.CustomerID).
				Float64("total", orderData.TotalAmount).
				Msg("processing order via saga")

			// Execute the saga!
			if err := sagaEngine.ExecuteSaga(ctx, orderData); err != nil {
				log.Error().Err(err).Str("order_id", orderData.ID).Msg("saga failed")
				// Don't return error — we handled it (compensated)
			}

			return nil
		})
		if err != nil && ctx.Err() == nil {
			log.Fatal().Err(err).Msg("kafka consumer failed")
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down saga orchestrator")
	cancel()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
