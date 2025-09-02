CREATE TABLE yookassa_payment_tokens (
    id SERIAL PRIMARY KEY,
    user_uid UUID NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL,         -- платёжный метод token (card_id)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_yookassa_payment_tokens_user_uid ON yookassa_payment_tokens(user_uid);