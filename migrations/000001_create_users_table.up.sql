CREATE TABLE IF NOT EXISTS users (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    login       TEXT            NOT NULL UNIQUE,
    password    TEXT            NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- TODO: DELEte
-- CREATE INDEX idx_users_active
--   ON users(email)
--   WHERE deleted_at IS NULL;

-- CREATE INDEX idx_orders_pending
--   ON orders(created_at)
--   WHERE status = 'pending';