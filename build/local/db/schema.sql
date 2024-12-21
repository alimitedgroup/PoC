BEGIN;

DROP TABLE IF EXISTS order_event,
order_merce,
orders,
merce;

-- EVENTS
CREATE TABLE
    IF NOT EXISTS order_event (
        id SERIAL PRIMARY KEY,
        message JSONB NOT NULL,
        created_at TIMESTAMP DEFAULT NOW () NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS merce_stock_update_event (
        id SERIAL PRIMARY KEY,
        message JSONB NOT NULL,
        created_at TIMESTAMP DEFAULT NOW () NOT NULL
    );

-- TABLES
CREATE TABLE
    IF NOT EXISTS orders (
        id SERIAL PRIMARY KEY,
        note TEXT,
        status INTEGER DEFAULT 0 NOT NULL,
        created_at TIMESTAMP DEFAULT NOW () NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS merce (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        description TEXT,
        stock INTEGER DEFAULT 0 NOT NULL,
        created_at TIMESTAMP DEFAULT NOW () NOT NULL
    );

CREATE TABLE
    IF NOT EXISTS order_merce (
        order_id INTEGER REFERENCES orders (id),
        merce_id INTEGER REFERENCES merce (id),
        stock INTEGER NOT NULL,
        created_at TIMESTAMP DEFAULT NOW () NOT NULL,
        PRIMARY KEY (order_id, merce_id)
    );

COMMIT;