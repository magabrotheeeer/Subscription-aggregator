DROP INDEX IF EXISTS idx_subscriptions_payment_method_id;

ALTER TABLE subscriptions
DROP COLUMN IF EXISTS payment_method_id;