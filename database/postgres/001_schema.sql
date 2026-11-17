-- Dhaba Orders schema. Idempotent: safe to re-run (e.g. from a Kubernetes migration Job).

CREATE TABLE IF NOT EXISTS menu_categories (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(80)  NOT NULL UNIQUE,
    slug        VARCHAR(80)  NOT NULL UNIQUE,
    sort_order  INTEGER      NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS menu_items (
    id             SERIAL PRIMARY KEY,
    category_id    INTEGER      NOT NULL REFERENCES menu_categories(id) ON DELETE RESTRICT,
    name           VARCHAR(120) NOT NULL,
    description    TEXT         NOT NULL DEFAULT '',
    price_cents    INTEGER      NOT NULL CHECK (price_cents >= 0),
    is_available   BOOLEAN      NOT NULL DEFAULT TRUE,
    is_vegetarian  BOOLEAN      NOT NULL DEFAULT FALSE,
    spice_level    SMALLINT     NOT NULL DEFAULT 0 CHECK (spice_level BETWEEN 0 AND 3),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_menu_items_category_name UNIQUE (category_id, name)
);
CREATE INDEX IF NOT EXISTS idx_menu_items_category ON menu_items(category_id);
CREATE INDEX IF NOT EXISTS idx_menu_items_available ON menu_items(is_available);

CREATE TABLE IF NOT EXISTS orders (
    id                  BIGSERIAL    PRIMARY KEY,
    customer_name       VARCHAR(80)  NOT NULL,
    table_number        INTEGER      CHECK (table_number IS NULL OR table_number BETWEEN 1 AND 500),
    notes               VARCHAR(500) NOT NULL DEFAULT '',
    status              VARCHAR(16)  NOT NULL DEFAULT 'NEW'
                        CHECK (status IN ('NEW', 'COOKING', 'READY')),
    total_cents         INTEGER      NOT NULL CHECK (total_cents >= 0),
    job_enqueued        BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    cooking_started_at  TIMESTAMPTZ,
    ready_at            TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_not_enqueued ON orders(created_at) WHERE job_enqueued = FALSE;

CREATE TABLE IF NOT EXISTS order_items (
    id                BIGSERIAL    PRIMARY KEY,
    order_id          BIGINT       NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id      INTEGER      NOT NULL REFERENCES menu_items(id) ON DELETE RESTRICT,
    item_name         VARCHAR(120) NOT NULL,
    unit_price_cents  INTEGER      NOT NULL CHECK (unit_price_cents >= 0),
    quantity          INTEGER      NOT NULL CHECK (quantity BETWEEN 1 AND 50),
    line_total_cents  INTEGER      NOT NULL CHECK (line_total_cents >= 0),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_menu_item ON order_items(menu_item_id);

-- Keep updated_at current on every UPDATE.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_menu_categories_updated ON menu_categories;
CREATE TRIGGER trg_menu_categories_updated BEFORE UPDATE ON menu_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_menu_items_updated ON menu_items;
CREATE TRIGGER trg_menu_items_updated BEFORE UPDATE ON menu_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_orders_updated ON orders;
CREATE TRIGGER trg_orders_updated BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Notify listeners (menu-api) whenever menu data changes so the Redis cache is invalidated.
CREATE OR REPLACE FUNCTION notify_menu_changed() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('menu_changed', TG_TABLE_NAME);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_menu_categories_notify ON menu_categories;
CREATE TRIGGER trg_menu_categories_notify AFTER INSERT OR UPDATE OR DELETE ON menu_categories
    FOR EACH STATEMENT EXECUTE FUNCTION notify_menu_changed();

DROP TRIGGER IF EXISTS trg_menu_items_notify ON menu_items;
CREATE TRIGGER trg_menu_items_notify AFTER INSERT OR UPDATE OR DELETE ON menu_items
    FOR EACH STATEMENT EXECUTE FUNCTION notify_menu_changed();
