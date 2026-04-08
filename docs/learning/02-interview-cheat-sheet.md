# OrderFlow — Interview Cheat Sheet

> 30 questions an interviewer will ask about this project. Memorize the answers.

---

## The Elevator Pitch (30 seconds)

> "I built OrderFlow — an event-driven order processing system using Go microservices.
> It uses the Saga Orchestration Pattern to coordinate distributed transactions across
> 5 services. When you place an order, it reserves inventory, processes payment, and
> confirms the order — all through Kafka events. If any step fails, compensating
> transactions automatically roll back. I used gRPC for service communication,
> PostgreSQL with database-per-service, and OpenTelemetry for distributed tracing."

---

## Section 1: Architecture Questions

### Q1: Walk me through what happens when a user places an order.

**Answer:**
"The browser sends a POST request to the API Gateway, which translates it from REST
to gRPC and calls the Order Service. The Order Service saves the order to PostgreSQL
and publishes an ORDER_CREATED event to Kafka.

The Saga Orchestrator consumes this event and starts a saga — first it calls the
Inventory Service via gRPC to reserve stock. If successful, it calls the Payment
Service to charge the customer. If payment succeeds, it confirms the order and
notifies the customer.

If any step fails — say payment is declined — the orchestrator runs compensating
transactions in reverse: it releases the reserved inventory and cancels the order.
The entire flow is traced end-to-end via OpenTelemetry and visible in Jaeger."

### Q2: Why microservices instead of a monolith?

**Answer:**
"For a real production system, microservices let you:
1. Scale independently — if payments get 10x traffic on Black Friday, scale only that
2. Deploy independently — update inventory logic without redeploying payments
3. Use different tech per service if needed
4. Isolate failures — payment service crash doesn't take down order creation

For this project specifically, I wanted to demonstrate distributed systems knowledge —
saga patterns, event-driven architecture, and service communication."

### Q3: Why the Saga Orchestration pattern?

**Answer:**
"In a monolith, you use a single database transaction to ensure consistency. In
microservices with database-per-service, you can't. The Saga pattern solves this by
breaking a transaction into a sequence of local transactions with compensating
actions.

I chose orchestration over choreography because:
1. The flow is centralized in the orchestrator — easy to understand and debug
2. You can see the exact state of every saga step
3. Adding a new step (like fraud check) means changing one file, not multiple services
4. It integrates naturally with distributed tracing"

### Q4: What happens if the Saga Orchestrator itself crashes mid-saga?

**Answer:**
"The orchestrator persists saga state to PostgreSQL. Each step completion is saved
before moving to the next step. If it crashes and restarts, it reads the saga state
from the database and resumes from the last completed step. Combined with Kafka's
durable event log and consumer offsets, no events are lost."

### Q5: Why database-per-service?

**Answer:**
"If services share a database, they become coupled through the schema. A column
change in the orders table could break the payment service. With database-per-service,
each service owns its data and exposes it only through its API. This enables
independent deployment and schema evolution."

---

## Section 2: Technology Choice Questions

### Q6: Why Go?

**Answer:**
"Go is ideal for microservices because:
- Compiles to a single static binary — simple deployment, no runtime needed
- Built-in concurrency with goroutines — handles thousands of connections efficiently
- Excellent gRPC and Protobuf support
- Small Docker images (~10MB vs ~300MB for Java)
- Fast compilation — all 6 services build in seconds
- Strong standard library — less dependency bloat"

### Q7: Why Kafka over RabbitMQ?

**Answer:**
"Three reasons:
1. **Durability** — Kafka stores events in a log. You can replay them. RabbitMQ
   deletes messages after delivery.
2. **Ordering** — Kafka guarantees order within a partition. I partition by order ID,
   so all events for one order are processed in sequence.
3. **Throughput** — Kafka handles millions of events/sec. While I don't need that
   for a demo, it demonstrates I can work with production-grade infrastructure.

RabbitMQ would be fine for simple task queues, but for event sourcing and event-driven
architecture, Kafka is the industry standard."

### Q8: Why gRPC over REST for internal communication?

**Answer:**
"REST with JSON is ~10x slower than gRPC with Protobuf for internal calls because:
1. Protobuf is binary — smaller payloads, faster serialization
2. HTTP/2 — multiplexing, header compression
3. Strict contracts — the .proto file is the single source of truth. If I change
   a field type, the compiler catches it. With REST/JSON, you'd get a runtime error.
4. Code generation — client and server stubs are auto-generated from .proto files

I still use REST for the API Gateway because browsers can't speak gRPC directly."

### Q9: Why PostgreSQL over MongoDB?

**Answer:**
"OrderFlow deals with financial data — payments, inventory counts, order totals. These
require ACID transactions:
- **Atomicity**: Reserve inventory + create reservation must both succeed or both fail
- **Consistency**: Stock count can never go negative
- **Isolation**: Two concurrent orders can't reserve the same last item

PostgreSQL provides strong ACID guarantees. MongoDB has weaker consistency models and
is better suited for flexible schemas and document storage."

### Q10: Why OpenTelemetry for tracing?

**Answer:**
"OpenTelemetry is the CNCF standard for observability. It's vendor-neutral — I can
export traces to Jaeger today, Datadog tomorrow, without changing application code.
In a microservices system, distributed tracing is essential to understand request
flow and debug latency issues across service boundaries."

---

## Section 3: Design Pattern Questions

### Q11: Explain the event-driven architecture in this project.

**Answer:**
"Services communicate through events published to Kafka topics. When something
happens (order created, payment completed), the service publishes a fact — 'this
happened' — to a topic. Other interested services consume these events
asynchronously.

This decouples services — the Order Service doesn't know or care who reads its
events. It just publishes facts. This means:
1. Adding a new consumer (like analytics) requires zero changes to existing services
2. Services can be down temporarily — events wait in Kafka
3. The event log serves as an audit trail of everything that happened"

### Q12: What's the difference between a command and an event?

**Answer:**
"An **event** is a fact that something happened: 'OrderCreated', 'PaymentCompleted'.
It's past tense. The publisher doesn't care who reads it.

A **command** is a request to do something: 'ReserveInventory', 'ProcessPayment'.
It's directed at a specific service and expects a response.

In OrderFlow, the Saga Orchestrator sends **commands** via gRPC (synchronous), and
services publish **events** to Kafka (asynchronous) to report what happened."

### Q13: How do you handle idempotency?

**Answer:**
"Every event has a unique event_id. Consumers track processed event IDs and skip
duplicates. This is critical because Kafka guarantees at-least-once delivery — a
message might be delivered twice during consumer rebalancing. Without idempotency,
we'd charge a customer twice or reserve inventory twice."

### Q14: What is the event envelope pattern?

**Answer:**
"Every Kafka message follows a standard structure:
- event_id: unique identifier
- event_type: what happened (ORDER_CREATED)
- aggregate_id: which entity (order ID)
- timestamp: when it happened
- payload: the actual data
- metadata: trace_id, source service

This standardization means every consumer knows how to parse any event without
knowing the producer. It also enables distributed tracing across Kafka — the
trace_id in metadata connects Kafka events to the original request trace."

---

## Section 4: Failure Handling Questions

### Q15: What happens if payment fails?

**Answer:**
"The Saga Orchestrator receives a PAYMENT_FAILED event. It triggers compensating
transactions in reverse order:
1. Release the inventory reservation (undo Step 1)
2. Update the order status to CANCELLED

Each compensation is also tracked in the saga_steps table. The order transitions
through states: PAYMENT_PENDING → ROLLING_BACK → CANCELLED."

### Q16: What if a compensation itself fails?

**Answer:**
"This is the hardest problem in distributed systems. In OrderFlow, if a compensation
fails:
1. It retries with exponential backoff (3 attempts)
2. If still failing, the saga is marked as COMPENSATION_FAILED
3. It's logged with full context for manual intervention

In production, you'd add a dead-letter queue and an admin dashboard for operators
to manually resolve stuck sagas. This is a known tradeoff of the saga pattern."

### Q17: How do you prevent double-charging?

**Answer:**
"Three safeguards:
1. **Idempotency keys** — Each payment request includes the order ID. The Payment
   Service checks if a payment already exists for that order.
2. **Kafka consumer offsets** — Consumers commit their position after processing,
   preventing reprocessing.
3. **Database constraints** — Unique index on (order_id) in the payments table
   prevents duplicate rows."

---

## Section 5: Infrastructure Questions

### Q18: Explain your Docker Compose setup.

**Answer:**
"I use Docker Compose to run all infrastructure locally:
- PostgreSQL with an init script that creates 4 databases (one per service)
- Zookeeper + Kafka for event streaming
- A Kafka init container that pre-creates all 10 topics
- Jaeger for distributed tracing

Health checks ensure services start in dependency order — Kafka waits for Zookeeper,
topic init waits for Kafka. One command starts everything: `docker compose up -d`."

### Q19: Explain your CI/CD pipeline.

**Answer:**
"GitHub Actions runs 4 parallel jobs on every pull request:
1. **Lint** — golangci-lint catches code quality issues
2. **Test** — go test with race detector finds concurrency bugs
3. **Build** — compiles all 6 services to verify nothing is broken
4. **Proto Lint** — buf lint ensures protobuf contracts follow best practices

PRs can't merge if any job fails. This catches issues before they reach the
develop branch."

### Q20: What's your Git workflow?

**Answer:**
"Git Flow: `main` is production (protected), `develop` is integration, feature
branches for all work. Every change follows: create branch → write code →
push → create PR with description → CI passes → squash merge → close issue.

I use conventional commits (feat, fix, chore, docs, ci) for readable history
and link every PR to a GitHub issue for traceability."

---

## Section 6: "What Would You Do Differently" Questions

### Q21: What would you add for production?

**Answer:**
"1. **Authentication** — JWT tokens validated at the API Gateway
2. **Rate limiting** — Token bucket algorithm to prevent abuse
3. **Circuit breaker** — Stop calling a service that's consistently failing
4. **Retry with backoff** — Automatic retries for transient failures
5. **Health checks** — Kubernetes liveness and readiness probes
6. **Metrics** — Prometheus for service health, latency, throughput
7. **Secret management** — HashiCorp Vault instead of env vars
8. **TLS** — Encrypted gRPC and Kafka connections"

### Q22: How would you scale this?

**Answer:**
"1. **Horizontal scaling** — Run multiple instances of each service behind a load
   balancer. Kafka consumer groups distribute work automatically.
2. **Kafka partitioning** — Partition by order ID ensures all events for one order
   go to the same consumer (ordering guarantee).
3. **Database read replicas** — For read-heavy services like order listing.
4. **Caching** — Redis for frequently accessed data (product catalog).
5. **Kubernetes** — Container orchestration for auto-scaling based on load."

### Q23: What are the tradeoffs of your architecture?

**Answer:**
"1. **Complexity** — 6 services is harder to develop and debug than one monolith
2. **Eventual consistency** — Between saga steps, data is temporarily inconsistent
3. **Operational overhead** — Need to manage Kafka, multiple databases, tracing
4. **Network dependency** — Services communicate over the network (slower, can fail)
5. **Testing difficulty** — Integration tests need all services running

I accepted these tradeoffs because the project's goal is to demonstrate distributed
systems skills, not to be the simplest solution."

---

## Section 7: Quick-Fire Technical Questions

### Q24: What is ACID?
**Atomicity** (all or nothing), **Consistency** (valid state), **Isolation** (concurrent safety), **Durability** (survives crashes).

### Q25: What is eventual consistency?
Data across services will become consistent eventually, but not immediately. During a saga, inventory might be reserved but payment not yet processed — temporary inconsistency.

### Q26: What is a dead-letter queue?
A topic/queue where failed messages go after max retries. Prevents one bad message from blocking the entire consumer. Used for manual investigation.

### Q27: What is a consumer group?
Multiple Kafka consumers sharing the work of reading a topic. Each partition is assigned to one consumer in the group. Adding consumers = more parallelism.

### Q28: What is an idempotent operation?
An operation that produces the same result whether you execute it once or multiple times. Critical in distributed systems where messages can be delivered more than once.

### Q29: What is a circuit breaker?
A pattern that stops calling a failing service after N failures. Instead of wasting time on calls that will fail, it fails fast and recovers later. Like a fuse in electrical wiring.

### Q30: What is observability?
The ability to understand a system's internal state from its external outputs. Three pillars: **Logs** (what happened), **Metrics** (how much), **Traces** (the journey of a request).
