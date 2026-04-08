# OrderFlow

> Event-driven order processing system with saga orchestration, Kafka event streaming, gRPC service communication, and distributed tracing.

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.4-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![Kafka](https://img.shields.io/badge/Apache_Kafka-3.7-231F20?logo=apachekafka&logoColor=white)](https://kafka.apache.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

---

## Problem

Modern e-commerce systems need to coordinate multiple services (orders, payments, inventory, shipping) while maintaining data consistency — without tight coupling or distributed transactions. A single order touches 4+ services, and any failure must trigger compensating actions across all of them.

## Solution

OrderFlow implements the **Saga Orchestration Pattern** to manage distributed transactions across microservices. An order saga coordinator drives the workflow: reserve inventory → process payment → confirm order → initiate shipping. If any step fails, compensating transactions roll back previous steps automatically.

## Architecture

```
┌─────────────┐     ┌──────────────────────────────────────────────────┐
│   Next.js    │     │              API Gateway (Go)                    │
│  Dashboard   │────▶│  REST → gRPC fan-out, auth, rate limiting       │
└─────────────┘     └──────────┬───────────────────────────────────────┘
                               │ gRPC
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                     ▼
   ┌─────────────┐    ┌──────────────┐     ┌──────────────┐
   │   Order      │    │   Payment    │     │  Inventory   │
   │   Service    │    │   Service    │     │   Service    │
   │   (Go)       │    │   (Go)       │     │   (Go)       │
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
          └──────────────────┘     └──────────────────┘
                     │
                     ▼
          ┌──────────────────┐
          │   Notification   │
          │   Service (Go)   │
          └──────────────────┘
```

## Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| **API Gateway** | Go + Chi router | High-performance HTTP→gRPC translation |
| **Services** | Go + gRPC | Type-safe inter-service communication |
| **Event Bus** | Apache Kafka | Durable, ordered event streaming |
| **Saga Engine** | Go (custom) | Orchestrates distributed transactions |
| **Databases** | PostgreSQL | ACID per service, database-per-service pattern |
| **Dashboard** | Next.js + TypeScript | Real-time order tracking UI |
| **Tracing** | OpenTelemetry + Jaeger | Distributed request tracing |
| **Containers** | Docker + Docker Compose | Local development & deployment |
| **CI/CD** | GitHub Actions | Automated testing & linting |

## Services

| Service | Responsibility | Port |
|---------|---------------|------|
| `api-gateway` | REST API, auth, rate limiting, gRPC fan-out | 8080 |
| `order-service` | Order CRUD, order state machine | 50051 |
| `payment-service` | Payment processing, refunds | 50052 |
| `inventory-service` | Stock management, reservations | 50053 |
| `saga-orchestrator` | Distributed transaction coordination | 50054 |
| `notification-service` | Email/webhook notifications | 50055 |
| `dashboard` | Real-time order tracking UI | 3000 |

## Getting Started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)
- [protoc](https://grpc.io/docs/protoc-installation/) (Protocol Buffers compiler)
- [buf](https://buf.build/docs/installation) (Protobuf tooling)

### Quick Start

```bash
# Clone the repository
git clone git@github.com:Nishchal45/orderflow.git
cd orderflow

# Start all infrastructure (Kafka, PostgreSQL, Jaeger)
docker compose up -d

# Run database migrations
make migrate-up

# Start all services
make run-all

# Start the dashboard
cd dashboard && npm install && npm run dev
```

The dashboard will be available at `http://localhost:3000` and the API at `http://localhost:8080`.

### API Examples

```bash
# Create an order
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_123",
    "items": [
      {"product_id": "prod_456", "quantity": 2, "price": 29.99}
    ]
  }'

# Get order status
curl http://localhost:8080/api/v1/orders/{order_id}

# List orders
curl http://localhost:8080/api/v1/orders?page=1&limit=20
```

## Project Structure

```
orderflow/
├── proto/                    # Protobuf definitions (shared contracts)
│   ├── order/
│   ├── payment/
│   ├── inventory/
│   └── saga/
├── services/
│   ├── api-gateway/          # REST → gRPC gateway
│   ├── order-service/        # Order management
│   ├── payment-service/      # Payment processing
│   ├── inventory-service/    # Stock management
│   ├── saga-orchestrator/    # Transaction coordination
│   └── notification-service/ # Notifications
├── dashboard/                # Next.js frontend
├── pkg/                      # Shared Go packages
│   ├── kafka/                # Kafka producer/consumer
│   ├── grpc/                 # gRPC interceptors
│   ├── tracing/              # OpenTelemetry setup
│   └── database/             # DB connection & migrations
├── deployments/
│   ├── docker/               # Dockerfiles per service
│   └── docker-compose.yml    # Local development stack
├── scripts/                  # Dev scripts & tooling
├── docs/                     # PRD, TRD, architecture docs
│   ├── PRD.md
│   ├── TRD.md
│   └── architecture/
├── .github/
│   └── workflows/            # CI/CD pipelines
├── Makefile                  # Build, run, test commands
└── buf.yaml                  # Protobuf configuration
```

## Development Workflow

This project follows a professional development workflow:

1. **Planning** — PRD → TRD → Architecture Design → Task Breakdown
2. **Branching** — `main` ← `develop` ← `feature/*`, `fix/*`, `chore/*`
3. **Commits** — [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `chore:`)
4. **Pull Requests** — Feature branch → PR with description → Review → Squash merge
5. **CI/CD** — Lint + Test + Build on every PR via GitHub Actions

## Documentation

- [Product Requirements Document (PRD)](./docs/PRD.md)
- [Technical Requirements Document (TRD)](./docs/TRD.md)
- [Architecture Decision Records](./docs/architecture/)

## License

[MIT](./LICENSE)
