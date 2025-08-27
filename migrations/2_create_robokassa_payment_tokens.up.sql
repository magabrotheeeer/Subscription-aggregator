CREATE TABLE robokassa_payment_tokens (
    id SERIAL PRIMARY KEY,
    user_uid UUID NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
    op_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_robokassa_payment_tokens_user_uid ON robokassa_payment_tokens(user_uid);