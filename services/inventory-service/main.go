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
	"github.com/Nishchal45/orderflow/services/inventory-service/internal/handler"
	"github.com/Nishchal45/orderflow/services/inventory-service/internal/repository"
)

func main() {
	log := logger.New("inventory-service")
	log.Info().Msg("starting inventory service")

	dbCfg := config.LoadDatabaseConfig("orderflow_inventory")
	kafkaCfg := config.LoadKafkaConfig()

	db, err := database.New(database.Config{DSN: dbCfg.DSN()})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("connected to PostgreSQL")

	producer := kafka.NewProducer(kafkaCfg.Brokers)
	defer producer.Close()

	repo := repository.NewInventoryRepository(db)
	h := handler.NewInventoryHandler(repo, producer, log)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/inventory/reserve", h.ReserveInventory)
	mux.HandleFunc("POST /api/v1/inventory/release", h.ReleaseInventory)
	mux.HandleFunc("GET /api/v1/inventory/{id}", h.GetStock)
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"inventory-service"}`)
	})

	port := "8082"
	if p := os.Getenv("HTTP_PORT"); p != "" {
		port = p
	}

	go func() {
		log.Info().Str("port", port).Msg("inventory service listening")
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down inventory service")
}
