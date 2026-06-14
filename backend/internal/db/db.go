package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"margin.at/internal/db/sqlcdb"
	"margin.at/internal/domain"
)

type DB struct {
	pool        *pgxpool.Pool
	migrationDB *sql.DB
	q           *sqlcdb.Queries
	crypter     fieldCrypter
}

func (db *DB) Pool() *pgxpool.Pool { return db.pool }

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

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.MaxConns = 25
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{pool: pool, migrationDB: stdlib.OpenDBFromPool(pool), q: sqlcdb.New(pool)}, nil
}

func (db *DB) Close() error {
	if db.migrationDB != nil {
		db.migrationDB.Close()
	}
	if db.pool != nil {
		db.pool.Close()
	}
	return nil
}

func (db *DB) AdvisoryLock(ctx context.Context, key int64, fn func() error) (bool, error) {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("advisory lock: acquire conn: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		return false, fmt.Errorf("advisory lock: acquire: %w", err)
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", key) //nolint:errcheck

	return true, fn()
}

func (db *DB) TryAdvisoryLock(ctx context.Context, key int64, fn func() error) (bool, error) {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("try advisory lock: acquire conn: %w", err)
	}
	defer conn.Release()

	var acquired bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, fmt.Errorf("try advisory lock: %w", err)
	}
	if !acquired {
		return false, nil
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", key) //nolint:errcheck

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
