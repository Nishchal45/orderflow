-- Table 1: sagas — one row per order being processed
-- Think of it as the head chef's order board

CREATE TABLE IF NOT EXISTS sagas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL,                              -- which order this saga is for
    status          VARCHAR(50) NOT NULL DEFAULT 'STARTED',     -- STARTED, COMPLETED, FAILED, COMPENSATING, COMPENSATED
    current_step    VARCHAR(100) NOT NULL DEFAULT 'RESERVE_INVENTORY',  -- what step we're on right now
    failure_reason  TEXT,                                        -- why it failed (if it did)
    started_at      TIMESTAMPTZ DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

-- Table 2: saga_steps — tracks each individual step
-- Like a checklist: ✅ reserve inventory, ✅ process payment, ❌ confirm order

CREATE TABLE IF NOT EXISTS saga_steps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saga_id         UUID NOT NULL REFERENCES sagas(id),         -- which saga this step belongs to
    step_name       VARCHAR(100) NOT NULL,                       -- RESERVE_INVENTORY, PROCESS_PAYMENT, etc.
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING',      -- PENDING, EXECUTING, COMPLETED, FAILED, COMPENSATED
    error           TEXT,                                         -- error message if step failed
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sagas_order_id ON sagas(order_id);
CREATE INDEX IF NOT EXISTS idx_saga_steps_saga_id ON saga_steps(saga_id);
