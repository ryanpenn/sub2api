package lifecycle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
)

const dependencyProbeTimeout = 2 * time.Second

type databasePinger interface {
	PingContext(context.Context) error
}

type databaseProbe struct {
	done chan struct{}
	err  error
}

type Manager struct {
	db       databasePinger
	redis    *redis.Client
	draining atomic.Bool

	dbProbeMu sync.Mutex
	dbProbe   *databaseProbe

	mu      sync.Mutex
	sockets map[*websocket.Conn]struct{}
	changed chan struct{}
}

func NewManager(db *sql.DB, redisClient *redis.Client) *Manager {
	if db == nil {
		return newManager(nil, redisClient)
	}
	return newManager(db, redisClient)
}

func newManager(db databasePinger, redisClient *redis.Client) *Manager {
	return &Manager{
		db:      db,
		redis:   redisClient,
		sockets: make(map[*websocket.Conn]struct{}),
		changed: make(chan struct{}, 1),
	}
}

func (m *Manager) Ready(ctx context.Context) error {
	if m.draining.Load() {
		return errors.New("instance is draining")
	}
	ctx, cancel := context.WithTimeout(ctx, dependencyProbeTimeout)
	defer cancel()
	if m.db == nil {
		return errors.New("database client is unavailable")
	}
	if err := m.probeDatabase(ctx); err != nil {
		return fmt.Errorf("database probe failed: %w", err)
	}
	if m.redis == nil {
		return errors.New("redis client is unavailable")
	}
	if err := m.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis probe failed: %w", err)
	}
	return nil
}

func (m *Manager) probeDatabase(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.dbProbeMu.Lock()
	probe := m.dbProbe
	if probe == nil {
		probe = &databaseProbe{done: make(chan struct{})}
		m.dbProbe = probe
		// lib/pq can remain blocked after ctx cancellation while it sends a
		// separate cancel request. Keep that work bounded to one goroutine and
		// let every readiness caller return on its own deadline. The shared
		// probe gets its own deadline so one disconnected caller cannot cancel
		// work that another caller is waiting for.
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dependencyProbeTimeout)
		go m.runDatabaseProbe(probeCtx, cancel, probe)
	}
	m.dbProbeMu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-probe.done:
		return probe.err
	}
}

func (m *Manager) runDatabaseProbe(ctx context.Context, cancel context.CancelFunc, probe *databaseProbe) {
	defer cancel()
	err := m.db.PingContext(ctx)

	m.dbProbeMu.Lock()
	probe.err = err
	close(probe.done)
	if m.dbProbe == probe {
		m.dbProbe = nil
	}
	m.dbProbeMu.Unlock()
}

func (m *Manager) BeginDrain() { m.draining.Store(true) }

func (m *Manager) IsDraining() bool { return m.draining.Load() }

func (m *Manager) RegisterWebSocket(conn *websocket.Conn) bool {
	if conn == nil || m.draining.Load() {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.draining.Load() {
		return false
	}
	m.sockets[conn] = struct{}{}
	m.notifyChanged()
	return true
}

func (m *Manager) UnregisterWebSocket(conn *websocket.Conn) {
	m.mu.Lock()
	delete(m.sockets, conn)
	m.mu.Unlock()
	m.notifyChanged()
}

func (m *Manager) WaitForWebSockets(ctx context.Context) error {
	for {
		m.mu.Lock()
		empty := len(m.sockets) == 0
		m.mu.Unlock()
		if empty {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.changed:
		}
	}
}

func (m *Manager) CloseWebSockets() {
	m.mu.Lock()
	connections := make([]*websocket.Conn, 0, len(m.sockets))
	for conn := range m.sockets {
		connections = append(connections, conn)
	}
	m.mu.Unlock()
	var wg sync.WaitGroup
	for _, conn := range connections {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = conn.Close(websocket.StatusServiceRestart, "service restart")
			_ = conn.CloseNow()
		}()
	}
	wg.Wait()
}

func (m *Manager) notifyChanged() {
	select {
	case m.changed <- struct{}{}:
	default:
	}
}
