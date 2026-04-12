package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nishchal45/orderflow/pkg/logger"
	"github.com/Nishchal45/orderflow/services/api-gateway/internal/middleware"
	"github.com/Nishchal45/orderflow/services/api-gateway/internal/proxy"
)

func main() {
	log := logger.New("api-gateway")
	log.Info().Msg("starting API gateway")

	// Backend service URLs (configurable via env vars)
	orderServiceURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8081")
	inventoryServiceURL := getEnv("INVENTORY_SERVICE_URL", "http://localhost:8082")
	paymentServiceURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8083")
	sagaServiceURL := getEnv("SAGA_SERVICE_URL", "http://localhost:8084")

	// Create reverse proxies to backend services
	orderProxy, err := proxy.NewServiceProxy(orderServiceURL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create order service proxy")
	}
	log.Info().Str("url", orderServiceURL).Msg("order service proxy ready")

	inventoryProxy, err := proxy.NewServiceProxy(inventoryServiceURL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create inventory service proxy")
	}

	paymentProxy, err := proxy.NewServiceProxy(paymentServiceURL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create payment service proxy")
	}

	sagaProxy, err := proxy.NewServiceProxy(sagaServiceURL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create saga service proxy")
	}

	// Set up routes
	mux := http.NewServeMux()

	// Order routes → forward to Order Service
	mux.Handle("/api/v1/orders", orderProxy)
	mux.Handle("/api/v1/orders/", orderProxy)

	// Inventory routes → forward to Inventory Service
	mux.Handle("/api/v1/inventory/", inventoryProxy)

	// Payment routes → forward to Payment Service
	mux.Handle("/api/v1/payments/", paymentProxy)

	// Saga routes → forward to Saga Orchestrator
	mux.Handle("/api/v1/saga/", sagaProxy)

	// Health check (gateway's own)
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"api-gateway"}`)
	})

	// Apply middleware (order matters: outermost runs first)
	var handler http.Handler = mux
	handler = middleware.CORS(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.Recovery(log)(handler)
	handler = middleware.Logger(log)(handler)

	// Start server
	port := getEnv("HTTP_PORT", "8080")
	go func() {
		log.Info().Str("port", port).Msg("api gateway listening")
		if err := http.ListenAndServe(":"+port, handler); err != nil {
			log.Fatal().Err(err).Msg("gateway failed")
		}
	}()

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down api gateway")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
