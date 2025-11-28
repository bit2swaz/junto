CREATE TABLE vault_items (
    id BIGSERIAL PRIMARY KEY,
    couple_id BIGINT NOT NULL REFERENCES couples(id),
    created_by BIGINT NOT NULL REFERENCES users(id),
    content_text TEXT NOT NULL,
    unlock_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_vault_items_couple_id ON vault_items(couple_id);
