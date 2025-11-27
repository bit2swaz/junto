-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    avatar_config JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create couples table
CREATE TABLE IF NOT EXISTS couples (
    id BIGSERIAL PRIMARY KEY,
    user1_id BIGINT NOT NULL REFERENCES users(id),
    user2_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add couple_id to users table
ALTER TABLE users ADD COLUMN couple_id BIGINT REFERENCES couples(id);
