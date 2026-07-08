package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (db *DB) GetReadingRoomConfig(ctx context.Context, did string) (*ReadingRoomConfig, error) {
	row := db.pool.QueryRow(ctx, `
		SELECT did, title, subtitle, description, theme, featured_uris,
		       custom_domain, cf_hostname_id, domain_status, domain_verification_records,
		       show_external_bookmarks, created_at, updated_at
		FROM reading_room_configs WHERE did = $1
	`, did)

	var c ReadingRoomConfig
	err := row.Scan(&c.DID, &c.Title, &c.Subtitle, &c.Description, &c.Theme, &c.FeaturedURIs,
		&c.CustomDomain, &c.CFHostnameID, &c.DomainStatus, &c.DomainVerificationRecords,
		&c.ShowExternalBookmarks, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) UpsertReadingRoomConfig(ctx context.Context, c *ReadingRoomConfig) error {
	if c.Theme == "" {
		c.Theme = "{}"
	}
	if c.FeaturedURIs == "" {
		c.FeaturedURIs = "[]"
	}
	if c.DomainVerificationRecords == "" {
		c.DomainVerificationRecords = "[]"
	}
	_, err := db.pool.Exec(ctx, `
		INSERT INTO reading_room_configs (did, title, subtitle, description, theme, featured_uris,
		       custom_domain, cf_hostname_id, domain_status, domain_verification_records, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (did) DO UPDATE SET
		       title = EXCLUDED.title,
		       subtitle = EXCLUDED.subtitle,
		       description = EXCLUDED.description,
		       theme = EXCLUDED.theme,
		       featured_uris = EXCLUDED.featured_uris,
		       custom_domain = EXCLUDED.custom_domain,
		       cf_hostname_id = EXCLUDED.cf_hostname_id,
		       domain_status = EXCLUDED.domain_status,
		       domain_verification_records = EXCLUDED.domain_verification_records,
		       updated_at = NOW()
	`, c.DID, c.Title, c.Subtitle, c.Description, c.Theme, c.FeaturedURIs,
		c.CustomDomain, c.CFHostnameID, c.DomainStatus, c.DomainVerificationRecords,
		time.Now(), time.Now())
	return err
}

func (db *DB) UpsertReadingRoomConfigFromRecord(ctx context.Context, did, title, subtitle, description, themeJSON, featuredURIsJSON string, showExternalBookmarks bool) error {
	if themeJSON == "" {
		themeJSON = "{}"
	}
	if featuredURIsJSON == "" {
		featuredURIsJSON = "[]"
	}
	_, err := db.pool.Exec(ctx, `
		INSERT INTO reading_room_configs (did, title, subtitle, description, theme, featured_uris, show_external_bookmarks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (did) DO UPDATE SET
		       title = EXCLUDED.title,
		       subtitle = EXCLUDED.subtitle,
		       description = EXCLUDED.description,
		       theme = EXCLUDED.theme,
		       featured_uris = EXCLUDED.featured_uris,
		       show_external_bookmarks = EXCLUDED.show_external_bookmarks,
		       updated_at = NOW()
	`, did, title, subtitle, description, themeJSON, featuredURIsJSON, showExternalBookmarks)
	return err
}

func (db *DB) UpdateReadingRoomDomain(ctx context.Context, did, domain, cfHostnameID, status, verificationRecords string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE reading_room_configs
		SET custom_domain = $2, cf_hostname_id = $3, domain_status = $4,
		    domain_verification_records = $5, updated_at = NOW()
		WHERE did = $1
	`, did, domain, cfHostnameID, status, verificationRecords)
	return err
}

func (db *DB) GetReadingRoomByDomain(ctx context.Context, domain string) (*ReadingRoomConfig, error) {
	row := db.pool.QueryRow(ctx, `
		SELECT did, title, subtitle, description, theme, featured_uris,
		       custom_domain, cf_hostname_id, domain_status, domain_verification_records,
		       show_external_bookmarks, created_at, updated_at
		FROM reading_room_configs WHERE custom_domain = $1 AND domain_status = 'active'
	`, domain)

	var c ReadingRoomConfig
	err := row.Scan(&c.DID, &c.Title, &c.Subtitle, &c.Description, &c.Theme, &c.FeaturedURIs,
		&c.CustomDomain, &c.CFHostnameID, &c.DomainStatus, &c.DomainVerificationRecords,
		&c.ShowExternalBookmarks, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) GetSubscription(ctx context.Context, did string) (*ReadingRoomSubscription, error) {
	row := db.pool.QueryRow(ctx, `
		SELECT did, stripe_customer_id, stripe_subscription_id, status, plan, current_period_end,
		       created_at, updated_at
		FROM reading_room_subscriptions WHERE did = $1
	`, did)

	var s ReadingRoomSubscription
	err := row.Scan(&s.DID, &s.StripeCustomerID, &s.StripeSubscriptionID, &s.Status, &s.Plan,
		&s.CurrentPeriodEnd, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) UpsertSubscription(ctx context.Context, s *ReadingRoomSubscription) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO reading_room_subscriptions (did, stripe_customer_id, stripe_subscription_id, status, plan,
		       current_period_end, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (did) DO UPDATE SET
		       stripe_customer_id = EXCLUDED.stripe_customer_id,
		       stripe_subscription_id = EXCLUDED.stripe_subscription_id,
		       status = EXCLUDED.status,
		       plan = EXCLUDED.plan,
		       current_period_end = EXCLUDED.current_period_end,
		       updated_at = NOW()
	`, s.DID, s.StripeCustomerID, s.StripeSubscriptionID, s.Status, s.Plan,
		s.CurrentPeriodEnd, time.Now(), time.Now())
	return err
}

func (db *DB) GetSubscriptionByCustomerID(ctx context.Context, customerID string) (*ReadingRoomSubscription, error) {
	row := db.pool.QueryRow(ctx, `
		SELECT did, stripe_customer_id, stripe_subscription_id, status, plan, current_period_end,
		       created_at, updated_at
		FROM reading_room_subscriptions WHERE stripe_customer_id = $1
	`, customerID)

	var s ReadingRoomSubscription
	err := row.Scan(&s.DID, &s.StripeCustomerID, &s.StripeSubscriptionID, &s.Status, &s.Plan,
		&s.CurrentPeriodEnd, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) HasActiveSubscription(ctx context.Context, did string) bool {
	sub, err := db.GetSubscription(ctx, did)
	if err != nil || sub == nil {
		return false
	}
	if sub.Status != "active" && sub.Status != "trialing" {
		return false
	}
	if sub.CurrentPeriodEnd != nil && sub.CurrentPeriodEnd.Before(time.Now()) {
		return false
	}
	return true
}
