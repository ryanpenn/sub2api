package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type scheduledPlanRepoProbe struct {
	ScheduledTestPlanRepository
	listDueCalls atomic.Int64
}

func (r *scheduledPlanRepoProbe) ListDue(context.Context, time.Time) ([]*ScheduledTestPlan, error) {
	r.listDueCalls.Add(1)
	return nil, nil
}

func TestScheduledTestRunnerSkipsCycleWhenPeerOwnsLeaderLock(t *testing.T) {
	cache := &fakeLeaderLockCache{}
	release, acquired := tryAcquireSingletonLeaderLock(context.Background(), cache, nil, scheduledTestLeaderLockKey, "peer", time.Minute)
	if !acquired {
		t.Fatal("peer failed to acquire test leader lock")
	}
	defer release()

	repo := &scheduledPlanRepoProbe{}
	runner := NewScheduledTestRunnerService(repo, nil, nil, nil, &config.Config{})
	runner.SetLeaderLock(cache, nil)
	runner.runScheduledCycle(context.Background())
	if got := repo.listDueCalls.Load(); got != 0 {
		t.Fatalf("ListDue calls = %d, want 0 while peer owns lock", got)
	}
}

func TestScheduledTestRunnerRunsSingleCoordinatedCycle(t *testing.T) {
	repo := &scheduledPlanRepoProbe{}
	runner := NewScheduledTestRunnerService(repo, nil, nil, nil, &config.Config{})
	runner.SetLeaderLock(&fakeLeaderLockCache{}, nil)
	runner.runScheduledCycle(context.Background())
	if got := repo.listDueCalls.Load(); got != 1 {
		t.Fatalf("ListDue calls = %d, want 1", got)
	}
}
