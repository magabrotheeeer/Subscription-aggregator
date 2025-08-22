ALTER TABLE subscriptions 
ADD CONSTRAINT fk_subscriptions_user
FOREIGN KEY (username) REFERENCES users(username)
ON DELETE CASCADE;