CREATE EXTENSION "pgcrypto";

CREATE TABLE users (
    uid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    trial_end_date TIMESTAMPTZ,
    subscription_status TEXT NOT NULL DEFAULT 'trial', -- trial, active, expired
    subscription_expiry TIMESTAMPTZ
);

CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    service_name TEXT NOT NULL,
    price INT NOT NULL,
    username TEXT NOT NULL,
    user_uid UUID NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    counter_months INT NOT NULL,
    next_payment_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
);

CREATE INDEX idx_subscriptions_username ON subscriptions(username);
CREATE INDEX idx_subscriptions_payment_method_id ON subscriptions(payment_method_id);