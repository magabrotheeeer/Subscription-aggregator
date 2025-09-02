CREATE TABLE yookassa_payments (
    id SERIAL PRIMARY KEY,
    user_uid UUID NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
    subscription_id INTEGER REFERENCES subscriptions(id) ON DELETE SET NULL,
    payment_id VARCHAR(255) NOT NULL,    -- ID платежа из ЮKassa
    amount BIGINT NOT NULL,              -- сумма в копейках или минимальных единицах
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    status VARCHAR(50) NOT NULL,
    payment_token_id INTEGER REFERENCES yookassa_payment_tokens(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_yookassa_payments_user_uid ON yookassa_payments(user_uid);
CREATE INDEX idx_yookassa_payments_subscription_id ON yookassa_payments(subscription_id);