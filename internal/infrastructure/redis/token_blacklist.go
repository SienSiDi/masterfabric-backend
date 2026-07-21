package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklist stores revoked refresh-token hashes in Redis with a TTL equal to
// the token's remaining lifetime. This lets /auth/refresh reject revoked tokens
// without hitting Postgres on every call.
type TokenBlacklist struct {
	client *redis.Client
}

func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

func blacklistKey(tokenHash string) string { return "mf:revoked:" + tokenHash }

// Revoke adds the token hash to the blacklist with the given TTL.
// If Redis is unavailable, this is a no-op (server runs without blacklist).
func (b *TokenBlacklist) Revoke(ctx context.Context, tokenHash string, ttl time.Duration) error {
	if b.client == nil {
		return nil
	}
	if ttl <= 0 {
		return nil
	}
	if err := b.client.Set(ctx, blacklistKey(tokenHash), "1", ttl).Err(); err != nil {
		return fmt.Errorf("set blacklist: %w", err)
	}
	return nil
}

// IsRevoked reports whether the token hash is on the blacklist.
// If Redis is unavailable, returns false (fail open — don't block users).
func (b *TokenBlacklist) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	if b.client == nil {
		return false, nil
	}
	n, err := b.client.Exists(ctx, blacklistKey(tokenHash)).Result()
	if err != nil {
		return false, nil // fail open
	}
	return n > 0, nil
}
