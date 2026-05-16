package edgeagent

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"time"
)

type PrefetchTask struct {
	Key         string
	Priority    int
	ScheduledAt time.Time
}

type SmartPrefetchManager struct {
	cache          *Cache
	fetcher        *Fetcher
	tasks          chan PrefetchTask
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	maxConcurrent  int
	bandwidthLimit int
	nightModeStart int
	nightModeEnd   int
	nightBandwidth int
}

func NewSmartPrefetchManager(cache *Cache, fetcher *Fetcher,
	maxConcurrent, bandwidthLimit int) *SmartPrefetchManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &SmartPrefetchManager{
		cache:          cache,
		fetcher:        fetcher,
		tasks:          make(chan PrefetchTask, 1000),
		ctx:            ctx,
		cancel:         cancel,
		maxConcurrent:  maxConcurrent,
		bandwidthLimit: bandwidthLimit,
		nightModeStart: 1,
		nightModeEnd:   7,
		nightBandwidth: 50,
	}
}

func (spm *SmartPrefetchManager) Start() {
	for i := 0; i < spm.maxConcurrent; i++ {
		spm.wg.Add(1)
		go spm.worker()
	}
	slog.Info("smart prefetch started", "workers", spm.maxConcurrent)
}

func (spm *SmartPrefetchManager) Stop() {
	spm.cancel()
	close(spm.tasks)
	spm.wg.Wait()
}

func (spm *SmartPrefetchManager) worker() {
	defer spm.wg.Done()
	for task := range spm.tasks {
		if spm.ctx.Err() != nil {
			return
		}
		if !spm.isNightMode() && task.Priority < 8 {
			continue
		}
		if spm.cache.Has(spm.ctx, task.Key) {
			continue
		}

		result, err := spm.fetcher.Fetch(spm.ctx, task.Key)
		if err != nil {
			slog.Debug("prefetch fetch failed", "key", task.Key, "err", err)
			continue
		}

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, result.Body); err != nil {
			result.Body.Close()
			slog.Debug("prefetch read failed", "key", task.Key, "err", err)
			continue
		}
		result.Body.Close()

		if err := spm.cache.Put(spm.ctx, task.Key, buf, int64(buf.Len())); err != nil {
			slog.Debug("prefetch cache put failed", "key", task.Key, "err", err)
		}
	}
}

func (spm *SmartPrefetchManager) isNightMode() bool {
	h := time.Now().Hour()
	if spm.nightModeStart < spm.nightModeEnd {
		return h >= spm.nightModeStart && h < spm.nightModeEnd
	}
	return h >= spm.nightModeStart || h < spm.nightModeEnd
}

func (spm *SmartPrefetchManager) SchedulePrefetch(key string, priority int) {
	select {
	case spm.tasks <- PrefetchTask{Key: key, Priority: priority, ScheduledAt: time.Now()}:
	default:
		slog.Warn("prefetch queue full", "key", key)
	}
}

func (spm *SmartPrefetchManager) SetNightMode(start, end, bandwidth int) {
	spm.nightModeStart = start
	spm.nightModeEnd = end
	spm.nightBandwidth = bandwidth
}
