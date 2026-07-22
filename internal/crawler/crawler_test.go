package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DrowsyWings/web-crawler/internal/frontier"
	"github.com/DrowsyWings/web-crawler/internal/stats"
	"github.com/DrowsyWings/web-crawler/pkg/models"

	"github.com/stretchr/testify/assert"
)

type mockHttp struct {
	res map[string]*http.Response
	err error
}

var htmlData = `
	<html>
	<head>
		<title>Test Page</title>
		<meta name="description" content="Just testing">
		<meta name="keywords" content="test,test,test">
	</head>
	<body>
		Hello World
		<a href="/test1">test 1</a>
		<a href="https://example.com/test2">test 2</a>
		<a href="https://test3.com/">test3</a>
	</body>
	</html>
`

func (m *mockHttp) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	if r, ok := m.res[req.URL.String()]; ok {
		return r, nil
	}
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(htmlData)),
	}, nil
}

type fakeStore struct {
	mu      sync.Mutex
	results []models.CrawlResult
}

func (s *fakeStore) SaveResult(ctx context.Context, r models.CrawlResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, r)
	return nil
}

func (s *fakeStore) ExportResults(ctx context.Context) ([]models.CrawlResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.CrawlResult(nil), s.results...), nil
}

func (s *fakeStore) Close() error { return nil }

func newTestStats() *stats.Stats {
	return &stats.Stats{
		CrawledCh:    make(chan struct{}, 100),
		FoundCh:      make(chan struct{}, 100),
		DuplicateCh:  make(chan struct{}, 100),
		FilteredCh:   make(chan struct{}, 100),
		InProgressCh: make(chan struct{}, 100),
		CompletedCh:  make(chan struct{}, 100),
		ErrorCh:      make(chan struct{}, 100),
		QueueSizeCh:  make(chan int, 100),
	}
}

func assertRecv[T any](t *testing.T, ch <-chan T, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Errorf("expected %s signal", name)
	}
}

func TestNewCrawler(t *testing.T) {
	cfg := models.Config{SeedURL: "https://example.com", MaxDepth: 3, Workers: 5, Delay: time.Second}
	st := &stats.Stats{}
	client := &mockHttp{}
	cr := NewCrawler(cfg, frontier.NewMemory(10), &fakeStore{}, st, client)

	assert.Equal(t, "example.com", cr.Domain)
	assert.Equal(t, 3, cr.MaxDepth)
	assert.Equal(t, 5, cr.Workers)
	assert.Equal(t, time.Second, cr.Delay)
	assert.Equal(t, cfg, cr.Config)
	assert.Equal(t, st, cr.Stats)
	assert.Equal(t, client, cr.HTTPClient)
	assert.NotNil(t, cr.Frontier)
	assert.NotNil(t, cr.Store)
}

func TestProcessTaskSuccess(t *testing.T) {
	st := newTestStats()
	store := &fakeStore{}
	cr := NewCrawler(models.Config{SeedURL: "https://example.com", MaxDepth: 2}, frontier.NewMemory(100), store, st, &mockHttp{})

	go cr.processTask(context.Background(), frontier.Task{URL: "https://example.com/test", Depth: 1})

	assertRecv(t, st.InProgressCh, "inprogress")
	assertRecv(t, st.CrawledCh, "crawled")
	assertRecv(t, st.CompletedCh, "complete")

	results, _ := store.ExportResults(context.Background())
	assert.Len(t, results, 1)
	assert.Equal(t, "Test Page", results[0].Title)
	assert.Equal(t, "Just testing", results[0].Description)
	assert.Equal(t, "test,test,test", results[0].Keywords)
}

func TestProcessTaskDepthExceeded(t *testing.T) {
	st := newTestStats()
	cr := NewCrawler(models.Config{SeedURL: "https://example.com", MaxDepth: 1}, frontier.NewMemory(100), &fakeStore{}, st, &mockHttp{})

	go cr.processTask(context.Background(), frontier.Task{URL: "https://example.com/test", Depth: 5})

	assertRecv(t, st.FilteredCh, "filtered")
}

func TestProcessTaskHTTPError(t *testing.T) {
	st := newTestStats()
	client := &mockHttp{err: fmt.Errorf("network error")}
	cr := NewCrawler(models.Config{SeedURL: "https://example.com", MaxDepth: 2}, frontier.NewMemory(100), &fakeStore{}, st, client)

	go cr.processTask(context.Background(), frontier.Task{URL: "https://example.com/test", Depth: 1})

	assertRecv(t, st.ErrorCh, "error")
}

func TestProcessTaskNon200Status(t *testing.T) {
	st := newTestStats()
	client := &mockHttp{
		res: map[string]*http.Response{
			"https://example.com/test": {
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("")),
			},
		},
	}
	cr := NewCrawler(models.Config{SeedURL: "https://example.com", MaxDepth: 2}, frontier.NewMemory(100), &fakeStore{}, st, client)

	go cr.processTask(context.Background(), frontier.Task{URL: "https://example.com/test", Depth: 1})

	assertRecv(t, st.ErrorCh, "error")
}

func TestProcessTaskEnqueuesSameDomainLinks(t *testing.T) {
	st := newTestStats()
	f := frontier.NewMemory(100)
	cr := NewCrawler(models.Config{SeedURL: "https://example.com", MaxDepth: 2}, f, &fakeStore{}, st, &mockHttp{})

	cr.processTask(context.Background(), frontier.Task{URL: "https://example.com", Depth: 0})

	// htmlData exposes two same-domain links; the external one is dropped by the parser.
	queued, _, _ := f.Pending(context.Background())
	assert.Equal(t, int64(2), queued)
}

func TestStartCrawlCompletes(t *testing.T) {
	st := stats.NewStats()
	store := &fakeStore{}
	cr := NewCrawler(models.Config{SeedURL: "https://example.com", MaxDepth: 1, Workers: 2}, frontier.NewMemory(1000), store, st, &mockHttp{})

	done := make(chan struct{})
	go func() {
		cr.Start(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("crawl did not complete")
	}

	results, _ := store.ExportResults(context.Background())
	assert.NotEmpty(t, results)
}
