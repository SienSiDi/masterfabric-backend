package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
	RevokedAt  *time.Time
}

// IsRevoked reports whether the token has been revoked.
func (t RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }

// IsExpired reports whether the token is past its expiry.
func (t RefreshToken) IsExpired(now time.Time) bool { return now.After(t.ExpiresAt) }

// RemainingLifetime returns the time left until expiry (zero if already expired).
func (t RefreshToken) RemainingLifetime(now time.Time) time.Duration {
	d := t.ExpiresAt.Sub(now)
	if d < 0 {
		return 0
	}
	return d
}
