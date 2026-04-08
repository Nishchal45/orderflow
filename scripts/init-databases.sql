-- ============================================================
-- OrderFlow: Database initialization script
-- Creates a separate database for each service (database-per-service pattern)
-- This runs automatically when PostgreSQL container starts for the first time
-- ============================================================

-- Order Service database
CREATE DATABASE orderflow_orders;

-- Payment Service database
CREATE DATABASE orderflow_payments;

-- Inventory Service database
CREATE DATABASE orderflow_inventory;

-- Saga Orchestrator database
CREATE DATABASE orderflow_sagas;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE orderflow_orders TO orderflow;
GRANT ALL PRIVILEGES ON DATABASE orderflow_payments TO orderflow;
GRANT ALL PRIVILEGES ON DATABASE orderflow_inventory TO orderflow;
GRANT ALL PRIVILEGES ON DATABASE orderflow_sagas TO orderflow;
