package leader

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL or DATABASE_URL to run leader election tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS leader_election (
			name TEXT PRIMARY KEY,
			holder TEXT NOT NULL,
			acquired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at TIMESTAMPTZ NOT NULL
		)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, _ = pool.Exec(context.Background(), `DELETE FROM leader_election WHERE name = $1`, lockName)
	return pool
}

func TestTryAcquire_SingleWinner(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	a, err := tryAcquire(ctx, pool, "pod-a")
	if err != nil || !a {
		t.Fatalf("pod-a should acquire: ok=%v err=%v", a, err)
	}

	b, err := tryAcquire(ctx, pool, "pod-b")
	if err != nil {
		t.Fatalf("pod-b acquire err: %v", err)
	}
	if b {
		t.Fatal("pod-b acquired a lease already held by pod-a")
	}

	again, err := tryAcquire(ctx, pool, "pod-a")
	if err != nil || !again {
		t.Fatalf("pod-a should re-acquire its own lease: ok=%v err=%v", again, err)
	}
}

func TestTryAcquire_TakeoverAfterExpiry(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	if ok, err := tryAcquire(ctx, pool, "pod-a"); err != nil || !ok {
		t.Fatalf("pod-a should acquire: ok=%v err=%v", ok, err)
	}

	if _, err := pool.Exec(ctx,
		`UPDATE leader_election SET expires_at = now() - interval '1 second' WHERE name = $1`, lockName); err != nil {
		t.Fatalf("expire: %v", err)
	}

	ok, err := tryAcquire(ctx, pool, "pod-b")
	if err != nil || !ok {
		t.Fatalf("pod-b should take over expired lease: ok=%v err=%v", ok, err)
	}

	if ok, err := tryAcquire(ctx, pool, "pod-a"); err != nil || ok {
		t.Fatalf("pod-a should not reclaim now that pod-b holds it: ok=%v err=%v", ok, err)
	}
}

func TestRenew(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	if ok, err := tryAcquire(ctx, pool, "pod-a"); err != nil || !ok {
		t.Fatalf("acquire: ok=%v err=%v", ok, err)
	}

	held, err := renew(ctx, pool, "pod-a")
	if err != nil || !held {
		t.Fatalf("pod-a should renew its lease: held=%v err=%v", held, err)
	}

	held, err = renew(ctx, pool, "pod-b")
	if err != nil {
		t.Fatalf("renew err: %v", err)
	}
	if held {
		t.Fatal("pod-b renewed a lease it does not hold")
	}
}
