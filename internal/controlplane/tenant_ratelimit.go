package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/darkinno/edge-dispatch-framework/internal/models"
	"github.com/redis/go-redis/v9"
)

type TenantRateLimiter struct {
	rdb    *redis.Client
	mu     sync.RWMutex
	limits map[string]*tenantLimit
}

type tenantLimit struct {
	ReqPerSec int
	Burst     int
}

const (
	defaultReqPerSec = 100
	defaultBurst     = 200
)

var (
	rateLimitScript = redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])

		-- Get current state
		local tokens = tonumber(redis.call("HGET", key, "tokens"))
		local last_refill = tonumber(redis.call("HGET", key, "last_refill"))

		if tokens == nil then
			tokens = burst
		end
		if last_refill == nil then
			last_refill = now
		end

		-- Refill tokens based on elapsed time
		local elapsed = now - last_refill
		if elapsed > 0 then
			local refill = elapsed * rate / 1000.0
			tokens = math.min(burst, tokens + refill)
			last_refill = now
		end

		if tokens >= 1 then
			tokens = tokens - 1
			redis.call("HSET", key, "tokens", tokens, "last_refill", last_refill)
			redis.call("EXPIRE", key, 10)
			return 1
		end

		redis.call("HSET", key, "tokens", tokens, "last_refill", last_refill)
		redis.call("EXPIRE", key, 10)
		return 0
	`)
)

func NewTenantRateLimiter(rdb *redis.Client) *TenantRateLimiter {
	return &TenantRateLimiter{
		rdb:    rdb,
		limits: make(map[string]*tenantLimit),
	}
}

func (trl *TenantRateLimiter) SetLimit(tenantID string, reqPerSec, burst int) {
	trl.mu.Lock()
	defer trl.mu.Unlock()
	trl.limits[tenantID] = &tenantLimit{ReqPerSec: reqPerSec, Burst: burst}
}

func (trl *TenantRateLimiter) getLimit(tenantID string) (reqPerSec, burst int) {
	trl.mu.RLock()
	defer trl.mu.RUnlock()
	if l, ok := trl.limits[tenantID]; ok {
		return l.ReqPerSec, l.Burst
	}
	return defaultReqPerSec, defaultBurst
}

func (trl *TenantRateLimiter) Allow(ctx context.Context, tenantID string) bool {
	reqPerSec, burst := trl.getLimit(tenantID)
	key := fmt.Sprintf("rl:tenant:%s", tenantID)
	nowMs := time.Now().UnixMilli()

	val, err := rateLimitScript.Run(ctx, trl.rdb, []string{key}, reqPerSec, burst, nowMs).Result()
	if err != nil {
		slog.Warn("rate limiter redis error", "tenant", tenantID, "err", err)
		return true
	}

	allowed, ok := val.(int64)
	if !ok {
		return true
	}
	return allowed == 1
}

func (trl *TenantRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Actor-TenantId")
		if tenantID == "" {
			tenantID = "default"
		}
		if !trl.Allow(r.Context(), tenantID) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    "RATE_LIMITED",
					Message: "tenant rate limit exceeded",
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
