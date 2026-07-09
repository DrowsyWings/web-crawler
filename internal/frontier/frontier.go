package frontier

import "context"

type Task struct {
	URL   string
	Depth int
}

type Frontier interface {
	// added is false when the URL was deduplicated.
	Push(ctx context.Context, t Task) (added bool, err error)

	// ack must be called once the task is processed to release its lease;
	// ok is false when nothing is available right now.
	Claim(ctx context.Context) (t Task, ack func(context.Context) error, ok bool, err error)

	// The crawl is complete when both counts reach zero.
	Pending(ctx context.Context) (queued, inflight int64, err error)

	Close() error
}
