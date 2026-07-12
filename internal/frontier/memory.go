package frontier

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/DrowsyWings/web-crawler/internal/normalize"
)

var _ Frontier = (*Memory)(nil)

type Memory struct {
	queue    chan Task
	mu       sync.Mutex
	seen     map[string]bool
	inflight int64
}

func NewMemory(capacity int) *Memory {
	return &Memory{
		queue: make(chan Task, capacity),
		seen:  make(map[string]bool),
	}
}

func (m *Memory) Push(ctx context.Context, t Task) (bool, error) {
	key := normalize.Normalize(t.URL)

	m.mu.Lock()
	if m.seen[key] {
		m.mu.Unlock()
		return false, nil
	}
	m.seen[key] = true
	m.mu.Unlock()

	select {
	case m.queue <- t:
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (m *Memory) Claim(ctx context.Context) (Task, func(context.Context) error, bool, error) {
	// Count the claim before dequeuing so Pending never under-counts (a
	// transient over-count is harmless; an under-count would end the crawl early).
	atomic.AddInt64(&m.inflight, 1)
	select {
	case t := <-m.queue:
		var once sync.Once
		ack := func(context.Context) error {
			once.Do(func() { atomic.AddInt64(&m.inflight, -1) })
			return nil
		}
		return t, ack, true, nil
	default:
		atomic.AddInt64(&m.inflight, -1)
		return Task{}, nil, false, nil
	}
}

func (m *Memory) Pending(ctx context.Context) (int64, int64, error) {
	return int64(len(m.queue)), atomic.LoadInt64(&m.inflight), nil
}

func (m *Memory) Close() error { return nil }
