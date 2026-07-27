package oauthsession

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type testSession struct {
	State string `json:"state"`
	Value string `json:"value"`
}

func TestStoreSupportsCrossInstanceOneTimeConsumption(t *testing.T) {
	server := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: server.Addr()})
	clientB := redis.NewClient(&redis.Options{Addr: server.Addr()})
	storeA, err := New[testSession](clientA, "openai", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	storeB, err := New[testSession](clientB, "openai", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := storeA.Put(context.Background(), "sid", &testSession{State: "state", Value: "value"}); err != nil {
		t.Fatal(err)
	}

	var consumed atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session, found, matched, takeErr := storeB.Take(context.Background(), "sid", "state")
			if takeErr != nil {
				t.Errorf("Take() error = %v", takeErr)
				return
			}
			if found && matched {
				if session == nil || session.Value != "value" {
					t.Errorf("Take() session = %#v", session)
				}
				consumed.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := consumed.Load(); got != 1 {
		t.Fatalf("successful consumptions = %d, want 1", got)
	}
}

func TestStoreStateMismatchDoesNotConsume(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store, err := New[testSession](client, "gemini", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "sid", &testSession{State: "expected"}); err != nil {
		t.Fatal(err)
	}
	if _, found, matched, err := store.Take(context.Background(), "sid", "wrong"); err != nil || !found || matched {
		t.Fatalf("mismatch result = found:%v matched:%v err:%v", found, matched, err)
	}
	if _, found, matched, err := store.Take(context.Background(), "sid", "expected"); err != nil || !found || !matched {
		t.Fatalf("second result = found:%v matched:%v err:%v", found, matched, err)
	}
}

func TestStoreTTLNamespaceAndRedisFailure(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	storeA, _ := New[testSession](client, "claude", time.Minute)
	storeB, _ := New[testSession](client, "grok", time.Minute)
	if err := storeA.Put(context.Background(), "sid", &testSession{State: "state"}); err != nil {
		t.Fatal(err)
	}
	if _, found, _, err := storeB.Take(context.Background(), "sid", "state"); err != nil || found {
		t.Fatalf("namespace isolation = found:%v err:%v", found, err)
	}
	server.FastForward(time.Minute + time.Second)
	if _, found, _, err := storeA.Take(context.Background(), "sid", "state"); err != nil || found {
		t.Fatalf("expired result = found:%v err:%v", found, err)
	}

	server.Close()
	if _, _, _, err := storeA.Take(context.Background(), "sid", "state"); err == nil {
		t.Fatal("Take() error = nil after Redis failure")
	}
}
