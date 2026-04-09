CREATE TABLE IF NOT EXISTS products (
    id          VARCHAR(255) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    stock       INT NOT NULL DEFAULT 0,
    reserved    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reservations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL,
    product_id  VARCHAR(255) NOT NULL REFERENCES products(id),
    quantity    INT NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'RESERVED',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reservations_order_id ON reservations(order_id);

-- Seed sample products
INSERT INTO products (id, name, stock) VALUES
    ('burger', 'Classic Burger', 100),
    ('fries', 'French Fries', 200),
    ('pizza', 'Margherita Pizza', 50),
    ('soda', 'Cola', 300),
    ('salad', 'Caesar Salad', 75)
ON CONFLICT (id) DO NOTHING;
