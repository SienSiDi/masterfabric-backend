package llm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"sync"
	"time"

	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
)

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type CachedMonitoringRepository struct {
	inner llmrepo.MonitoringRepository
	mu    sync.RWMutex
	cache map[string]*cacheEntry
	ttl   time.Duration
}

func NewCachedMonitoringRepository(inner llmrepo.MonitoringRepository, ttl time.Duration) *CachedMonitoringRepository {
	return &CachedMonitoringRepository{inner: inner, cache: make(map[string]*cacheEntry), ttl: ttl}
}

func cacheKey(filter llmrepo.MonitoringFilter) string {
	b, _ := json.Marshal(filter)
	h := sha256.Sum256(b)
	return string(h[:])
}

func (r *CachedMonitoringRepository) GetReport(ctx context.Context, filter llmrepo.MonitoringFilter) (*llmrepo.MonitoringReport, error) {
	key := cacheKey(filter)
	now := time.Now()

	r.mu.RLock()
	entry, ok := r.cache[key]
	r.mu.RUnlock()

	if ok && now.Before(entry.expiresAt) {
		report := &llmrepo.MonitoringReport{}
		if err := json.Unmarshal(entry.data, report); err == nil {
			return report, nil
		}
	}

	report, err := r.inner.GetReport(ctx, filter)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(report)
	if err != nil {
		return report, nil
	}

	r.mu.Lock()
	r.cache[key] = &cacheEntry{data: b, expiresAt: now.Add(r.ttl)}
	r.mu.Unlock()

	return report, nil
}
