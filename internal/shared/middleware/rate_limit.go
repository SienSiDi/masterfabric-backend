package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
)

// RateLimiter is a Redis-backed fixed-window counter using a Lua script for atomicity.
type RateLimiter struct {
	client *redis.Client
	script *redis.Script
}

var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])

local current = redis.call("INCR", key)
if current == 1 then
    redis.call("PEXPIRE", key, window_ms)
end

local remaining = limit - current
if remaining < 0 then
    remaining = 0
end

local pttl = redis.call("PTTL", key)
local reset_at = math.floor((pttl / 1000) + (ARGV[3] / 1000))

return {current, remaining, reset_at}
`)

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client, script: rateLimitScript}
}

// Allow checks whether the key is within the limit for the window.
// Uses a single Lua script for atomic INCR+EXPIRE+TTL (1 RTT instead of 3).
func (rl *RateLimiter) Allow(r *http.Request, key string, limit int, window time.Duration) (bool, int, int64) {
	if rl.client == nil {
		return true, limit, 0
	}
	ctx := r.Context()
	fullKey := "mf:rl:" + key
	now := time.Now().UnixMilli()
	result, err := rl.script.Run(ctx, rl.client, []string{fullKey},
		limit, window.Milliseconds(), now).Int64Slice()
	if err != nil {
		return true, limit, 0
	}
	count := int(result[0])
	remaining := int(result[1])
	resetAt := result[2]
	return count <= limit, remaining, resetAt
}

// RateLimit returns middleware that rate-limits requests using a Redis fixed-window counter.
// keyFn extracts the rate-limit key from the request (e.g. "login:alice@example.com").
// limit is the max requests per window. On over-limit, returns 429 with X-RateLimit-* headers.
func RateLimit(rl *RateLimiter, keyFn func(r *http.Request) string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			allowed, remaining, resetAt := rl.Allow(r, key, limit, window)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))
			if !allowed {
				response.Error(w, domainerr.New(domainerr.CodeTooMany, fmt.Sprintf("rate limit exceeded (max %d per %s)", limit, window), nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
