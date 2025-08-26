CREATE TABLE payment_methods (
    id SERIAL PRIMARY KEY,
    user_uid UUID NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
    provider_customer_id VARCHAR(255) NOT NULL,
    payment_method_token VARCHAR(255) NOT NULL,
    card_brand VARCHAR(50),
    card_last_four VARCHAR(4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX idx_payment_methods_user_uid ON payment_methods(user_uid);