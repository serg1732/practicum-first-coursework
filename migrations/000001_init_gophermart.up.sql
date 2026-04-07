-- migrations/000001_init_gophermart.up.sql

-- Создание таблицы метрик
CREATE TABLE IF NOT EXISTS accounts (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    login VARCHAR(255) NOT NULL unique,
    password VARCHAR(255) NOT NULL
);

CREATE TYPE status as ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');


CREATE TABLE IF NOT EXISTS orders (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    order_id TEXT NOT NULL unique,
    accrual DOUBLE PRECISION,
    status status NOT NULL default 'NEW',
    uploaded_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS withdraws (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    order_id TEXT NOT NULL,
    sum DOUBLE PRECISION default 0.0,
    processed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

Create TABLE IF NOT EXISTS balance (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    balance DOUBLE PRECISION default 0.0,
    withdraw DOUBLE PRECISION default 0.0
)