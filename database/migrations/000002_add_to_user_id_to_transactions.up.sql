ALTER TABLE transactions ADD COLUMN to_user_id UUID REFERENCES users(id);
CREATE INDEX idx_transactions_to_user_id ON transactions(to_user_id); 