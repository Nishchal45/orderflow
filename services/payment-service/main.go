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
	"github.com/Nishchal45/orderflow/services/payment-service/internal/handler"
	"github.com/Nishchal45/orderflow/services/payment-service/internal/repository"
)

func main() {
	log := logger.New("payment-service")
	log.Info().Msg("starting payment service")

	dbCfg := config.LoadDatabaseConfig("orderflow_payments")
	kafkaCfg := config.LoadKafkaConfig()

	db, err := database.New(database.Config{DSN: dbCfg.DSN()})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("connected to PostgreSQL")

	producer := kafka.NewProducer(kafkaCfg.Brokers)
	defer producer.Close()

	repo := repository.NewPaymentRepository(db)
	h := handler.NewPaymentHandler(repo, producer, log)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/payments/process", h.ProcessPayment)
	mux.HandleFunc("POST /api/v1/payments/refund", h.RefundPayment)
	mux.HandleFunc("GET /api/v1/payments/{order_id}", h.GetPayment)
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"payment-service"}`)
	})

	port := "8083"
	if p := os.Getenv("HTTP_PORT"); p != "" {
		port = p
	}

	go func() {
		log.Info().Str("port", port).Msg("payment service listening")
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down payment service")
}
