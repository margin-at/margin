package firehose

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func readUntilCancel(ctx context.Context, conn *websocket.Conn) (err error) {
	stopWatch := make(chan struct{})
	defer close(stopWatch)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stopWatch:
		}
	}()

	for {
		_, _, rerr := conn.ReadMessage()
		if rerr != nil {
			if ctx.Err() != nil {
				return nil
			}
			return rerr
		}
	}
}

func dialSilent(t *testing.T) (*websocket.Conn, func()) {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		select {}
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}
	return conn, func() { conn.Close(); srv.Close() }
}

func TestSubscribeLoopCancelsWithoutPanic(t *testing.T) {
	conn, cleanup := dialSilent(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- &panicErr{r}
			}
		}()
		done <- readUntilCancel(ctx, conn)
	}()

	time.Sleep(300 * time.Millisecond)
	start := time.Now()
	cancel()

	select {
	case err := <-done:
		if pe, ok := err.(*panicErr); ok {
			t.Fatalf("read loop panicked on cancel: %v", pe.v)
		}
		if err != nil {
			t.Fatalf("read loop returned error on cancel: %v", err)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Fatalf("read loop took %v to return after cancel — too slow", d)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("read loop did not return within 3s of cancel")
	}
}

type panicErr struct{ v any }

func (e *panicErr) Error() string { return "panic" }
