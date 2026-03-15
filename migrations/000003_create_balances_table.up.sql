CREATE TABLE IF NOT EXISTS balances (
    id                    UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    current_balance       NUMERIC(21,2)   NOT NULL,
    user_id               UUID            NOT NULL UNIQUE,
    created_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ,

    CONSTRAINT fk_user
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT
);