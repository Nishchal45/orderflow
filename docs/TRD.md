# Technical Requirements Document (TRD)

**Project:** OrderFlow
**Author:** Nishchal Vekariya
**Date:** 2026-04-08
**Status:** Draft
**Version:** 1.0

---

## 1. System Architecture

OrderFlow follows a **microservices architecture** with:
- **API Gateway pattern** — single entry point for clients
- **Database-per-service** — each service owns its data
- **Event-driven communication** — Kafka for async messaging
- **Saga orchestration** — centralized transaction coordinator
- **gRPC for internal communication** — type-safe, fast, contract-first

### Architecture Style Decisions

| Decision | Choice | Alternatives Considered | Rationale |
|----------|--------|------------------------|-----------|
| Service communication | gRPC | REST, GraphQL | Type-safe contracts, code generation, streaming support, ~10x faster than REST for internal calls |
| Event bus | Apache Kafka | RabbitMQ, Redis Streams, NATS | Durable event log, replay capability, partition-based ordering, industry standard |
| Saga pattern | Orchestration | Choreography | Centralized control flow, easier to debug/trace, explicit failure handling |
| Language | Go | Node.js, Java, Rust | Fast compilation, built-in concurrency, small binaries, excellent gRPC support |
| API format | REST (external) | GraphQL | Simpler for CRUD operations, wider tooling support, recruiter-friendly |
| Database | PostgreSQL | MySQL, MongoDB | ACID transactions, JSON support, mature ecosystem |
| Frontend | Next.js + TypeScript | React SPA, Vue | SSR capability, TypeScript-first, App Router, excellent DX |
| Tracing | OpenTelemetry + Jaeger | Zipkin, Datadog | Vendor-neutral, CNCF standard, free/open-source |
| Containerization | Docker Compose | Kubernetes | Appropriate for local development, lower complexity |

## 2. Service Specifications

### 2.1 API Gateway (`services/api-gateway`)

**Responsibility:** HTTP REST entry point, request routing, gRPC fan-out

| Aspect | Detail |
|--------|--------|
| Language | Go 1.22 |
| Framework | Chi router |
| Port | 8080 |
| Protocol | HTTP/REST (inbound), gRPC (outbound to services) |

**Endpoints:**

```
POST   /api/v1/orders              → Create order
GET    /api/v1/orders              → List orders (paginated)
GET    /api/v1/orders/:id          → Get order by ID
GET    /api/v1/orders/:id/saga     → Get saga execution timeline
GET    /api/v1/orders/:id/events   → Get Kafka events for order
POST   /api/v1/orders/simulate     → Create order with failure simulation
GET    /api/v1/health              → Health check
GET    /api/v1/metrics             → Prometheus metrics
```

**Middleware stack:**
1. Request ID injection (X-Request-ID)
2. OpenTelemetry trace propagation
3. Structured logging (request/response)
4. Rate limiting (token bucket, 100 req/s)
5. CORS (allow dashboard origin)
6. Recovery (panic handler)

### 2.2 Order Service (`services/order-service`)

**Responsibility:** Order CRUD, order state machine management

| Aspect | Detail |
|--------|--------|
| Language | Go 1.22 |
| Protocol | gRPC |
| Port | 50051 |
| Database | PostgreSQL (database: `orderflow_orders`) |

**gRPC Methods:**
```protobuf
service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (Order);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc UpdateOrderStatus(UpdateOrderStatusRequest) returns (Order);
}
```

**Database Schema:**
```sql
CREATE TABLE orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id   VARCHAR(255) NOT NULL,
    status        VARCHAR(50) NOT NULL DEFAULT 'CREATED',
    total_amount  DECIMAL(10,2) NOT NULL,
    currency      VARCHAR(3) DEFAULT 'USD',
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE order_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL REFERENCES orders(id),
    product_id  VARCHAR(255) NOT NULL,
    quantity    INT NOT NULL,
    unit_price  DECIMAL(10,2) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE order_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID NOT NULL REFERENCES orders(id),
    event_type VARCHAR(100) NOT NULL,
    payload    JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_events_order_id ON order_events(order_id);
```

### 2.3 Payment Service (`services/payment-service`)

**Responsibility:** Simulated payment processing, refunds

| Aspect | Detail |
|--------|--------|
| Language | Go 1.22 |
| Protocol | gRPC |
| Port | 50052 |
| Database | PostgreSQL (database: `orderflow_payments`) |

**gRPC Methods:**
```protobuf
service PaymentService {
  rpc ProcessPayment(ProcessPaymentRequest) returns (PaymentResponse);
  rpc RefundPayment(RefundPaymentRequest) returns (PaymentResponse);
  rpc GetPayment(GetPaymentRequest) returns (Payment);
}
```

**Database Schema:**
```sql
CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL,
    amount          DECIMAL(10,2) NOT NULL,
    currency        VARCHAR(3) DEFAULT 'USD',
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    failure_reason  TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
```

**Simulation Logic:**
- Default: payment succeeds
- If `simulate_failure: true` → returns PAYMENT_FAILED with reason
- Random delay 100-500ms to simulate real processing

### 2.4 Inventory Service (`services/inventory-service`)

**Responsibility:** Stock management, reservations, releases

| Aspect | Detail |
|--------|--------|
| Language | Go 1.22 |
| Protocol | gRPC |
| Port | 50053 |
| Database | PostgreSQL (database: `orderflow_inventory`) |

**gRPC Methods:**
```protobuf
service InventoryService {
  rpc ReserveInventory(ReserveInventoryRequest) returns (InventoryResponse);
  rpc ReleaseInventory(ReleaseInventoryRequest) returns (InventoryResponse);
  rpc GetStock(GetStockRequest) returns (StockResponse);
}
```

**Database Schema:**
```sql
CREATE TABLE products (
    id          VARCHAR(255) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    stock       INT NOT NULL DEFAULT 0,
    reserved    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE reservations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL,
    product_id  VARCHAR(255) NOT NULL REFERENCES products(id),
    quantity    INT NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'RESERVED',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reservations_order_id ON reservations(order_id);
```

**Stock Logic:**
- `available = stock - reserved`
- Reserve: increment `reserved`, create reservation record
- Release: decrement `reserved`, update reservation status to `RELEASED`
- Check: reject if `available < requested quantity`

### 2.5 Saga Orchestrator (`services/saga-orchestrator`)

**Responsibility:** Coordinates distributed transactions, manages saga state

| Aspect | Detail |
|--------|--------|
| Language | Go 1.22 |
| Protocol | Kafka consumer/producer + gRPC client |
| Port | 50054 (health/metrics only) |
| Database | PostgreSQL (database: `orderflow_sagas`) |

**Database Schema:**
```sql
CREATE TABLE sagas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'STARTED',
    current_step    VARCHAR(100) NOT NULL DEFAULT 'RESERVE_INVENTORY',
    started_at      TIMESTAMPTZ DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    failure_reason  TEXT
);

CREATE TABLE saga_steps (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saga_id     UUID NOT NULL REFERENCES sagas(id),
    step_name   VARCHAR(100) NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    started_at  TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error       TEXT
);

CREATE INDEX idx_sagas_order_id ON sagas(order_id);
CREATE INDEX idx_saga_steps_saga_id ON saga_steps(saga_id);
```

**Saga Steps Definition:**
```
Step 1: RESERVE_INVENTORY
  → Action: Call InventoryService.ReserveInventory
  → Compensate: Call InventoryService.ReleaseInventory

Step 2: PROCESS_PAYMENT
  → Action: Call PaymentService.ProcessPayment
  → Compensate: Call PaymentService.RefundPayment

Step 3: CONFIRM_ORDER
  → Action: Call OrderService.UpdateOrderStatus(CONFIRMED)
  → Compensate: Call OrderService.UpdateOrderStatus(CANCELLED)

Step 4: NOTIFY
  → Action: Publish NOTIFICATION event
  → Compensate: None (notifications are fire-and-forget)
```

### 2.6 Notification Service (`services/notification-service`)

**Responsibility:** Consumes events, logs notifications (simulated)

| Aspect | Detail |
|--------|--------|
| Language | Go 1.22 |
| Protocol | Kafka consumer |
| Port | 50055 (health/metrics only) |

**Behavior:**
- Consumes `order.confirmed`, `order.cancelled` topics
- Logs notification (no real email/SMS)
- Publishes `NOTIFICATION_SENT` event for audit trail

### 2.7 Dashboard (`dashboard/`)

**Responsibility:** Real-time order tracking UI

| Aspect | Detail |
|--------|--------|
| Language | TypeScript |
| Framework | Next.js 14 (App Router) |
| Port | 3000 |
| State | React Query (TanStack Query) |
| Styling | Tailwind CSS |

**Pages:**
```
/                    → Order list with live status
/orders/:id          → Order detail with saga timeline
/create              → Create new order form
/events              → Live Kafka event stream viewer
```

**Real-time updates:** Polling every 2 seconds via React Query (SSE in future iteration)

## 3. Kafka Topics

| Topic | Producer | Consumer | Purpose |
|-------|----------|----------|---------|
| `order.created` | Order Service | Saga Orchestrator | New order placed |
| `inventory.reserved` | Inventory Service | Saga Orchestrator | Stock reserved |
| `inventory.released` | Inventory Service | Saga Orchestrator | Stock released (compensate) |
| `inventory.failed` | Inventory Service | Saga Orchestrator | Insufficient stock |
| `payment.completed` | Payment Service | Saga Orchestrator | Payment succeeded |
| `payment.failed` | Payment Service | Saga Orchestrator | Payment failed |
| `payment.refunded` | Payment Service | Saga Orchestrator | Payment refunded (compensate) |
| `order.confirmed` | Saga Orchestrator | Notification Service | Order saga completed |
| `order.cancelled` | Saga Orchestrator | Notification Service | Order saga failed/rolled back |
| `notification.sent` | Notification Service | (audit) | Notification delivered |

**Event Schema (standard envelope):**
```json
{
  "event_id": "uuid",
  "event_type": "ORDER_CREATED",
  "aggregate_id": "order-uuid",
  "timestamp": "2026-04-08T12:00:00Z",
  "version": 1,
  "payload": { ... },
  "metadata": {
    "trace_id": "opentelemetry-trace-id",
    "source": "order-service"
  }
}
```

## 4. Protobuf Structure

```
proto/
├── buf.yaml               # Buf workspace config
├── buf.gen.yaml            # Code generation config
├── order/
│   └── v1/
│       └── order.proto     # Order service contract
├── payment/
│   └── v1/
│       └── payment.proto   # Payment service contract
├── inventory/
│   └── v1/
│       └── inventory.proto # Inventory service contract
└── saga/
    └── v1/
        └── saga.proto      # Saga service contract
```

## 5. Shared Packages (`pkg/`)

| Package | Purpose |
|---------|---------|
| `pkg/kafka` | Kafka producer/consumer wrappers with OpenTelemetry |
| `pkg/grpc` | gRPC dial helpers, interceptors (logging, tracing) |
| `pkg/tracing` | OpenTelemetry initialization (Jaeger exporter) |
| `pkg/database` | PostgreSQL connection pool, migration runner |
| `pkg/logger` | Structured logging (zerolog) |
| `pkg/config` | Environment-based configuration loading |
| `pkg/events` | Event envelope types, serialization/deserialization |

## 6. Infrastructure (Docker Compose)

| Service | Image | Port(s) |
|---------|-------|---------|
| PostgreSQL | postgres:16 | 5432 |
| Apache Kafka | confluentinc/cp-kafka:7.6 | 9092 |
| Zookeeper | confluentinc/cp-zookeeper:7.6 | 2181 |
| Jaeger | jaegertracing/all-in-one:1.56 | 16686 (UI), 4318 (OTLP) |

## 7. CI/CD Pipeline (GitHub Actions)

### On every Pull Request:
```yaml
jobs:
  lint:     → golangci-lint, eslint (dashboard)
  test:     → go test ./..., npm test (dashboard)
  build:    → go build per service, next build
  proto:    → buf lint, buf breaking (detect breaking changes)
```

### On merge to main:
```yaml
jobs:
  docker:   → Build & push Docker images (future)
```

## 8. Configuration

All services use **environment variables** (12-factor app):

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=orderflow
DB_PASSWORD=orderflow
DB_NAME=orderflow_orders
DB_SSL_MODE=disable

# Kafka
KAFKA_BROKERS=localhost:9092

# Tracing
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Service
SERVICE_NAME=order-service
GRPC_PORT=50051
```

## 9. Error Handling Strategy

| Layer | Strategy |
|-------|----------|
| gRPC | Return proper gRPC status codes (NOT_FOUND, INVALID_ARGUMENT, INTERNAL) |
| Kafka | Retry with exponential backoff (3 attempts), then dead-letter topic |
| Database | Connection pooling with retry, transaction rollback on error |
| API Gateway | Map gRPC codes to HTTP codes, structured error JSON |
| Saga | Compensating transactions on any step failure |

**API Error Response Format:**
```json
{
  "error": {
    "code": "INSUFFICIENT_INVENTORY",
    "message": "Product prod_456 has only 3 units available",
    "request_id": "req_abc123",
    "trace_id": "trace_xyz789"
  }
}
```

## 10. Monitoring & Observability

| Signal | Tool | Purpose |
|--------|------|---------|
| Traces | OpenTelemetry → Jaeger | Request flow across services |
| Logs | zerolog (structured JSON) | Debugging, audit trail |
| Metrics | Prometheus (future) | Service health, latency, throughput |
| Events | Kafka topic viewer (dashboard) | Event flow visibility |

## 11. Security Considerations

Since this is a portfolio/demo project:
- No authentication (out of scope per PRD)
- No TLS for gRPC (local only)
- No secret management (env vars in docker-compose)
- CORS restricted to dashboard origin
- Input validation on all API endpoints
- SQL injection prevention via parameterized queries
