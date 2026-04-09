-- This UNDOES the "up" migration (deletes everything)
-- Used when you need to start fresh

DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
