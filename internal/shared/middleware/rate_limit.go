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

// RateLimiter is a Redis-backed fixed-window counter.
type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow checks whether the key is within the limit for the window. If allowed,
// it increments the counter and returns true. If over limit, returns false.
// It also sets the X-RateLimit-* headers on the response.
func (rl *RateLimiter) Allow(r *http.Request, key string, limit int, window time.Duration) (bool, int, int64) {
	ctx := r.Context()
	fullKey := "mf:rl:" + key
	count, err := rl.client.Incr(ctx, fullKey).Result()
	if err != nil {
		// fail open — don't block on Redis errors
		return true, limit, 0
	}
	if count == 1 {
		_ = rl.client.Expire(ctx, fullKey, window).Err()
	}
	ttl, _ := rl.client.TTL(ctx, fullKey).Result()
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	resetAt := time.Now().Add(ttl).Unix()
	if int(count) > limit {
		return false, 0, resetAt
	}
	return true, remaining, resetAt
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
