package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	leaderKey     = "cp:leader"
	leaderTTL     = 10 * time.Second
	renewInterval = 3 * time.Second
)

type LeaderElection struct {
	rdb      redis.UniversalClient
	instance string
	isLeader atomic.Bool
	cancel   context.CancelFunc
}

func NewLeaderElection(rdb redis.UniversalClient) *LeaderElection {
	host, _ := os.Hostname()
	return &LeaderElection{
		rdb:      rdb,
		instance: fmt.Sprintf("%s-%s", host, uuid.New().String()[:8]),
	}
}

func (l *LeaderElection) Start(ctx context.Context) {
	ctx, l.cancel = context.WithCancel(ctx)

	l.tryAcquire(ctx)

	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("leader election stopping", "instance", l.instance)
			l.resign()
			return
		case <-ticker.C:
			l.tryAcquire(ctx)
		}
	}
}

func (l *LeaderElection) tryAcquire(ctx context.Context) {
	ok, err := l.rdb.SetNX(ctx, leaderKey, l.instance, leaderTTL).Result()
	if err != nil {
		slog.Warn("leader election: redis error", "err", err)
		return
	}
	if ok {
		if !l.isLeader.Load() {
			slog.Info("leader acquired", "instance", l.instance)
		}
		l.isLeader.Store(true)
		return
	}

	val, err := l.rdb.Get(ctx, leaderKey).Result()
	if err != nil {
		return
	}
	if val == l.instance {
		if l.rdb.Expire(ctx, leaderKey, leaderTTL).Err() == nil {
			l.isLeader.Store(true)
			return
		}
	}

	if l.isLeader.Load() {
		slog.Warn("leader lost", "instance", l.instance)
	}
	l.isLeader.Store(false)
}

func (l *LeaderElection) resign() {
	if !l.isLeader.Load() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := l.rdb.Del(ctx, leaderKey).Err(); err != nil {
		slog.Warn("leader election: failed to resign", "err", err)
	}
	l.isLeader.Store(false)
	slog.Info("leader resigned", "instance", l.instance)
}

func (l *LeaderElection) IsLeader() bool {
	return l.isLeader.Load()
}

func (l *LeaderElection) InstanceID() string {
	return l.instance
}

func (l *LeaderElection) Stop() {
	if l.cancel != nil {
		l.cancel()
	}
}
