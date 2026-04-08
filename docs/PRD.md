# Product Requirements Document (PRD)

**Project:** OrderFlow
**Author:** Nishchal Vekariya
**Date:** 2026-04-08
**Status:** Draft
**Version:** 1.0

---

## 1. Overview

OrderFlow is an event-driven order processing platform that demonstrates how modern e-commerce backends handle distributed transactions across multiple microservices. It uses the Saga Orchestration Pattern to coordinate order placement, payment processing, inventory management, and shipping — ensuring data consistency without distributed locks.

## 2. Problem Statement

In a monolithic e-commerce system, placing an order is a single database transaction:

```
BEGIN TRANSACTION
  → Insert order
  → Deduct payment
  → Reduce inventory
  → Create shipment
COMMIT
```

This breaks in a microservices world where each service owns its own database. You can't wrap 4 different databases in a single transaction. If payment succeeds but inventory deduction fails, you need to **automatically refund** the payment. This is the **distributed transaction problem**.

### What goes wrong without a saga:
- Payment charged but order never confirmed (customer charged for nothing)
- Inventory reserved but payment fails (phantom stock reduction)
- Order confirmed but shipping never triggered (order stuck)
- Partial failures leave the system in an inconsistent state

## 3. Target Users

| User | Need |
|------|------|
| **Hiring managers / Recruiters** | See a production-grade system design portfolio project |
| **Engineers reviewing the repo** | Understand saga patterns, event-driven architecture, gRPC |
| **Nishchal (author)** | Demonstrate backend engineering depth and system design skills |

## 4. Goals

| # | Goal | Measurable Outcome |
|---|------|--------------------|
| G1 | Demonstrate saga orchestration | Happy path order completes across 4 services in < 2 seconds |
| G2 | Show failure handling | Payment failure triggers automatic inventory rollback |
| G3 | Event-driven architecture | All inter-service communication via Kafka events |
| G4 | Observability | Every request traceable end-to-end via distributed tracing |
| G5 | Real-time visibility | Dashboard shows live order status updates |

## 5. Non-Goals (Out of Scope)

- Real payment gateway integration (we simulate payments)
- User authentication / authorization (no login system)
- Production deployment to cloud (local Docker only)
- Multi-region / multi-tenant support
- Real email delivery (log-based notifications only)
- Performance optimization for high throughput (this is a demo, not a load-tested system)

## 6. User Stories

### Epic 1: Order Placement
| ID | Story | Acceptance Criteria |
|----|-------|---------------------|
| US-1 | As a customer, I can place an order with multiple items | Order is created with status `PENDING`, returns order ID |
| US-2 | As a customer, I can view my order status | GET endpoint returns current order state and timeline |
| US-3 | As a customer, I can see all my orders | Paginated list of orders with status filters |

### Epic 2: Saga Orchestration
| ID | Story | Acceptance Criteria |
|----|-------|---------------------|
| US-4 | As the system, when an order is placed, the saga begins automatically | Saga coordinator picks up `ORDER_CREATED` event within 1 second |
| US-5 | As the system, inventory is reserved before payment is charged | Inventory reservation event precedes payment event in Kafka |
| US-6 | As the system, if payment fails, inventory is released | Compensating transaction fires within 2 seconds of failure |
| US-7 | As the system, if inventory is insufficient, order is rejected | Order status transitions to `REJECTED` with reason |
| US-8 | As the system, successful saga completes the order | Order status transitions: PENDING → CONFIRMED → SHIPPED |

### Epic 3: Dashboard
| ID | Story | Acceptance Criteria |
|----|-------|---------------------|
| US-9 | As a user, I can view all orders in real-time | Dashboard auto-updates without page refresh |
| US-10 | As a user, I can see the saga execution timeline | Visual step-by-step saga progress for each order |
| US-11 | As a user, I can trigger test orders from the dashboard | "Create Order" button with sample data |
| US-12 | As a user, I can simulate failures | Toggle to force payment/inventory failures |

### Epic 4: Observability
| ID | Story | Acceptance Criteria |
|----|-------|---------------------|
| US-13 | As a developer, I can trace a request across all services | Jaeger UI shows full trace with spans per service |
| US-14 | As a developer, I can see all Kafka events for an order | Event log shows topic, partition, timestamp, payload |

## 7. Order State Machine

```
                    ┌──────────┐
                    │ CREATED  │
                    └────┬─────┘
                         │ saga starts
                         ▼
                 ┌───────────────┐
                 │  INVENTORY    │
                 │  RESERVING    │
                 └───┬───────┬──┘
            success  │       │  failure
                     ▼       ▼
            ┌──────────┐  ┌──────────┐
            │ PAYMENT  │  │ REJECTED │
            │ PENDING  │  └──────────┘
            └──┬────┬──┘
          success│  │failure
               ▼    ▼
    ┌──────────┐  ┌──────────────┐
    │CONFIRMED │  │ ROLLING_BACK │
    └────┬─────┘  └──────┬───────┘
         │               │
         ▼               ▼
    ┌──────────┐  ┌──────────┐
    │ SHIPPED  │  │ CANCELLED│
    └──────────┘  └──────────┘
```

## 8. Saga Flow (Happy Path)

```
Step 1: Customer places order
        → Order Service creates order (status: CREATED)
        → Publishes ORDER_CREATED event to Kafka

Step 2: Saga Orchestrator picks up ORDER_CREATED
        → Sends "Reserve Inventory" command to Inventory Service
        → Order status: INVENTORY_RESERVING

Step 3: Inventory Service reserves stock
        → Publishes INVENTORY_RESERVED event
        → Order status: PAYMENT_PENDING

Step 4: Saga Orchestrator picks up INVENTORY_RESERVED
        → Sends "Process Payment" command to Payment Service

Step 5: Payment Service charges customer
        → Publishes PAYMENT_COMPLETED event
        → Order status: CONFIRMED

Step 6: Saga Orchestrator picks up PAYMENT_COMPLETED
        → Sends "Ship Order" command to Notification Service
        → Order status: SHIPPED
```

## 9. Saga Flow (Failure — Payment Fails)

```
Steps 1-3: Same as happy path (inventory reserved)

Step 4: Payment Service fails to charge
        → Publishes PAYMENT_FAILED event

Step 5: Saga Orchestrator picks up PAYMENT_FAILED
        → Sends "Release Inventory" compensating command
        → Order status: ROLLING_BACK

Step 6: Inventory Service releases reserved stock
        → Publishes INVENTORY_RELEASED event
        → Order status: CANCELLED
```

## 10. Success Criteria

| Criteria | Target |
|----------|--------|
| Happy path order completion | < 2 seconds end-to-end |
| Failure detection + rollback | < 3 seconds |
| Dashboard real-time update | < 500ms from event to UI |
| All services start with one command | `docker compose up` |
| Distributed trace coverage | 100% of requests traced |
| Code test coverage | > 70% for business logic |

## 11. Milestones

| Milestone | Deliverable | Target |
|-----------|-------------|--------|
| M1 | Project scaffolding + CI/CD + Docker | Week 1 |
| M2 | Proto definitions + Order Service + API Gateway | Week 2 |
| M3 | Inventory + Payment Services | Week 3 |
| M4 | Saga Orchestrator + Kafka integration | Week 4 |
| M5 | Notification Service + Dashboard | Week 5 |
| M6 | Distributed tracing + Testing + Polish | Week 6 |

## 12. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Kafka complexity for local dev | High setup friction | Docker Compose with pre-configured topics |
| Saga logic is hard to debug | Difficult to trace failures | Structured logging + Jaeger tracing |
| Scope creep (auth, real payments) | Delayed delivery | Strict non-goals list above |
| gRPC learning curve | Slow initial progress | Use buf for codegen, start with simple RPCs |
