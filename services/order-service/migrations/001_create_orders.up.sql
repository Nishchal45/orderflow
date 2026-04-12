-- This creates the "orders" shelf in our fridge
-- Each row = one customer order

CREATE TABLE IF NOT EXISTS orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- unique order ID
    customer_id   VARCHAR(255) NOT NULL,                       -- who ordered
    status        VARCHAR(50) NOT NULL DEFAULT 'CREATED',      -- current state
    total_amount  DECIMAL(10,2) NOT NULL DEFAULT 0,            -- total price
    currency      VARCHAR(3) DEFAULT 'USD',                    -- USD, EUR, etc.
    simulate_payment_failure   BOOLEAN DEFAULT FALSE,          -- for testing failures
    simulate_inventory_failure BOOLEAN DEFAULT FALSE,          -- for testing failures
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

-- This stores what items are in each order
-- One order can have many items (1 burger + 2 fries = 2 rows here)

CREATE TABLE IF NOT EXISTS order_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL REFERENCES orders(id),    -- which order this belongs to
    product_id  VARCHAR(255) NOT NULL,                  -- what product
    quantity    INT NOT NULL,                            -- how many
    unit_price  DECIMAL(10,2) NOT NULL,                 -- price per item
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- This logs every status change (like a diary for each order)
-- "Order ABC: CREATED at 10:00, CONFIRMED at 10:02, SHIPPED at 10:05"

CREATE TABLE IF NOT EXISTS order_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID NOT NULL REFERENCES orders(id),
    event_type VARCHAR(100) NOT NULL,
    payload    JSONB,           -- extra data about what happened
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes make lookups faster (like an index in a book)
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_events_order_id ON order_events(order_id);
