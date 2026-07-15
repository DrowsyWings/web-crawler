package frontier

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRedisClient(t *testing.T) redis.UniversalClient {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	return client
}

func TestRedisConformance(t *testing.T) {
	runConformance(t, func() Frontier {
		return NewRedis(newTestRedisClient(t), "testjob", time.Minute)
	})
}

func TestRedisReaperRequeuesExpiredLease(t *testing.T) {
	ctx := context.Background()
	// Short visibility so the lease expires in real time (deadlines use the
	// client clock, not a Redis TTL, so FastForward wouldn't apply here).
	f := NewRedis(newTestRedisClient(t), "job", 20*time.Millisecond)

	added, err := f.Push(ctx, Task{URL: "https://a.com/x", Depth: 1})
	require.NoError(t, err)
	require.True(t, added)

	task, _, ok, err := f.Claim(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "https://a.com/x", task.URL)

	queued, inflight, _ := f.Pending(ctx)
	assert.Equal(t, int64(0), queued)
	assert.Equal(t, int64(1), inflight)

	time.Sleep(40 * time.Millisecond)

	n, err := f.Reap(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	queued, inflight, _ = f.Pending(ctx)
	assert.Equal(t, int64(1), queued)
	assert.Equal(t, int64(0), inflight)
}
