package db

import (
	"context"
	"database/sql"
	"embed"
	"time"

	"github.com/pressly/goose/v3"
)

//go:embed migrations
var migrationsFS embed.FS

const (
	LockMigrate        int64 = 0x6D617267696E
	LockFirehose       int64 = 0x6669726568736F65
	LockSessionCleanup int64 = 0x73657373696F6E
	LockBackfill       int64 = 0x6261636B66696C6C
)

func (db *DB) Migrate() error {
	ctx := context.Background()

	acquired, err := db.TryAdvisoryLock(ctx, LockMigrate, func() error {
		goose.SetBaseFS(migrationsFS)
		if err := goose.SetDialect("postgres"); err != nil {
			return err
		}
		return goose.Up(db.DB, "migrations")
	})
	if err != nil {
		return err
	}
	if !acquired {
		_, err = db.AdvisoryLock(ctx, LockMigrate, func() error { return nil })
		return err
	}
	return nil
}

func (db *DB) GetCursor(id string) (int64, error) {
	var cursor int64
	err := db.QueryRow("SELECT last_cursor FROM cursors WHERE id = $1", id).Scan(&cursor)
	switch err {
	case nil:
		return cursor, nil
	case sql.ErrNoRows:
		return 0, nil
	default:
		return 0, err
	}
}

func (db *DB) SetCursor(id string, cursor int64) error {
	_, err := db.Exec(`
		INSERT INTO cursors (id, last_cursor, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT(id) DO UPDATE SET
			last_cursor = EXCLUDED.last_cursor,
			updated_at = EXCLUDED.updated_at
	`, id, cursor, time.Now())
	return err
}
