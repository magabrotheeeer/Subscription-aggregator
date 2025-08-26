ALTER TABLE subscriptions
DROP CONSTRAINT IF EXISTS fk_subscriptions_user_username;

ALTER TABLE subscriptions
DROP CONSTRAINT IF EXISTS fk_subscriptions_user_uid;