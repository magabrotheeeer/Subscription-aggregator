CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    subscription_id INTEGER NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    user_uid UUID NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
    provider_payment_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    status VARCHAR(50) NOT NULL,
    payment_method_id INTEGER REFERENCES robokassa_payment_tokens(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX idx_payments_user_uid ON payments(user_uid);
CREATE INDEX idx_payments_status ON payments(status);