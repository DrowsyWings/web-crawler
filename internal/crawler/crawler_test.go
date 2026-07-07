package crawler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DrowsyWings/web-crawler/internal/stats"
	"github.com/DrowsyWings/web-crawler/pkg/models"

	bolt "go.etcd.io/bbolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHttp struct {
	mock.Mock
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

func (m *mockHttp) Get(u string) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	if r, ok := m.res[u]; ok {
		return r, nil
	}
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(htmlData)),
	}, nil
}

func makeTestDb(t *testing.T) *bolt.DB {
	db, e := bolt.Open(":memory:", 0600, nil)
	assert.NoError(t, e)
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucket([]byte("visited"))
		tx.CreateBucket([]byte("results"))
		return nil
	})
	return db
}

func TestNewCrawler(t *testing.T) {
	c := models.CrawlConfig{
		SeedUrl: "https://example.com",
		Depth: "3",
		RateLimits: "5",
		DomainRestrictions: "1s",
	}
	db := makeTestDb(t)
	defer db.Close()
	st := &stats.Stats{}
	client := &mockHttp{}
	cr := NewCrawler(c, db, st, client)
	assert.NotNil(t, cr)
	assert.Equal(t, "example.com", cr.Domain)
	assert.Equal(t, 3, cr.MaxDepth)
	assert.Equal(t, 5, cr.Workers)
	assert.Equal(t, time.Second, cr.Delay)
	assert.Equal(t, c, cr.Config)
	assert.Equal(t, db, cr.DB)
	assert.Equal(t, st, cr.Stats)
	assert.Equal(t, client, cr.HTTPClient)
	assert.NotNil(t, cr.Visited)
	assert.NotNil(t, cr.Queue)
	assert.NotNil(t, cr.done)
}

func TestNewCrawlerDefaults(t *testing.T) {
	c := models.CrawlConfig{
		SeedUrl: "https://example.com",
		Depth: "invalid",
		RateLimits: "invalid",
		DomainRestrictions: "invalid",
	}
	db := makeTestDb(t)
	defer db.Close()
	st := &stats.Stats{}
	client := &mockHttp{}
	cr := NewCrawler(c, db, st, client)
	assert.Equal(t, 2, cr.MaxDepth)
	assert.Equal(t, 4, cr.Workers)
	assert.Equal(t, time.Duration(0), cr.Delay)
}

func TestAddTask(t *testing.T) {
	c := models.CrawlConfig{SeedUrl: "https://example.com"}
	db := makeTestDb(t)
	defer db.Close()
	cr := NewCrawler(c, db, &stats.Stats{}, &mockHttp{})
	task := Task{URL: "https://example.com/test", Depth: 1}
	cr.addTask(task)
	assert.Equal(t, int64(1), cr.pendingWork)
	select {
	case got := <-cr.Queue:
		assert.Equal(t, task.URL, got.URL)
		assert.Equal(t, task.Depth, got.Depth)
	case <-time.After(100 * time.Millisecond):
		t.Error("no task")
	}
}

func TestIsVisitedInMemory(t *testing.T) {
	c := &Crawler{
		Visited: map[string]bool{},
		VisitedM: sync.Mutex{},
	}
	u := "https://example.com"
	assert.False(t, c.isVisitedInMemory(u))
	c.markInMemoryVisited(u)
	assert.True(t, c.isVisitedInMemory(u))
}

func TestMarkInMemoryVisited(t *testing.T) {
	c := &Crawler{
		Visited: map[string]bool{},
		VisitedM: sync.Mutex{},
	}
	u := "https://example.com"
	c.markInMemoryVisited(u)
	c.VisitedM.Lock()
	defer c.VisitedM.Unlock()
	if !c.Visited[u] {
		t.Errorf("expected URL %q visited", u)
	}
}

func TestProcessTaskSuccess(t *testing.T) {
	c := models.CrawlConfig{SeedUrl: "https://example.com"}
	db := makeTestDb(t)
	defer db.Close()
	st := &stats.Stats{
		InProgressCh: make(chan struct{}, 10),
		CompletedCh: make(chan struct{}, 10),
		CrawledCh: make(chan struct{}, 10),
		FoundCh: make(chan struct{}, 10),
		QueueSizeCh: make(chan int, 10),
	}
	client := &mockHttp{}
	cr := NewCrawler(c, db, st, client)
	task := Task{URL: "https://example.com/test", Depth: 1}
	go cr.processTask(task)
	select {
	case <-st.InProgressCh:
	case <-time.After(100 * time.Millisecond):
		t.Error("no inprogress")
	}
	select {
	case <-st.CompletedCh:
	case <-time.After(100 * time.Millisecond):
		t.Error("no complete")
	}
	select {
	case <-st.CrawledCh:
	case <-time.After(100 * time.Millisecond):
		t.Error("no crawled")
	}
}

func TestProcessTaskDepthExceeded(t *testing.T) {
	c := models.CrawlConfig{SeedUrl: "https://example.com", Depth: "1"}
	db := makeTestDb(t)
	defer db.Close()
	st := &stats.Stats{
		InProgressCh: make(chan struct{}, 10),
		CompletedCh: make(chan struct{}, 10),
		FilteredCh: make(chan struct{}, 10),
	}
	cr := NewCrawler(c, db, st, &mockHttp{})
	task := Task{URL: "https://example.com/test", Depth: 5}
	go cr.processTask(task)
	select {
	case <-st.FilteredCh:
	case <-time.After(100 * time.Millisecond):
		t.Error("no filter")
	}
}

func TestProcessTaskHTTPError(t *testing.T) {
	c := models.CrawlConfig{SeedUrl: "https://example.com"}
	db := makeTestDb(t)
	defer db.Close()
	st := &stats.Stats{
		InProgressCh: make(chan struct{}, 10),
		CompletedCh: make(chan struct{}, 10),
		ErrorCh: make(chan struct{}, 10),
	}
	client := &mockHttp{
		err: fmt.Errorf("network error"),
	}
	cr := NewCrawler(c, db, st, client)
	task := Task{URL: "https://example.com/test", Depth: 1}
	go cr.processTask(task)
	select {
	case <-st.ErrorCh:
	case <-time.After(100 * time.Millisecond):
		t.Error("no error ch")
	}
}

func TestProcessTaskNon200Status(t *testing.T) {
	c := models.CrawlConfig{SeedUrl: "https://example.com"}
	db := makeTestDb(t)
	defer db.Close()
	st := &stats.Stats{
		InProgressCh: make(chan struct{}, 10),
		CompletedCh: make(chan struct{}, 10),
		ErrorCh: make(chan struct{}, 10),
	}
	client := &mockHttp{
		res: map[string]*http.Response{
			"https://example.com/test": {
				StatusCode: 404,
				Body: io.NopCloser(strings.NewReader("")),
			},
		},
	}
	cr := NewCrawler(c, db, st, client)
	task := Task{URL: "https://example.com/test", Depth: 1}
	go cr.processTask(task)
	select {
	case <-st.ErrorCh:
	case <-time.After(100 * time.Millisecond):
		t.Error("no error")
	}
}

func TestRunWorker(t *testing.T) {
	c := models.CrawlConfig{SeedUrl: "https://example.com"}
	db := makeTestDb(t)
	defer db.Close()
	st := &stats.Stats{
		InProgressCh: make(chan struct{}, 10),
		CompletedCh: make(chan struct{}, 10),
		CrawledCh: make(chan struct{}, 10),
		QueueSizeCh: make(chan int, 10),
		FoundCh: make(chan struct{}, 10),
		ErrorCh: make(chan struct{}, 10),
		FilteredCh: make(chan struct{}, 10),
		DuplicateCh: make(chan struct{}, 10),
	}
	cr := NewCrawler(c, db, st, &mockHttp{})
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-st.InProgressCh:
			case <-st.CompletedCh:
			case <-st.CrawledCh:
			case <-st.QueueSizeCh:
			case <-st.FoundCh:
			case <-st.ErrorCh:
			case <-st.FilteredCh:
			case <-st.DuplicateCh:
			case <-done:
				return
			}
		}
	}()
	defer close(done)
	task := Task{URL: "https://example.com/test", Depth: 1}
	cr.addTask(task)
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(cr.Queue)
	}()
	cr.WG.Add(1)
	go cr.runWorker()
	workerDone := make(chan bool)
	go func() {
		cr.WG.Wait()
		workerDone <- true
	}()
	select {
	case <-workerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("worker timeout")
	}
	assert.Equal(t, int64(0), cr.activeWorkers)
}

func TestMonitorCompletion(t *testing.T) {
	c := models.CrawlConfig{SeedUrl: "https://example.com"}
	db := makeTestDb(t)
	defer db.Close()
	cr := NewCrawler(c, db, &stats.Stats{}, &mockHttp{})
	go cr.monitorCompletion()
	select {
	case <-cr.done:
	case <-time.After(2 * time.Second):
		t.Error("no done")
	}
}

func TestConcurrentVisitedAccess(t *testing.T) {
	c := &Crawler{
		Visited: map[string]bool{},
		VisitedM: sync.Mutex{},
	}
	var wg sync.WaitGroup
	uList := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/3",
	}
	for _, u := range uList {
		wg.Add(1)
		go func(x string) {
			defer wg.Done()
			c.markInMemoryVisited(x)
		}(u)
	}
	wg.Wait()
	for _, u := range uList {
		assert.True(t, c.isVisitedInMemory(u))
	}
}
