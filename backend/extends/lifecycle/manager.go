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

type Manager struct {
	db       *sql.DB
	redis    *redis.Client
	draining atomic.Bool

	mu      sync.Mutex
	sockets map[*websocket.Conn]struct{}
	changed chan struct{}
}

func NewManager(db *sql.DB, redisClient *redis.Client) *Manager {
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
	if err := m.db.PingContext(ctx); err != nil {
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
