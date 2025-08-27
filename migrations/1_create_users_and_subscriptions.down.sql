DROP INDEX idx_subscriptions_payment_method_id;
DROP INDEX idx_subscriptions_username;

DROP TABLE subscriptions;
DROP TABLE users;

DROP EXTENSION "pgcrypto";