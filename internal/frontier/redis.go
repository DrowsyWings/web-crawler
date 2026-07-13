package frontier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DrowsyWings/web-crawler/internal/normalize"
	"github.com/redis/go-redis/v9"
)

var _ Frontier = (*Redis)(nil)

type Redis struct {
	client     redis.UniversalClient
	queueKey   string
	procKey    string
	seenKey    string
	visibility time.Duration
}

// NewRedis builds a frontier over keys namespaced by jobID, so instances that
// share a jobID cooperate on one crawl. visibility is the lease duration after
// which an unacked (crashed-worker) task is reclaimable by the reaper.
func NewRedis(client redis.UniversalClient, jobID string, visibility time.Duration) *Redis {
	prefix := "crawl:" + jobID
	return &Redis{
		client:     client,
		queueKey:   prefix + ":queue",
		procKey:    prefix + ":processing",
		seenKey:    prefix + ":seen",
		visibility: visibility,
	}
}

var enqueueScript = redis.NewScript(`
if redis.call('SADD', KEYS[1], ARGV[1]) == 1 then
	redis.call('RPUSH', KEYS[2], ARGV[2])
	return 1
end
return 0
`)

// Pop one task and record its lease atomically, so pending never drops to zero
// while a worker holds a task it is about to expand.
var claimScript = redis.NewScript(`
local item = redis.call('LPOP', KEYS[1])
if not item then return false end
redis.call('ZADD', KEYS[2], ARGV[1], item)
return item
`)

var pendingScript = redis.NewScript(`
return {redis.call('LLEN', KEYS[1]), redis.call('ZCARD', KEYS[2])}
`)

func (r *Redis) Push(ctx context.Context, t Task) (bool, error) {
	payload, err := json.Marshal(t)
	if err != nil {
		return false, err
	}
	n, err := enqueueScript.Run(ctx, r.client,
		[]string{r.seenKey, r.queueKey},
		normalize.Normalize(t.URL), payload,
	).Int()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (r *Redis) Claim(ctx context.Context) (Task, func(context.Context) error, bool, error) {
	deadline := time.Now().Add(r.visibility).UnixMilli()
	res, err := claimScript.Run(ctx, r.client, []string{r.queueKey, r.procKey}, deadline).Result()
	if err == redis.Nil {
		return Task{}, nil, false, nil
	}
	if err != nil {
		return Task{}, nil, false, err
	}

	payload, ok := res.(string)
	if !ok {
		return Task{}, nil, false, fmt.Errorf("unexpected claim result: %T", res)
	}
	var t Task
	if err := json.Unmarshal([]byte(payload), &t); err != nil {
		return Task{}, nil, false, err
	}

	ack := func(ctx context.Context) error {
		return r.client.ZRem(ctx, r.procKey, payload).Err()
	}
	return t, ack, true, nil
}

func (r *Redis) Pending(ctx context.Context) (int64, int64, error) {
	res, err := pendingScript.Run(ctx, r.client, []string{r.queueKey, r.procKey}).Slice()
	if err != nil {
		return 0, 0, err
	}
	if len(res) != 2 {
		return 0, 0, fmt.Errorf("unexpected pending result: %v", res)
	}
	queued, _ := res[0].(int64)
	inflight, _ := res[1].(int64)
	return queued, inflight, nil
}

func (r *Redis) Close() error { return r.client.Close() }
