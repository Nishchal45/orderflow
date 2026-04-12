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

	// Create reverse proxies to backend services
	orderProxy, err := proxy.NewServiceProxy(orderServiceURL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create order service proxy")
	}
	log.Info().Str("url", orderServiceURL).Msg("order service proxy ready")

	// Set up routes
	mux := http.NewServeMux()

	// Order routes → forward to Order Service
	mux.Handle("/api/v1/orders", orderProxy)
	mux.Handle("/api/v1/orders/", orderProxy)

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
