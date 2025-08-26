CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE TABLE users (
    uid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user'
);

CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    service_name TEXT NOT NULL,
    price INT NOT NULL,
    username TEXT NOT NULL,
    user_uid UUID,
    start_date DATE NOT NULL,
    counter_months INT NOT NULL
);