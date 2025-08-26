ALTER TABLE subscriptions
ADD COLUMN payment_method_id INTEGER REFERENCES payment_methods(id) ON DELETE SET NULL;
CREATE INDEX idx_subscriptions_payment_method_id ON subscriptions(payment_method_id);