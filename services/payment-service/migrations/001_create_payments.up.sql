CREATE TABLE IF NOT EXISTS payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL,
    amount          DECIMAL(10,2) NOT NULL,
    currency        VARCHAR(3) DEFAULT 'USD',
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    failure_reason  TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_order_id_unique ON payments(order_id) WHERE status != 'REFUNDED';
