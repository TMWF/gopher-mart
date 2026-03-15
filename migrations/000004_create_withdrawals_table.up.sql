CREATE TABLE IF NOT EXISTS withdrawals (
    id                    UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    sum                   NUMERIC(21,2)   NOT NULL,
    order_id              UUID            NOT NULL,
    created_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ,

    CONSTRAINT fk_order
    FOREIGN KEY (order_id)
    REFERENCES orders (id)
    ON DELETE RESTRICT
);