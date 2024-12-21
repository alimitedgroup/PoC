BEGIN;

DROP TABLE IF EXISTS merce_event,
merce;

-- EVENTS
CREATE TABLE
    IF NOT EXISTS create_merce_event (
        id SERIAL PRIMARY KEY,
        message JSONB NOT NULL,
        created_at TIMESTAMP DEFAULT NOW () NOT NULL
    );

-- TABLES
CREATE TABLE
    IF NOT EXISTS merce (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        description TEXT,
        stock INTEGER DEFAULT 0 NOT NULL,
        created_at TIMESTAMP DEFAULT NOW () NOT NULL
    );

COMMIT;