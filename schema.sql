BEGIN;

DROP TABLE IF EXISTS order_event,
order_merce,
orders,
merce;

CREATE TABLE IF NOT EXISTS order_event (
    id SERIAL PRIMARY KEY,
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    note TEXT,
    status INTEGER DEFAULT 0 NOT NULL,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE TABLE IF NOT EXISTS merce (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    stock INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE TABLE IF NOT EXISTS order_merce (
    order_id INTEGER REFERENCES orders(id),
    merce_id INTEGER REFERENCES merce(id),
    stock INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    PRIMARY KEY (order_id, merce_id)
);

INSERT INTO
    merce (name, stock, description)
VALUES
    ('T-shirt', 10, 'T-shirt description');

INSERT INTO
    merce (name, stock, description)
VALUES
    ('Pants', 10, 'Pants description');

INSERT INTO
    merce (name, stock, description)
VALUES
    ('Shoes', 10, 'Shoes description');

COMMIT;