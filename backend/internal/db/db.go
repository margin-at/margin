package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"margin.at/internal/domain"
)

type DB struct {
	*sql.DB
	crypter fieldCrypter
}

type (
	Note             = domain.Note
	Annotation       = domain.Annotation
	Selector         = domain.Selector
	Highlight        = domain.Highlight
	Bookmark         = domain.Bookmark
	Reply            = domain.Reply
	Like             = domain.Like
	Collection       = domain.Collection
	CollectionItem   = domain.CollectionItem
	Notification     = domain.Notification
	APIKey           = domain.APIKey
	Profile          = domain.Profile
	Preferences      = domain.Preferences
	Block            = domain.Block
	Mute             = domain.Mute
	ModerationReport = domain.ModerationReport
	ModerationAction = domain.ModerationAction
	ContentLabel     = domain.ContentLabel
)

func New(dsn string) (*DB, error) {
	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		return nil, fmt.Errorf("only PostgreSQL is supported; DSN must start with postgres:// or postgresql://")
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{DB: sqlDB}, nil
}

func (db *DB) Close() error { return db.DB.Close() }

func (db *DB) AdvisoryLock(ctx context.Context, key int64, fn func() error) (bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return false, fmt.Errorf("advisory lock: acquire conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		return false, fmt.Errorf("advisory lock: acquire: %w", err)
	}
	defer conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", key) //nolint:errcheck

	return true, fn()
}

func (db *DB) TryAdvisoryLock(ctx context.Context, key int64, fn func() error) (bool, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return false, fmt.Errorf("try advisory lock: acquire conn: %w", err)
	}
	defer conn.Close()

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, fmt.Errorf("try advisory lock: %w", err)
	}
	if !acquired {
		return false, nil
	}
	defer conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", key) //nolint:errcheck

	return true, fn()
}

func ParseSelector(selectorJSON *string) (*Selector, error) {
	if selectorJSON == nil || *selectorJSON == "" {
		return nil, nil
	}
	var s Selector
	if err := json.Unmarshal([]byte(*selectorJSON), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func ParseTags(tagsJSON *string) ([]string, error) {
	if tagsJSON == nil || *tagsJSON == "" {
		return nil, nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(*tagsJSON), &tags); err != nil {
		return nil, err
	}
	return tags, nil
}
