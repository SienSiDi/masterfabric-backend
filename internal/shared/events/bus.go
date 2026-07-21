package events

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Topic     string    `json:"topic"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

type Handler func(ctx context.Context, event Event) error

type Bus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(topic string, handler Handler)
}

type inProcessBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewInProcessBus() Bus {
	return &inProcessBus{handlers: make(map[string][]Handler)}
}

func NewEvent(eventType, topic, source string, payload any) Event {
	return Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Topic:     topic,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

func (b *inProcessBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	hs := b.handlers[event.Topic]
	b.mu.RUnlock()
	var wg sync.WaitGroup
	errs := make([]error, len(hs))
	for i, h := range hs {
		wg.Add(1)
		go func(fn Handler, idx int) {
			defer wg.Done()
			errs[idx] = fn(ctx, event)
		}(h, i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *inProcessBus) Subscribe(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}
