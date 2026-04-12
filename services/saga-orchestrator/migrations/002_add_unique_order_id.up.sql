-- Prevent duplicate sagas for the same order.
-- Even if the application-level idempotency check has a race condition,
-- the database enforces uniqueness as the last line of defense.
CREATE UNIQUE INDEX IF NOT EXISTS idx_sagas_order_id_unique ON sagas(order_id);
