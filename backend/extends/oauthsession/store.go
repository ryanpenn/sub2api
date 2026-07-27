package oauthsession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var takeScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
  return {0, ""}
end
if ARGV[1] ~= "" then
  local decoded = cjson.decode(value)
  if type(decoded["state"]) ~= "string" or decoded["state"] ~= ARGV[1] then
    return {2, ""}
  end
end
redis.call("DEL", KEYS[1])
return {1, value}
`)

// Store persists short-lived OAuth sessions in Redis. T is the provider's
// existing OAuth session type; no shared domain model is introduced here.
type Store[T any] struct {
	client    *redis.Client
	namespace string
	ttl       time.Duration
}

func New[T any](client *redis.Client, namespace string, ttl time.Duration) (*Store[T], error) {
	if client == nil {
		return nil, errors.New("oauth session redis client is required")
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" || strings.ContainsAny(namespace, ":{} \t\r\n") {
		return nil, fmt.Errorf("invalid oauth session namespace %q", namespace)
	}
	if ttl <= 0 {
		return nil, errors.New("oauth session ttl must be positive")
	}
	return &Store[T]{client: client, namespace: namespace, ttl: ttl}, nil
}

func (s *Store[T]) Put(ctx context.Context, sessionID string, session *T) error {
	if strings.TrimSpace(sessionID) == "" || session == nil {
		return errors.New("oauth session id and value are required")
	}
	value, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("encode oauth session: %w", err)
	}
	if err := s.client.Set(ctx, s.key(sessionID), value, s.ttl).Err(); err != nil {
		return fmt.Errorf("store oauth session: %w", err)
	}
	return nil
}

// Take atomically reads and consumes a session. When expectedState is not
// empty, a mismatch is reported without deleting the session.
func (s *Store[T]) Take(ctx context.Context, sessionID, expectedState string) (*T, bool, bool, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, false, false, errors.New("oauth session id is required")
	}
	result, err := takeScript.Run(ctx, s.client, []string{s.key(sessionID)}, expectedState).Slice()
	if err != nil {
		return nil, false, false, fmt.Errorf("take oauth session: %w", err)
	}
	if len(result) != 2 {
		return nil, false, false, fmt.Errorf("take oauth session: unexpected result length %d", len(result))
	}
	status, ok := result[0].(int64)
	if !ok {
		return nil, false, false, fmt.Errorf("take oauth session: unexpected status %T", result[0])
	}
	switch status {
	case 0:
		return nil, false, false, nil
	case 2:
		return nil, true, false, nil
	case 1:
		encoded, ok := result[1].(string)
		if !ok {
			return nil, false, false, fmt.Errorf("take oauth session: unexpected value %T", result[1])
		}
		var session T
		if err := json.Unmarshal([]byte(encoded), &session); err != nil {
			return nil, false, false, fmt.Errorf("decode oauth session: %w", err)
		}
		return &session, true, true, nil
	default:
		return nil, false, false, fmt.Errorf("take oauth session: unexpected status %d", status)
	}
}

func (s *Store[T]) Stop() {}

func (s *Store[T]) key(sessionID string) string {
	return "sub2api:oauth-session:" + s.namespace + ":" + sessionID
}
