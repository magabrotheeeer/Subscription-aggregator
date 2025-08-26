ALTER TABLE subscriptions
ADD CONSTRAINT fk_subscriptions_user_username
FOREIGN KEY (username) REFERENCES users(username)
ON DELETE CASCADE;

ALTER TABLE subscriptions
ADD CONSTRAINT fk_subscriptions_user_uid
FOREIGN KEY (user_uid) REFERENCES users(uid)
ON DELETE CASCADE;
