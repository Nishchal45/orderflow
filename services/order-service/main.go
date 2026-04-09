package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nishchal45/orderflow/pkg/config"
	"github.com/Nishchal45/orderflow/pkg/database"
	"github.com/Nishchal45/orderflow/pkg/kafka"
	"github.com/Nishchal45/orderflow/pkg/logger"
	"github.com/Nishchal45/orderflow/services/order-service/internal/handler"
	"github.com/Nishchal45/orderflow/services/order-service/internal/repository"
)

func main() {
	// Initialize logger
	log := logger.New("order-service")
	log.Info().Msg("starting order service")

	// Load config
	dbCfg := config.LoadDatabaseConfig("orderflow_orders")
	kafkaCfg := config.LoadKafkaConfig()

	// Connect to PostgreSQL
	db, err := database.New(database.Config{DSN: dbCfg.DSN()})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("connected to PostgreSQL")

	// Create Kafka producer
	producer := kafka.NewProducer(kafkaCfg.Brokers)
	defer producer.Close()
	log.Info().Msg("kafka producer ready")

	// Set up repository and handler
	repo := repository.NewOrderRepository(db)
	h := handler.NewOrderHandler(repo, producer, log)

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/orders", h.CreateOrder)
	mux.HandleFunc("GET /api/v1/orders", h.ListOrders)
	mux.HandleFunc("GET /api/v1/orders/{id}", h.GetOrder)
	mux.HandleFunc("PATCH /api/v1/orders/{id}/status", h.UpdateOrderStatus)
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"order-service"}`)
	})

	// Start server
	port := "8081"
	if p := os.Getenv("HTTP_PORT"); p != "" {
		port = p
	}

	go func() {
		log.Info().Str("port", port).Msg("order service listening")
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down order service")
}
