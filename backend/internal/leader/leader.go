package leader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	lockName        = "background_workers"
	leaseTTL        = 30 * time.Second
	heartbeat       = 10 * time.Second
	acquireInterval = 5 * time.Second
)

func instanceID() string {
	host, _ := os.Hostname()
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return host + "-" + hex.EncodeToString(b)
}

func Acquire(ctx context.Context, pool *pgxpool.Pool) (leaderCtx context.Context, release func(), err error) {
	id := instanceID()

	waiting := false
	for {
		ok, aerr := tryAcquire(ctx, pool, id)
		if aerr != nil {
			log.Printf("leader: acquire query failed: %v", aerr)
		} else if ok {
			break
		} else if !waiting {
			log.Printf("leader: another instance is leader — standing by")
			waiting = true
		}

		select {
		case <-ctx.Done():
			return nil, func() {}, ctx.Err()
		case <-time.After(acquireInterval):
		}
	}

	leaderCtx, cancel := context.WithCancel(ctx)
	go renewLoop(leaderCtx, cancel, pool, id)

	release = func() {
		cancel()
		rctx, rcancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rcancel()
		_, _ = pool.Exec(rctx,
			`DELETE FROM leader_election WHERE name = $1 AND holder = $2`, lockName, id)
	}
	return leaderCtx, release, nil
}

func tryAcquire(ctx context.Context, pool *pgxpool.Pool, id string) (bool, error) {
	var holder string
	err := pool.QueryRow(ctx, `
		INSERT INTO leader_election (name, holder, acquired_at, expires_at)
		VALUES ($1, $2, now(), now() + $3::interval)
		ON CONFLICT (name) DO UPDATE
		SET holder = EXCLUDED.holder,
		    acquired_at = now(),
		    expires_at = EXCLUDED.expires_at
		WHERE leader_election.expires_at < now() OR leader_election.holder = EXCLUDED.holder
		RETURNING holder`,
		lockName, id, intervalString(leaseTTL),
	).Scan(&holder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return holder == id, nil
}

func renew(ctx context.Context, pool *pgxpool.Pool, id string) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE leader_election
		SET expires_at = now() + $3::interval
		WHERE name = $1 AND holder = $2`,
		lockName, id, intervalString(leaseTTL),
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func renewLoop(ctx context.Context, cancel context.CancelFunc, pool *pgxpool.Pool, id string) {
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()
	misses := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			held, err := renew(ctx, pool, id)
			if err != nil {
				misses++
				log.Printf("leader: renew failed (%d/3): %v", misses, err)
				if misses >= 3 {
					log.Printf("leader: lost lease after repeated renew failures — stopping workers")
					cancel()
					return
				}
				continue
			}
			misses = 0
			if !held {
				log.Printf("leader: lease taken over by another instance — stopping workers")
				cancel()
				return
			}
		}
	}
}

func intervalString(d time.Duration) string {
	return fmt.Sprintf("%d seconds", int(d.Seconds()))
}
