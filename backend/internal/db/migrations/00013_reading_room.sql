-- +goose Up
CREATE TABLE IF NOT EXISTS reading_room_configs (
    did TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    subtitle TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    theme JSONB NOT NULL DEFAULT '{"accentColor":"#3b82f6","fontFamily":"sans-serif","layout":"masonry"}'::jsonb,
    featured_uris JSONB NOT NULL DEFAULT '[]'::jsonb,
    custom_domain TEXT NOT NULL DEFAULT '',
    cf_hostname_id TEXT NOT NULL DEFAULT '',
    domain_status TEXT NOT NULL DEFAULT '',
    domain_verification_records JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reading_room_subscriptions (
    did TEXT PRIMARY KEY,
    stripe_customer_id TEXT NOT NULL DEFAULT '',
    stripe_subscription_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'incomplete',
    plan TEXT NOT NULL DEFAULT 'monthly',
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rr_subscriptions_customer ON reading_room_subscriptions(stripe_customer_id);
CREATE INDEX IF NOT EXISTS idx_rr_subscriptions_status ON reading_room_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_rr_configs_domain ON reading_room_configs(custom_domain) WHERE custom_domain != '';

-- +goose Down
DROP TABLE IF EXISTS reading_room_subscriptions;
DROP TABLE IF EXISTS reading_room_configs;
