package usecase

import (
	"context"
	"sync"
	"time"
)

// mockBlacklist is an in-memory RefreshBlacklist for unit tests.
type mockBlacklist struct {
	mu       sync.Mutex
	revoked  map[string]time.Time // hash -> expiry
	revokeOk bool                 // toggle for fault injection
}

func newMockBlacklist() *mockBlacklist {
	return &mockBlacklist{revoked: make(map[string]time.Time), revokeOk: true}
}

func (m *mockBlacklist) IsRevoked(_ context.Context, hash string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.revoked[hash]
	return ok, nil
}

func (m *mockBlacklist) Revoke(_ context.Context, hash string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.revokeOk {
		return errMockConflict
	}
	m.revoked[hash] = time.Now().Add(ttl)
	return nil
}
