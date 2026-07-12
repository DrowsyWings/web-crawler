package frontier

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// runConformance exercises the behavior every Frontier must share. Each
// implementation (memory, redis) runs it against a fresh instance.
func runConformance(t *testing.T, newFrontier func() Frontier) {
	ctx := context.Background()

	t.Run("push dedups repeats", func(t *testing.T) {
		f := newFrontier()
		defer f.Close()

		added, err := f.Push(ctx, Task{URL: "https://a.com/x"})
		assert.NoError(t, err)
		assert.True(t, added)

		added, err = f.Push(ctx, Task{URL: "https://a.com/x"})
		assert.NoError(t, err)
		assert.False(t, added)
	})

	t.Run("push dedups on normalized url", func(t *testing.T) {
		f := newFrontier()
		defer f.Close()

		a, _ := f.Push(ctx, Task{URL: "https://a.com/x"})
		b, _ := f.Push(ctx, Task{URL: "https://a.com/x/"})
		assert.True(t, a)
		assert.False(t, b)
	})

	t.Run("claim on empty returns not ok", func(t *testing.T) {
		f := newFrontier()
		defer f.Close()

		_, _, ok, err := f.Claim(ctx)
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("claim returns task and tracks inflight until ack", func(t *testing.T) {
		f := newFrontier()
		defer f.Close()

		f.Push(ctx, Task{URL: "https://a.com/x", Depth: 2})

		task, ack, ok, err := f.Claim(ctx)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "https://a.com/x", task.URL)
		assert.Equal(t, 2, task.Depth)

		queued, inflight, _ := f.Pending(ctx)
		assert.Equal(t, int64(0), queued)
		assert.Equal(t, int64(1), inflight)

		assert.NoError(t, ack(ctx))

		queued, inflight, _ = f.Pending(ctx)
		assert.Equal(t, int64(0), queued)
		assert.Equal(t, int64(0), inflight)
	})

	t.Run("concurrent push of same url admits exactly one", func(t *testing.T) {
		f := newFrontier()
		defer f.Close()

		var wg sync.WaitGroup
		var admitted int64
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if added, _ := f.Push(ctx, Task{URL: "https://a.com/same"}); added {
					atomic.AddInt64(&admitted, 1)
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, int64(1), admitted)
	})
}
