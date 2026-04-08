# Architecture Diagrams

## 1. System Architecture (High Level)

```mermaid
graph TB
    Client[Next.js Dashboard :3000]

    Client -->|REST HTTP| GW[API Gateway :8080]

    GW -->|gRPC| OS[Order Service :50051]
    GW -->|gRPC| PS[Payment Service :50052]
    GW -->|gRPC| IS[Inventory Service :50053]

    OS -->|Publish Events| K[Apache Kafka]
    PS -->|Publish Events| K
    IS -->|Publish Events| K

    K -->|Consume Events| SO[Saga Orchestrator :50054]
    K -->|Consume Events| NS[Notification Service :50055]

    SO -->|gRPC Commands| OS
    SO -->|gRPC Commands| PS
    SO -->|gRPC Commands| IS
    SO -->|Publish Events| K

    OS --- DB1[(PostgreSQL orders)]
    PS --- DB2[(PostgreSQL payments)]
    IS --- DB3[(PostgreSQL inventory)]
    SO --- DB4[(PostgreSQL sagas)]

    GW -.->|Traces| J[Jaeger :16686]
    OS -.->|Traces| J
    PS -.->|Traces| J
    IS -.->|Traces| J
    SO -.->|Traces| J

    style Client fill:#7c3aed,color:#fff
    style GW fill:#2563eb,color:#fff
    style K fill:#e11d48,color:#fff
    style SO fill:#ea580c,color:#fff
    style J fill:#16a34a,color:#fff
```

## 2. Saga Orchestration Flow (Happy Path)

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as API Gateway
    participant OS as Order Service
    participant K as Kafka
    participant SO as Saga Orchestrator
    participant IS as Inventory Service
    participant PS as Payment Service
    participant NS as Notification Service

    C->>GW: POST /api/v1/orders
    GW->>OS: gRPC CreateOrder()
    OS->>OS: Save order (CREATED)
    OS-->>GW: Order ID
    GW-->>C: 201 Created
    OS->>K: Publish ORDER_CREATED

    K->>SO: Consume ORDER_CREATED
    SO->>SO: Create saga (STARTED)

    rect rgb(59, 130, 246, 0.1)
        Note over SO,IS: Step 1: Reserve Inventory
        SO->>IS: gRPC ReserveInventory()
        IS->>IS: Reserve stock
        IS->>K: Publish INVENTORY_RESERVED
        K->>SO: Consume INVENTORY_RESERVED
        SO->>SO: Update step (COMPLETED)
    end

    rect rgb(16, 185, 129, 0.1)
        Note over SO,PS: Step 2: Process Payment
        SO->>PS: gRPC ProcessPayment()
        PS->>PS: Charge customer
        PS->>K: Publish PAYMENT_COMPLETED
        K->>SO: Consume PAYMENT_COMPLETED
        SO->>SO: Update step (COMPLETED)
    end

    rect rgb(139, 92, 246, 0.1)
        Note over SO,OS: Step 3: Confirm Order
        SO->>OS: gRPC UpdateOrderStatus(CONFIRMED)
        SO->>K: Publish ORDER_CONFIRMED
    end

    rect rgb(245, 158, 11, 0.1)
        Note over SO,NS: Step 4: Notify
        K->>NS: Consume ORDER_CONFIRMED
        NS->>NS: Log notification
        NS->>K: Publish NOTIFICATION_SENT
    end
```

## 3. Saga Failure & Compensation Flow

```mermaid
sequenceDiagram
    participant SO as Saga Orchestrator
    participant IS as Inventory Service
    participant PS as Payment Service
    participant OS as Order Service
    participant K as Kafka

    Note over SO: Step 1: Reserve Inventory (SUCCESS)
    SO->>IS: gRPC ReserveInventory()
    IS-->>SO: OK (reserved)

    Note over SO: Step 2: Process Payment (FAILS)
    SO->>PS: gRPC ProcessPayment()
    PS-->>SO: ERROR (insufficient funds)
    PS->>K: Publish PAYMENT_FAILED

    rect rgb(239, 68, 68, 0.1)
        Note over SO: COMPENSATION BEGINS (reverse order)

        Note over SO,IS: Compensate Step 1: Release Inventory
        SO->>IS: gRPC ReleaseInventory()
        IS->>IS: Release reserved stock
        IS->>K: Publish INVENTORY_RELEASED

        Note over SO,OS: Compensate Step 0: Cancel Order
        SO->>OS: gRPC UpdateOrderStatus(CANCELLED)
        SO->>K: Publish ORDER_CANCELLED
    end
```

## 4. Data Flow Diagram

```mermaid
graph LR
    subgraph "Client Layer"
        D[Dashboard]
    end

    subgraph "Gateway Layer"
        GW[API Gateway]
    end

    subgraph "Service Layer"
        OS[Order Service]
        PS[Payment Service]
        IS[Inventory Service]
        SO[Saga Orchestrator]
        NS[Notification Service]
    end

    subgraph "Data Layer"
        DB1[(Orders DB)]
        DB2[(Payments DB)]
        DB3[(Inventory DB)]
        DB4[(Sagas DB)]
    end

    subgraph "Messaging Layer"
        K[Kafka]
    end

    subgraph "Observability Layer"
        J[Jaeger]
    end

    D -->|HTTP REST| GW
    GW -->|gRPC| OS & PS & IS

    OS --> DB1
    PS --> DB2
    IS --> DB3
    SO --> DB4

    OS & PS & IS -->|Events| K
    K -->|Events| SO & NS
    SO -->|Commands via gRPC| OS & PS & IS

    OS & PS & IS & SO & GW -.->|OTLP| J
```

## 5. Order State Machine

```mermaid
stateDiagram-v2
    [*] --> CREATED: Customer places order

    CREATED --> INVENTORY_RESERVING: Saga starts

    INVENTORY_RESERVING --> PAYMENT_PENDING: Stock reserved
    INVENTORY_RESERVING --> REJECTED: Insufficient stock

    PAYMENT_PENDING --> CONFIRMED: Payment succeeds
    PAYMENT_PENDING --> ROLLING_BACK: Payment fails

    ROLLING_BACK --> CANCELLED: Compensation complete

    CONFIRMED --> SHIPPED: Notification sent

    SHIPPED --> [*]
    CANCELLED --> [*]
    REJECTED --> [*]
```

## 6. Docker Compose Service Map

```mermaid
graph TB
    subgraph "Docker Compose Network: orderflow"
        subgraph "Infrastructure"
            PG[(PostgreSQL :5432)]
            ZK[Zookeeper :2181]
            KF[Kafka :9092]
            JG[Jaeger :16686]
        end

        subgraph "Application Services"
            GW[API Gateway :8080]
            OS[Order Service :50051]
            PS[Payment Service :50052]
            IS[Inventory Service :50053]
            SO[Saga Orchestrator :50054]
            NS[Notification Service :50055]
        end

        subgraph "Frontend"
            FE[Dashboard :3000]
        end
    end

    FE --> GW
    GW --> OS & PS & IS
    OS & PS & IS & SO --> PG
    OS & PS & IS --> KF
    KF --> SO & NS
    ZK --> KF
    GW & OS & PS & IS & SO -.-> JG
```
