# OrderFlow

> Event-driven order processing system with saga orchestration, Kafka event streaming, and distributed tracing.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.4-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![Kafka](https://img.shields.io/badge/Apache_Kafka-3.7-231F20?logo=apachekafka&logoColor=white)](https://kafka.apache.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![CI](https://github.com/Nishchal45/orderflow/actions/workflows/ci.yml/badge.svg)](https://github.com/Nishchal45/orderflow/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

---

## Problem

Modern e-commerce systems need to coordinate multiple services (orders, payments, inventory) while maintaining data consistency. A single order touches 3+ services with separate databases. If payment succeeds but inventory is insufficient, you need automatic rollback across all services.

## Solution

OrderFlow implements the **Saga Orchestration Pattern** to manage distributed transactions. A central orchestrator coordinates: reserve inventory, process payment, confirm order. If any step fails, compensating transactions automatically undo previous steps.

## Architecture

```
┌─────────────┐     ┌──────────────────────────────────────────────────┐
│   Next.js    │     │              API Gateway (Go :8080)              │
│  Dashboard   │────▶│  REST endpoints, CORS, logging, request ID      │
│   (:3000)    │     └──────────┬───────────────────────────────────────┘
└─────────────┘                │
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                     ▼
   ┌─────────────┐    ┌──────────────┐     ┌──────────────┐
   │   Order      │    │   Payment    │     │  Inventory   │
   │   Service    │    │   Service    │     │   Service    │
   │   (:8081)    │    │   (:8083)    │     │   (:8082)    │
   └──────┬──────┘    └──────┬───────┘     └──────┬───────┘
          │                  │                     │
          ▼                  ▼                     ▼
   ┌─────────────┐    ┌──────────────┐     ┌──────────────┐
   │  PostgreSQL  │    │  PostgreSQL   │     │  PostgreSQL   │
   │  (orders)    │    │  (payments)   │     │  (inventory)  │
   └─────────────┘    └──────────────┘     └──────────────┘
          │                  │                     │
          └──────────┬───────┴─────────────────────┘
                     ▼
          ┌──────────────────┐     ┌──────────────────┐
          │   Apache Kafka   │────▶│  Saga             │
          │   Event Bus      │     │  Orchestrator     │
          │   (:9092)        │     │  (:8084)          │
          └──────────────────┘     └──────────────────┘
                     │
                     ▼
          ┌──────────────────┐
          │   Notification   │
          │   Service (:8085)│
          └──────────────────┘
```

## Key Features

- **Saga Orchestration** — Automatic distributed transaction coordination with compensation (rollback)
- **Event-Driven Architecture** — All services communicate via Kafka events
- **Database-per-Service** — Each microservice owns its own PostgreSQL database
- **Order State Machine** — Enforced valid status transitions (CREATED → CONFIRMED → SHIPPED)
- **Failure Simulation** — Toggle payment/inventory failures to test saga rollback
- **Real-time Dashboard** — Next.js UI with auto-refreshing order list and saga timeline
- **One-Command Setup** — `docker compose up` starts all infrastructure

## Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| **API Gateway** | Go + net/http | High-performance request routing and middleware |
| **Services** | Go | Fast compilation, single binary, built-in concurrency |
| **Event Bus** | Apache Kafka | Durable, ordered event streaming between services |
| **Saga Engine** | Go (custom) | Orchestrates distributed transactions with compensation |
| **Databases** | PostgreSQL | ACID transactions per service, database-per-service pattern |
| **Dashboard** | Next.js + TypeScript + Tailwind | Real-time order tracking with auto-refresh |
| **Containers** | Docker Compose | Local development stack with health checks |
| **CI/CD** | GitHub Actions | Automated build, test, and proto lint on every PR |

## Services

| Service | Port | Responsibility |
|---------|------|---------------|
| API Gateway | 8080 | REST entry point, request routing, middleware |
| Order Service | 8081 | Order CRUD, state machine, Kafka event publishing |
| Inventory Service | 8082 | Stock reservation/release with row-level locking |
| Payment Service | 8083 | Simulated payment processing with configurable failure |
| Saga Orchestrator | 8084 | Distributed transaction coordination and compensation |
| Notification Service | 8085 | Kafka consumer for order confirmations/cancellations |
| Dashboard | 3000 | Real-time order tracking UI |

## Saga Flow

### Happy Path

```
Customer places order
  → Order Service saves order, publishes ORDER_CREATED
  → Saga Orchestrator receives event, starts saga
    → Step 1: Reserve Inventory  ✅
    → Step 2: Process Payment    ✅
    → Step 3: Confirm Order      ✅
  → Order status: CONFIRMED
  → Notification: "Order confirmed!"
```

### Failure Path (Payment Fails)

```
Customer places order (simulate_payment_failure: true)
  → Step 1: Reserve Inventory  ✅
  → Step 2: Process Payment    ❌ (declined)
  → COMPENSATION:
    → Undo Step 1: Release Inventory  ✅ (stock restored)
    → Cancel Order
  → Order status: CANCELLED
  → Notification: "Order cancelled"
```

## Getting Started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker Desktop](https://docs.docker.com/get-docker/)

### Quick Start

```bash
# Clone
git clone git@github.com:Nishchal45/orderflow.git
cd orderflow

# Start infrastructure (Postgres, Kafka, Jaeger)
docker compose up -d

# Run database migrations
export PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"
docker exec -i orderflow-postgres psql -U orderflow -d orderflow_orders < services/order-service/migrations/001_create_orders.up.sql
docker exec -i orderflow-postgres psql -U orderflow -d orderflow_inventory < services/inventory-service/migrations/001_create_inventory.up.sql
docker exec -i orderflow-postgres psql -U orderflow -d orderflow_payments < services/payment-service/migrations/001_create_payments.up.sql
docker exec -i orderflow-postgres psql -U orderflow -d orderflow_sagas < services/saga-orchestrator/migrations/001_create_sagas.up.sql

# Start all services (each in a separate terminal, or use &)
DB_PORT=5433 go run ./services/order-service/ &
DB_PORT=5433 go run ./services/inventory-service/ &
DB_PORT=5433 go run ./services/payment-service/ &
DB_PORT=5433 go run ./services/saga-orchestrator/ &
go run ./services/api-gateway/ &

# Start dashboard
cd dashboard && npm install && npm run dev
```

Open `http://localhost:3000` to see the dashboard.

### Test the Saga

```bash
# Happy path — order gets confirmed automatically
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "nishchal",
    "items": [
      {"product_id": "burger", "quantity": 2, "unit_price": 9.99},
      {"product_id": "fries", "quantity": 1, "unit_price": 4.99}
    ]
  }'

# Failure path — payment fails, inventory rolls back
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "test",
    "items": [{"product_id": "pizza", "quantity": 1, "unit_price": 14.99}],
    "simulate_payment_failure": true
  }'
```

## Project Structure

```
orderflow/
├── proto/                    # Protobuf service contracts (API-first design)
│   ├── order/v1/             # gRPC migration planned — currently REST
│   ├── payment/v1/
│   ├── inventory/v1/
│   └── saga/v1/
├── services/
│   ├── api-gateway/          # REST entry point (:8080)
│   ├── order-service/        # Order CRUD + state machine (:8081)
│   ├── inventory-service/    # Stock management (:8082)
│   ├── payment-service/      # Payment processing (:8083)
│   ├── saga-orchestrator/    # Transaction coordination (:8084)
│   └── notification-service/ # Event consumer (:8085)
├── dashboard/                # Next.js frontend (:3000)
├── pkg/                      # Shared Go packages
│   ├── kafka/                # Producer/consumer wrappers
│   ├── database/             # PostgreSQL connection pool
│   ├── events/               # Event envelope + topic constants
│   ├── logger/               # Structured JSON logging
│   ├── config/               # Environment-based config
│   └── tracing/              # OpenTelemetry setup
├── docs/                     # PRD, TRD, architecture diagrams
├── docker-compose.yml        # Infrastructure (Postgres, Kafka, Jaeger)
├── Makefile                  # Build, test, run commands
└── .github/workflows/        # CI pipeline
```

## Kafka Topics

| Topic | Producer | Consumer | Purpose |
|-------|----------|----------|---------|
| `order.created` | Order Service | Saga Orchestrator | Triggers saga execution |
| `order.confirmed` | Saga Orchestrator | Notification Service | Order completed |
| `order.cancelled` | Saga Orchestrator | Notification Service | Order rolled back |
| `inventory.reserved` | Inventory Service | — | Stock reserved |
| `inventory.released` | Inventory Service | — | Stock released (compensation) |
| `payment.completed` | Payment Service | — | Payment succeeded |
| `payment.failed` | Payment Service | — | Payment declined |

## Development

```bash
make help          # Show all commands
make build         # Build all services
make test          # Run tests
make docker-up     # Start infrastructure
make docker-down   # Stop infrastructure
make proto         # Generate protobuf code
make clean         # Remove build artifacts
```

## Future Improvements

- [ ] **gRPC for internal communication** — Proto contracts are defined, migrate services from REST to gRPC for type-safe, faster inter-service calls
- [ ] **OpenTelemetry + Jaeger tracing** — Tracing package exists in `pkg/tracing`, wire into all services for end-to-end request tracing
- [ ] **Integration tests** — Docker-based tests that spin up full infrastructure and run saga flows
- [ ] **Circuit breaker** — Stop calling failing services, fail fast and recover
- [ ] **Kubernetes deployment** — Helm charts for production deployment

## Documentation

- [Product Requirements Document](./docs/PRD.md) — What we built and why
- [Technical Requirements Document](./docs/TRD.md) — How it's built
- [Architecture Diagrams](./docs/architecture/system-design.md) — Visual system design

## License

[MIT](./LICENSE)
