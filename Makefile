# ============================================================
# OrderFlow — Development Commands
# ============================================================

SERVICES = api-gateway order-service payment-service inventory-service saga-orchestrator notification-service

.PHONY: help build test lint proto clean docker-up docker-down migrate-up migrate-down run-all

# Default target
help: ## Show this help message
	@echo "OrderFlow — Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ---- Build ----

build: ## Build all Go services
	@for svc in $(SERVICES); do \
		echo "Building $$svc..." && \
		go build -o bin/$$svc ./services/$$svc/ || exit 1; \
	done
	@echo "All services built → ./bin/"

build-%: ## Build a specific service (e.g., make build-order-service)
	@echo "Building $*..."
	@go build -o bin/$* ./services/$*/

# ---- Test ----

test: ## Run all tests
	@go test ./pkg/... ./services/... -race -count=1 -v

test-cover: ## Run tests with coverage report
	@go test ./pkg/... ./services/... -race -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report → coverage.html"

# ---- Lint ----

lint: ## Run linters
	@golangci-lint run ./...

# ---- Protobuf ----

proto: ## Generate Go code from protobuf definitions
	@cd proto && buf generate
	@echo "Protobuf code generated → ./gen/"

proto-lint: ## Lint protobuf files
	@cd proto && buf lint

proto-breaking: ## Check for breaking protobuf changes
	@cd proto && buf breaking --against '.git#branch=main,subdir=proto'

# ---- Docker ----

docker-up: ## Start infrastructure (Postgres, Kafka, Jaeger)
	@docker compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@docker compose ps

docker-down: ## Stop infrastructure
	@docker compose down

docker-clean: ## Stop infrastructure and remove volumes
	@docker compose down -v
	@echo "All data removed."

docker-logs: ## Tail infrastructure logs
	@docker compose logs -f

# ---- Database ----

migrate-up: ## Run all database migrations
	@for db in orders payments inventory sagas; do \
		echo "Migrating orderflow_$$db..." && \
		migrate -path services/$$(echo $$db | sed 's/s$$//')-service/migrations \
			-database "postgres://orderflow:orderflow@localhost:5432/orderflow_$$db?sslmode=disable" up 2>/dev/null || true; \
	done
	@echo "Migrations complete."

migrate-down: ## Rollback all database migrations
	@for db in orders payments inventory sagas; do \
		echo "Rolling back orderflow_$$db..." && \
		migrate -path services/$$(echo $$db | sed 's/s$$//')-service/migrations \
			-database "postgres://orderflow:orderflow@localhost:5432/orderflow_$$db?sslmode=disable" down -all 2>/dev/null || true; \
	done

# ---- Run ----

run-%: ## Run a specific service (e.g., make run-order-service)
	@go run ./services/$*/

run-all: ## Run all services (use with tmux or multiple terminals)
	@echo "Start each service in a separate terminal:"
	@echo ""
	@for svc in $(SERVICES); do \
		echo "  make run-$$svc"; \
	done
	@echo ""
	@echo "Or use: docker compose up (when Dockerfiles are added)"

# ---- Clean ----

clean: ## Remove build artifacts
	@rm -rf bin/ coverage.out coverage.html gen/
	@echo "Cleaned."
