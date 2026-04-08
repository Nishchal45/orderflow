# OrderFlow — Beginner's Guide to Every Technology

> You're building a project that uses 10+ technologies. This guide explains each one
> like you're 5, then like you're interviewing. No jargon without explanation.

---

## Table of Contents

1. [The Big Picture — What Are We Building?](#1-the-big-picture)
2. [What is a Microservice?](#2-microservices)
3. [What is Docker?](#3-docker)
4. [What is PostgreSQL?](#4-postgresql)
5. [What is Apache Kafka?](#5-kafka)
6. [What is gRPC and Protobuf?](#6-grpc)
7. [What is the Saga Pattern?](#7-saga)
8. [What is an API Gateway?](#8-api-gateway)
9. [What is Git Branching & PRs?](#9-git)
10. [What is CI/CD?](#10-cicd)
11. [What is Distributed Tracing (Jaeger)?](#11-tracing)
12. [What is a Makefile?](#12-makefile)
13. [What is Go (Golang)?](#13-golang)
14. [How It All Fits Together](#14-together)

---

## 1. The Big Picture — What Are We Building? {#1-the-big-picture}

### The Simple Version

Imagine you order a pizza online:

```
You click "Place Order"
   → The restaurant receives your order
   → Your credit card is charged
   → The kitchen checks if they have ingredients
   → They start making your pizza
   → You get a notification: "Your pizza is on the way!"
```

Now imagine **each of those steps is a separate computer program** running on a separate machine:
- Program 1: Takes orders
- Program 2: Charges credit cards
- Program 3: Checks ingredients
- Program 4: Coordinates everything
- Program 5: Sends notifications

That's what **OrderFlow** is. Five separate programs (called **microservices**) that talk to each other to process an order.

### Why Not Just One Program?

Good question. In a small pizza shop, one person does everything. But Amazon processes **4,000 orders per second**. One program can't handle that. So they split it:

| One Big Program (Monolith) | Many Small Programs (Microservices) |
|---|---|
| One team maintains everything | Different teams own different services |
| If payments break, everything breaks | If payments break, orders still get created |
| Hard to scale — scale everything or nothing | Scale only what's busy (payments on Black Friday) |
| Simple to start | Complex but powerful at scale |

**Interview answer:** "OrderFlow demonstrates a microservices architecture where each service owns its own database and communicates through events. I chose this pattern because it shows how real companies like Amazon and Uber handle distributed order processing."

---

## 2. What is a Microservice? {#2-microservices}

### Like You're 5

A microservice is like a **specialist doctor**. Instead of one doctor who does everything (general doctor), you have:
- A heart doctor (Order Service)
- A skin doctor (Payment Service)
- An eye doctor (Inventory Service)

Each doctor has their own office (database), their own phone number (port), and their own specialty. They send each other letters (events) when they need to coordinate.

### The Technical Version

A microservice is a small, independent application that:
1. **Does one thing well** (Single Responsibility)
2. **Has its own database** (no sharing tables with other services)
3. **Communicates over the network** (HTTP, gRPC, or messages)
4. **Can be deployed independently** (update payments without touching orders)

### In OrderFlow

| Service | What It Does | Its Database | Its Port |
|---------|-------------|-------------|----------|
| Order Service | Creates and tracks orders | `orderflow_orders` | 50051 |
| Payment Service | Charges and refunds money | `orderflow_payments` | 50052 |
| Inventory Service | Tracks product stock | `orderflow_inventory` | 50053 |
| Saga Orchestrator | Coordinates all the above | `orderflow_sagas` | 50054 |
| Notification Service | Sends alerts | (none) | 50055 |
| API Gateway | Front door for clients | (none) | 8080 |

### Interview Questions

**Q: What's the difference between a monolith and microservices?**
A: A monolith is one application with all features. Microservices split features into independent services, each with its own database. Microservices are more complex but allow independent scaling and deployment.

**Q: What's the biggest challenge with microservices?**
A: Data consistency. In a monolith, you use one database transaction. In microservices, each service has its own database, so you need patterns like Saga to maintain consistency across services.

**Q: How do microservices communicate?**
A: Two ways — synchronous (gRPC/REST: service A calls service B and waits for a response) and asynchronous (Kafka: service A publishes an event, service B reads it later).

---

## 3. What is Docker? {#3-docker}

### Like You're 5

Imagine you bake a cake at home. It comes out perfect. Now your friend asks for the recipe. They try it, but their oven is different, their flour is different, and the cake fails.

Docker is like putting **your entire kitchen** — oven, flour, recipe, everything — into a box. Your friend gets the exact same box. The cake is identical every time.

### The Technical Version

Docker is a tool that **packages your application with everything it needs** (operating system, libraries, config) into a container. A container is like a lightweight virtual machine.

Without Docker:
```
"It works on my machine" → Breaks on teammate's machine → Breaks in production
```

With Docker:
```
Same container everywhere → Works on my machine = works everywhere
```

### What You See in Docker Desktop

Open Docker Desktop on your Mac. You'll see 5 containers running:

| Container | What's Inside | Port |
|-----------|--------------|------|
| `orderflow-postgres` | PostgreSQL database | 5432 |
| `orderflow-zookeeper` | Kafka's coordinator | 2181 |
| `orderflow-kafka` | Event streaming engine | 9092 |
| `orderflow-kafka-init` | Script that creates topics (then stops) | — |
| `orderflow-jaeger` | Tracing dashboard | 16686 |

### Key Docker Concepts

| Concept | What It Is | Analogy |
|---------|-----------|---------|
| **Image** | A blueprint/recipe | A cake recipe |
| **Container** | A running instance of an image | An actual baked cake |
| **Dockerfile** | Instructions to build an image | The recipe written down |
| **docker-compose.yml** | Define multiple containers together | A menu with multiple recipes |
| **Volume** | Persistent storage | A notebook that survives even if you throw away the cake |
| **Port mapping** | Connect container port to your machine | Forwarding a phone number |

### The docker-compose.yml Explained

```yaml
# "I want a PostgreSQL database"
postgres:
  image: postgres:16-alpine    # Use this pre-built recipe
  ports:
    - '5432:5432'              # My machine's port 5432 → container's port 5432
  volumes:
    - postgres_data:/var/lib/postgresql/data  # Keep data even if container restarts
  healthcheck:                 # How to know it's ready
    test: pg_isready
```

### Commands You Should Know

```bash
docker compose up -d      # Start all containers (background)
docker compose down        # Stop all containers
docker compose ps          # See what's running
docker compose logs kafka  # See logs for one container
docker exec -it orderflow-postgres psql -U orderflow  # Get inside the database
```

### Interview Questions

**Q: What is Docker?**
A: Docker is a containerization platform that packages applications with their dependencies into isolated containers. This ensures consistent behavior across development, testing, and production environments.

**Q: What's the difference between a container and a virtual machine?**
A: Containers share the host OS kernel, making them lightweight (MB, starts in seconds). VMs include a full OS (GB, starts in minutes). Containers are ideal for microservices.

**Q: What is docker-compose?**
A: Docker Compose defines and runs multi-container applications. You describe all services in a YAML file and start them with one command. It handles networking, volumes, and dependencies.

**Q: What is a Docker volume?**
A: A volume persists data outside the container's lifecycle. Without a volume, data is lost when a container stops. With a volume, the database data survives restarts.

---

## 4. What is PostgreSQL? {#4-postgresql}

### Like You're 5

PostgreSQL is a **filing cabinet**. You have folders (tables), and each folder has papers (rows) with specific information (columns).

```
Filing Cabinet: orderflow_orders
└── Folder: orders
    ├── Paper 1: {id: abc, customer: "John", status: "CONFIRMED", total: $59.99}
    ├── Paper 2: {id: def, customer: "Jane", status: "PENDING", total: $29.99}
    └── Paper 3: {id: ghi, customer: "Bob", status: "CANCELLED", total: $99.99}
```

### Why PostgreSQL?

It's a **relational database** — data is stored in tables with relationships between them:

```
orders table          order_items table
┌────┬──────────┐    ┌────┬──────────┬──────────┐
│ id │ customer │    │ id │ order_id │ product  │
├────┼──────────┤    ├────┼──────────┼──────────┤
│ 1  │ John     │◄───│ 1  │ 1        │ Laptop   │
│    │          │◄───│ 2  │ 1        │ Mouse    │
│ 2  │ Jane     │◄───│ 3  │ 2        │ Phone    │
└────┴──────────┘    └────┴──────────┴──────────┘
```

Order 1 has 2 items (Laptop + Mouse). The `order_id` column in `order_items` **references** the `orders` table.

### Why "Database Per Service"?

In OrderFlow, each service has its **own database**:
- Order Service → `orderflow_orders` (only orders data)
- Payment Service → `orderflow_payments` (only payment data)
- Inventory Service → `orderflow_inventory` (only stock data)

Why? Because if Payment Service could read the Orders database directly, they become **coupled** — a change in orders breaks payments. Separate databases force services to communicate through APIs.

### Interview Questions

**Q: Why PostgreSQL over MongoDB?**
A: OrderFlow deals with financial data (payments, inventory counts) that requires ACID transactions — atomicity, consistency, isolation, durability. PostgreSQL guarantees these. MongoDB is better for unstructured/flexible data.

**Q: What is database-per-service pattern?**
A: Each microservice owns its private database. No other service can access it directly. This ensures loose coupling — services can only interact through APIs or events.

---

## 5. What is Apache Kafka? {#5-kafka}

### Like You're 5

Imagine a **bulletin board** at a school:

1. A teacher **posts** a notice: "Field trip on Friday"
2. Any student can **read** the notice
3. The notice **stays** on the board (it doesn't disappear after being read)
4. New students who join later can still read it

Kafka is that bulletin board, but for computer programs.

### The Technical Version

Kafka is a **distributed event streaming platform**. Services publish **events** (things that happened), and other services consume them.

```
Order Service publishes:  "An order was created!"
                              ↓
                    ┌── Kafka Topic: order.created ──┐
                    │                                 │
                    │  Event 1: Order ABC created     │
                    │  Event 2: Order DEF created     │
                    │  Event 3: Order GHI created     │
                    │                                 │
                    └─────────────────────────────────┘
                              ↓
Saga Orchestrator reads: "Oh, a new order! Let me start processing it."
```

### Key Concepts

| Concept | What It Is | Analogy |
|---------|-----------|---------|
| **Topic** | A category of events | A bulletin board for one subject (math, sports) |
| **Producer** | Service that publishes events | The teacher posting a notice |
| **Consumer** | Service that reads events | Students reading the notice |
| **Event** | A fact that something happened | The notice itself |
| **Partition** | A sub-section of a topic (for parallelism) | Multiple bulletin boards for the same subject |
| **Consumer Group** | Multiple consumers sharing work | 3 students splitting 30 notices to read faster |
| **Offset** | Your position in the topic | A bookmark — "I've read up to notice #5" |

### Our 10 Kafka Topics

```
order.created       ← "A new order was placed"
order.confirmed     ← "The order was successfully processed"
order.cancelled     ← "The order was rolled back"
inventory.reserved  ← "Stock was reserved for the order"
inventory.released  ← "Stock was released (rollback)"
inventory.failed    ← "Not enough stock"
payment.completed   ← "Payment went through"
payment.failed      ← "Payment was declined"
payment.refunded    ← "Payment was refunded (rollback)"
notification.sent   ← "Customer was notified"
```

### Why Kafka Instead of Direct Calls?

```
WITHOUT Kafka (tight coupling):
Order Service → calls Payment Service directly
               → if Payment Service is down, Order Service crashes too

WITH Kafka (loose coupling):
Order Service → publishes event to Kafka
               → Payment Service is down? Event waits in Kafka
               → Payment Service comes back → reads the event → processes it
```

Kafka acts as a **buffer**. Services don't need to be running at the same time.

### Interview Questions

**Q: What is Kafka?**
A: Kafka is a distributed event streaming platform. It acts as a durable message bus where services publish events and other services consume them asynchronously. Events are stored durably and can be replayed.

**Q: Why Kafka over RabbitMQ?**
A: Kafka stores events durably (like a log), so you can replay them. RabbitMQ deletes messages after they're consumed. Kafka also guarantees ordering within a partition and handles much higher throughput.

**Q: What's the difference between synchronous and asynchronous communication?**
A: Synchronous (gRPC/REST) — Service A calls B and waits for a response. If B is down, A fails. Asynchronous (Kafka) — A publishes an event and continues. B processes it later. More resilient but harder to coordinate.

---

## 6. What is gRPC and Protobuf? {#6-grpc}

### Like You're 5

When you call a friend, you speak English. They understand because you both agree on the language.

**gRPC** is a way for programs to call each other. **Protobuf** is the language they agree on.

### REST vs gRPC

You probably know REST APIs (like `GET /api/users`). gRPC is a different approach:

| | REST | gRPC |
|---|---|---|
| **Format** | JSON (text) | Protobuf (binary) |
| **Speed** | Slower (parsing text) | ~10x faster (compact binary) |
| **Contract** | Loose (documentation) | Strict (.proto file = contract) |
| **Code generation** | Manual | Automatic (generate client/server from .proto) |
| **Best for** | Public APIs (browsers) | Internal service-to-service |

### The .proto File Explained

```protobuf
// This is a CONTRACT. Both sides agree on this.

service OrderService {
  // "You can call CreateOrder and I'll return an Order"
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);

  // "You can call GetOrder and I'll return an Order"
  rpc GetOrder(GetOrderRequest) returns (Order);
}

// What a request looks like
message CreateOrderRequest {
  string customer_id = 1;           // Field #1: who's ordering
  repeated CreateOrderItem items = 2; // Field #2: list of items
}

// What an order looks like
message Order {
  string id = 1;
  string customer_id = 2;
  OrderStatus status = 3;
  double total_amount = 4;
}
```

From this one file, tools **automatically generate**:
- Go server code (handles incoming calls)
- Go client code (makes outgoing calls)
- Type checking (can't send a string where a number is expected)

### Why gRPC for Internal Services?

```
Client (browser) → REST (JSON) → API Gateway → gRPC (fast binary) → Order Service
                                               → gRPC → Payment Service
                                               → gRPC → Inventory Service
```

External clients use REST (browsers understand JSON). Internal services use gRPC (faster, type-safe).

### Interview Questions

**Q: What is gRPC?**
A: gRPC is a high-performance RPC framework from Google. Services define their API in .proto files, and code is auto-generated for clients and servers. It uses Protocol Buffers for fast binary serialization and HTTP/2 for transport.

**Q: Why gRPC over REST for microservices?**
A: gRPC is ~10x faster due to binary serialization, provides strict contracts via .proto files (catching errors at compile time, not runtime), and supports streaming. REST is better for public-facing APIs that browsers consume.

---

## 7. What is the Saga Pattern? {#7-saga}

### Like You're 5

You're planning a party. You need to:
1. Book a venue
2. Order a cake
3. Hire a DJ

You book the venue (Step 1 ✅), order the cake (Step 2 ✅), but the DJ cancels (Step 3 ❌).

Now you need to **undo everything**:
- Cancel the cake order (undo Step 2)
- Cancel the venue booking (undo Step 1)

This "do steps in order, undo them in reverse if something fails" is the **Saga Pattern**.

### The Technical Version

In a monolith, you'd use a database transaction:
```sql
BEGIN TRANSACTION
  INSERT INTO orders ...
  UPDATE payments ...
  UPDATE inventory ...
COMMIT (all succeed) or ROLLBACK (all undo)
```

In microservices, each service has its **own database**. You can't wrap 3 different databases in one transaction. So you use a Saga:

```
SAGA: Place Order

Step 1: Reserve Inventory    ✅ Success → Continue
Step 2: Process Payment      ❌ Failure!

COMPENSATE (undo in reverse):
Undo Step 1: Release Inventory   ✅ Stock restored
Result: Order CANCELLED
```

### Two Types of Saga

| Choreography | Orchestration (what we use) |
|---|---|
| Each service decides what to do next | One coordinator tells services what to do |
| Like a dance — everyone knows the steps | Like an orchestra — conductor directs everyone |
| Simple but hard to track | More code but easy to debug and trace |
| No single point of control | Saga Orchestrator is the controller |

### Our Saga Flow

```
HAPPY PATH (everything works):

  Customer clicks "Place Order"
       │
       ▼
  ┌─ Step 1: Reserve Inventory ──── ✅ "We have the items" ──┐
  │                                                            │
  │  ┌─ Step 2: Process Payment ──── ✅ "Card charged" ──┐    │
  │  │                                                     │    │
  │  │  ┌─ Step 3: Confirm Order ─── ✅ "Order confirmed" │    │
  │  │  │                                                  │    │
  │  │  │  ┌─ Step 4: Notify ──────── ✅ "Email sent"     │    │
  │  │  │  │                                               │    │
  └──┴──┴──┴── ORDER COMPLETE ─────────────────────────────┘    │
                                                                 │
FAILURE PATH (payment fails):                                    │
                                                                 │
  ┌─ Step 1: Reserve Inventory ──── ✅ "We have items" ────┐    │
  │                                                          │    │
  │  ┌─ Step 2: Process Payment ──── ❌ "Card declined!" ──┘    │
  │  │                                                            │
  │  │  COMPENSATION STARTS (reverse order):                      │
  │  │  ┌─ Undo Step 1: Release Inventory ── ✅ Stock restored   │
  │  │  │                                                         │
  └──┴──┴── ORDER CANCELLED ──────────────────────────────────────┘
```

### Interview Questions

**Q: What is the Saga pattern?**
A: A saga is a sequence of local transactions across multiple services. Each step has a compensating transaction. If any step fails, previously completed steps are undone in reverse order. It maintains data consistency without distributed locks.

**Q: Why Saga over two-phase commit (2PC)?**
A: 2PC locks resources across services until all agree — this blocks everything and doesn't scale. Saga uses compensating transactions instead of locks, so services stay available. The tradeoff is eventual consistency instead of immediate consistency.

**Q: What's the difference between saga choreography and orchestration?**
A: Choreography — each service publishes events and reacts to others' events. No central coordinator. Simple but hard to debug. Orchestration — a central coordinator tells each service what to do and handles failures. Easier to understand and trace.

---

## 8. What is an API Gateway? {#8-api-gateway}

### Like You're 5

A hotel has a **front desk**. Guests don't walk directly into the kitchen or the laundry room. They go to the front desk, and the front desk routes their request to the right department.

The API Gateway is the front desk. The browser (guest) talks to the gateway, and the gateway routes requests to the right microservice.

### Why Not Let Clients Talk Directly to Services?

```
WITHOUT Gateway:
Browser → Order Service (port 50051)
Browser → Payment Service (port 50052)
Browser → Inventory Service (port 50053)
Problem: Client needs to know all ports, handle different protocols

WITH Gateway:
Browser → API Gateway (port 8080) → routes to the right service
Client only knows one address!
```

### What Our Gateway Does

```
Browser sends:  POST http://localhost:8080/api/v1/orders
                              │
                    API Gateway receives it
                              │
                    Adds: Request ID, Logging, CORS
                              │
                    Converts REST → gRPC
                              │
                    Sends to: Order Service on port 50051
                              │
                    Gets response, converts gRPC → JSON
                              │
Browser receives: { "id": "abc", "status": "CREATED" }
```

### Interview Questions

**Q: What is an API Gateway?**
A: An API Gateway is a single entry point for all client requests. It handles routing, authentication, rate limiting, request/response transformation, and cross-cutting concerns like logging and tracing. Clients only interact with one endpoint.

---

## 9. What is Git Branching & Pull Requests? {#9-git}

### Like You're 5

Imagine writing a book with friends:
- The **published book** is the `main` branch (readers see this)
- The **draft** is the `develop` branch (your working copy)
- When you write a new chapter, you make a **photocopy** of the draft (feature branch)
- You write your chapter on the photocopy, not the draft
- When done, you show it to your friend for **review** (Pull Request)
- Friend says "looks good" → you paste it into the draft (merge)

### The Workflow

```
main (published book — never touch directly)
  │
  └── develop (working draft)
        │
        ├── feature/4-go-scaffolding      ← I worked on this
        │     └── PR #37 → merged to develop
        │
        ├── feature/5-docker-compose       ← Then this
        │     └── PR #38 → merged to develop
        │
        └── feature/6-protobuf            ← Then this
              └── PR #39 → merged to develop
```

### What's a Pull Request (PR)?

A PR is a **formal request** to merge your code. It includes:
1. **Title**: What you did (`feat(proto): add protobuf definitions`)
2. **Description**: Why and how
3. **Diff**: Line-by-line code changes (reviewers read this)
4. **Link to issue**: `Closes #6` (auto-closes the issue when merged)

### Why Not Just Push to Main?

| Direct Push | Pull Request |
|---|---|
| No review — bugs sneak in | Someone reviews before merge |
| No history of decisions | PR description explains why |
| Hard to revert | Revert one PR without affecting others |
| No CI check | CI runs tests before merge is allowed |

### Interview Questions

**Q: What branching strategy do you use?**
A: Git Flow — `main` for production, `develop` for integration, feature branches for individual work. All changes go through pull requests with code review before merging.

**Q: What are conventional commits?**
A: A standard format for commit messages: `type(scope): description`. Types include `feat`, `fix`, `chore`, `docs`, `ci`. This enables automated changelogs and makes history readable.

---

## 10. What is CI/CD? {#10-cicd}

### Like You're 5

Before submitting homework, your mom checks it:
- Spelling mistakes? (Lint)
- Answers correct? (Test)
- All pages present? (Build)

CI/CD is an **automatic homework checker** that runs every time you push code.

### CI = Continuous Integration

Every time you create a PR, **GitHub Actions** automatically:
1. **Lint** — checks code style (no typos, follow rules)
2. **Test** — runs all tests (does the code work?)
3. **Build** — compiles everything (does it even compile?)
4. **Proto Lint** — checks protobuf files (are the contracts valid?)

If any step fails → ❌ you can't merge → fix it first.

### CD = Continuous Deployment (we don't have this yet)

After merging to `main`, automatically deploy to production. We're not doing this because OrderFlow is a local project.

### Interview Questions

**Q: What is CI/CD?**
A: CI (Continuous Integration) automatically runs tests and checks on every code change. CD (Continuous Deployment) automatically deploys passing code to production. Together they catch bugs early and enable fast, safe releases.

---

## 11. What is Distributed Tracing? {#11-tracing}

### Like You're 5

You order a package on Amazon. You get a **tracking number**. You can see:
- Package picked up (warehouse)
- In transit (truck)
- At local facility
- Delivered

Distributed tracing is a **tracking number for a request**. When someone creates an order, you can trace it through:
- API Gateway (received request)
- Order Service (saved to database)
- Kafka (published event)
- Saga Orchestrator (started processing)
- Payment Service (charged card)
- and so on...

### What You See in Jaeger

Open `http://localhost:16686` in your browser. Once services are running, you'll see something like:

```
Trace: Create Order (total: 1.8s)
│
├── API Gateway: POST /api/v1/orders         (200ms)
│   └── Order Service: CreateOrder gRPC      (150ms)
│       └── PostgreSQL: INSERT               (50ms)
│
├── Saga Orchestrator: Process ORDER_CREATED (100ms)
│   └── Inventory Service: ReserveInventory  (200ms)
│       └── PostgreSQL: UPDATE               (30ms)
│
├── Payment Service: ProcessPayment          (400ms)
│   └── PostgreSQL: INSERT                   (25ms)
│
└── Notification Service: Send               (50ms)
```

You can see **exactly** where time is spent and where failures happen.

### Interview Questions

**Q: What is distributed tracing?**
A: Distributed tracing tracks a request as it flows through multiple services. Each service adds a "span" to the trace. Tools like Jaeger visualize the full journey, showing latency per service and where failures occur. We use OpenTelemetry as the standard.

---

## 12. What is a Makefile? {#12-makefile}

### Like You're 5

Instead of remembering 10 long commands, you create **shortcuts**:

| Instead of typing... | You type... |
|---|---|
| `docker compose up -d` | `make docker-up` |
| `go test ./pkg/... ./services/... -race` | `make test` |
| `go build -o bin/order-service ./services/order-service/` | `make build` |

### Interview Questions

**Q: Why use a Makefile?**
A: It standardizes development commands. A new developer runs `make help` to see all available commands. No need to memorize complex command-line arguments.

---

## 13. What is Go (Golang)? {#13-golang}

### Why Go for Microservices?

| Feature | Why It Matters |
|---------|---------------|
| **Fast compilation** | Build all 6 services in seconds |
| **Built-in concurrency** | Handle thousands of requests with goroutines |
| **Single binary** | No dependencies to install — just copy and run |
| **Excellent gRPC support** | First-class Protocol Buffer support |
| **Small Docker images** | ~10MB vs ~300MB for Java/Node |

### Interview Questions

**Q: Why Go over Node.js or Java for this project?**
A: Go compiles to a single binary (simple deployment), has excellent gRPC/Protobuf support, built-in concurrency with goroutines, and produces tiny Docker images (~10MB). It's the standard language for microservices at companies like Uber, Dropbox, and Google.

---

## 14. How It All Fits Together {#14-together}

```
YOU (browser at localhost:3000)
  │
  │ Click "Place Order"
  │
  ▼
NEXT.JS DASHBOARD ──HTTP POST──▶ API GATEWAY (Go, port 8080)
                                    │
                                    │ Converts REST → gRPC
                                    ▼
                              ORDER SERVICE (Go, port 50051)
                                    │
                                    │ 1. Saves order to PostgreSQL
                                    │ 2. Publishes "ORDER_CREATED" to Kafka
                                    ▼
                              KAFKA (port 9092)
                              Topic: order.created
                                    │
                                    │ Saga Orchestrator is listening...
                                    ▼
                              SAGA ORCHESTRATOR (Go, port 50054)
                                    │
                                    │ Step 1: Reserve inventory
                                    ▼
                              INVENTORY SERVICE (Go, port 50053)
                                    │
                                    │ Reserves stock in PostgreSQL
                                    │ Publishes "INVENTORY_RESERVED" to Kafka
                                    ▼
                              SAGA ORCHESTRATOR
                                    │
                                    │ Step 2: Process payment
                                    ▼
                              PAYMENT SERVICE (Go, port 50052)
                                    │
                                    │ Charges card (simulated)
                                    │ Publishes "PAYMENT_COMPLETED" to Kafka
                                    ▼
                              SAGA ORCHESTRATOR
                                    │
                                    │ Step 3: Confirm order
                                    │ Publishes "ORDER_CONFIRMED" to Kafka
                                    ▼
                              NOTIFICATION SERVICE (Go, port 50055)
                                    │
                                    │ Logs: "Customer notified!"
                                    ▼
                              JAEGER (port 16686)
                                    │
                                    │ Shows the full trace of everything above
                                    ▼
                              YOU SEE: Order status = CONFIRMED ✅
```

### Everything Running via Docker

```bash
docker compose up -d     # Starts Postgres, Kafka, Zookeeper, Jaeger
make run-all             # Starts all 6 Go services
cd dashboard && npm run dev  # Starts the Next.js UI
```

One command to understand them all:
```bash
make help                # Shows every available command
```
