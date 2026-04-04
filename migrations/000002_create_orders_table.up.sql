CREATE TABLE IF NOT EXISTS orders (
    id UUID     PRIMARY KEY     DEFAULT gen_random_uuid(),
    order_id    TEXT            NOT NULL UNIQUE,
    status      TEXT            NOT NULL,
    accrual     NUMERIC(21, 4)  NOT NULL DEFAULT 0,
    user_id     UUID            NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,

    CONSTRAINT chk_status_valid 
    CHECK (status IN ('NEW','PROCESSING','INVALID','PROCESSED')),

    CONSTRAINT fk_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_orders_user_id_order_id ON orders (order_id, user_id);

-- TODO: DELETE
-- CREATE INDEX idx_users_active
--   ON users(email)
--   WHERE deleted_at IS NULL;