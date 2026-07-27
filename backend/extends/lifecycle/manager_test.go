package lifecycle

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
)

type blockingDatabasePinger struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
}

func newBlockingDatabasePinger() *blockingDatabasePinger {
	return &blockingDatabasePinger{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (p *blockingDatabasePinger) PingContext(ctx context.Context) error {
	if p.calls.Add(1) == 1 {
		close(p.started)
		select {
		case <-p.release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestManagerReadyAndDrain(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectPing()
	redisServer := miniredis.RunT(t)
	manager := NewManager(db, redis.NewClient(&redis.Options{Addr: redisServer.Addr()}))

	if err := manager.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
	manager.BeginDrain()
	if err := manager.Ready(context.Background()); err == nil {
		t.Fatal("Ready() error = nil while draining")
	}
	if manager.RegisterWebSocket(nil) {
		t.Fatal("RegisterWebSocket(nil) = true while draining")
	}
}

func TestManagerClosesRegisteredWebSocketWithServiceRestart(t *testing.T) {
	manager := NewManager(nil, nil)
	registered := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		if !manager.RegisterWebSocket(conn) {
			_ = conn.CloseNow()
			return
		}
		close(registered)
		defer manager.UnregisterWebSocket(conn)
		for {
			if _, _, err := conn.Read(r.Context()); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, _, err := websocket.Dial(ctx, "ws"+server.URL[len("http"):], nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.CloseNow() }()
	<-registered
	manager.BeginDrain()
	closed := make(chan struct{})
	go func() {
		manager.CloseWebSockets()
		close(closed)
	}()
	_, _, err = client.Read(ctx)
	if got := websocket.CloseStatus(err); got != websocket.StatusServiceRestart {
		t.Fatalf("close status = %d, want %d (err=%v)", got, websocket.StatusServiceRestart, err)
	}
	<-closed
}

func TestManagerDependencyFailureIsNotReady(t *testing.T) {
	db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	manager := NewManager(db, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))
	if err := manager.Ready(context.Background()); err == nil {
		t.Fatal("Ready() error = nil with unavailable dependencies")
	}
}

func TestManagerReadyBoundsBlockedDatabaseProbe(t *testing.T) {
	redisServer := miniredis.RunT(t)
	pinger := newBlockingDatabasePinger()
	manager := newManager(pinger, redis.NewClient(&redis.Options{Addr: redisServer.Addr()}))
	probeCtx, cancelProbe := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancelProbe()

	const callers = 3
	errs := make(chan error, callers)
	for range callers {
		go func() {
			errs <- manager.Ready(probeCtx)
		}()
	}

	<-pinger.started
	for range callers {
		if err := <-errs; !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Ready() error = %v, want context deadline exceeded", err)
		}
	}
	if got := pinger.calls.Load(); got != 1 {
		t.Fatalf("PingContext() calls = %d, want 1 while probe is blocked", got)
	}
	followupCtx, cancelFollowup := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancelFollowup()
	if err := manager.Ready(followupCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("follow-up Ready() error = %v, want context deadline exceeded", err)
	}
	if got := pinger.calls.Load(); got != 1 {
		t.Fatalf("PingContext() calls = %d, want 1 after caller cancellation", got)
	}

	close(pinger.release)
	recoveryCtx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := manager.Ready(recoveryCtx); err != nil {
		t.Fatalf("Ready() after database recovery error = %v", err)
	}
}
