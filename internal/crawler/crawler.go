package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DrowsyWings/web-crawler/internal/parser"
	"github.com/DrowsyWings/web-crawler/internal/stats"
	"github.com/DrowsyWings/web-crawler/internal/storage"
	"github.com/DrowsyWings/web-crawler/pkg/models"

	bolt "go.etcd.io/bbolt"
)
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
type Task struct {
	URL   string
	Depth int
}

type Crawler struct {
	Config   models.Config
	DB       *bolt.DB
	Visited  map[string]bool
	VisitedM sync.Mutex
	Queue    chan Task
	WG       sync.WaitGroup
	Domain   string
	MaxDepth int
	Workers  int
	Delay    time.Duration
	Stats    *stats.Stats

	activeWorkers int64
	pendingWork   int64 
	done          chan struct{}
	HTTPClient HTTPClient
}

func NewCrawler(config models.Config, db *bolt.DB, stats *stats.Stats, client HTTPClient) *Crawler {
	parsedURL, _ := url.Parse(config.SeedURL)

	return &Crawler{
		Config:   config,
		DB:       db,
		Visited:  make(map[string]bool),
		Queue:    make(chan Task, 1000),
		Domain:   parsedURL.Host,
		MaxDepth: config.MaxDepth,
		Workers:  config.Workers,
		Delay:    config.Delay,
		Stats:    stats,
		done:     make(chan struct{}),
		HTTPClient: client,
	}
}

func (c *Crawler) Start(ctx context.Context) {

	go c.Stats.StartReporting()
	defer func() { c.Stats.DoneCh <- struct{}{} }()

	c.addTask(Task{URL: c.Config.SeedURL, Depth: 0})

	for i := 0; i < c.Workers; i++ {
		c.WG.Add(1)
		go c.runWorker(ctx)
	}

	go c.monitorCompletion()

	select {
	case <-c.done:
		// Crawl finished naturally: close the queue so workers drain and exit.
		close(c.Queue)
	case <-ctx.Done():
		// Interrupted: workers exit via ctx; leave the queue open so a worker
		// mid-task can't send on a closed channel.
		fmt.Println("Shutdown requested, stopping...")
	}
	c.WG.Wait()
	fmt.Println("Crawling complete.")
}

func (c *Crawler) addTask(task Task) {
	c.Queue <- task
	atomic.AddInt64(&c.pendingWork, 1)
}

func (c *Crawler) runWorker(ctx context.Context) {
	defer c.WG.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-c.Queue:
			if !ok {
				return
			}
			atomic.AddInt64(&c.activeWorkers, 1)
			atomic.AddInt64(&c.pendingWork, -1)

			c.processTask(ctx, task)

			atomic.AddInt64(&c.activeWorkers, -1)
		}
	}
}

func (c *Crawler) monitorCompletion() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if atomic.LoadInt64(&c.pendingWork) == 0 && atomic.LoadInt64(&c.activeWorkers) == 0 {
				time.Sleep(100 * time.Millisecond)
				if atomic.LoadInt64(&c.pendingWork) == 0 && atomic.LoadInt64(&c.activeWorkers) == 0 {
					close(c.done)
					return
				}
			}
		}
	}
}

func (c *Crawler) processTask(ctx context.Context, task Task) {
	c.Stats.InProgressCh <- struct{}{}
	defer func() { c.Stats.CompletedCh <- struct{}{} }()

	if task.Depth > c.MaxDepth {
		c.Stats.FilteredCh <- struct{}{}
		return
	}

		if isVisited, _ := storage.IsVisited(c.DB, task.URL); isVisited || c.isVisitedInMemory(task.URL) {
			c.Stats.DuplicateCh <- struct{}{}
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
		if err != nil {
			c.Stats.ErrorCh <- struct{}{}
			return
		}
		res, err := c.HTTPClient.Do(req)
		if err != nil || res.StatusCode != 200 {
			c.Stats.ErrorCh <- struct{}{}
			return
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			c.Stats.ErrorCh <- struct{}{}
			return
		}

		result, err := parser.ParseHTML(task.URL, string(body))
		if err != nil {
			c.Stats.ErrorCh <- struct{}{}
			return
		}

		storage.MarkVisited(c.DB, task.URL)
		c.markInMemoryVisited(task.URL)

		crawlData := models.CrawlResult{
			Url:       task.URL,
			Title:     result.Title,
			Timestamp: time.Now().Format(time.RFC3339),
			Status:    res.Status,
		}
		storage.SaveResult(c.DB, crawlData)
		c.Stats.CrawledCh <- struct{}{}

		for _, link := range result.Links {
			linkURL, _ := url.Parse(link)
			if linkURL.Host != c.Domain {
				c.Stats.FilteredCh <- struct{}{}
				continue
			}
			if !c.isVisitedInMemory(link) {
				select {
				case <-c.done:
					return
				default:
					c.addTask(Task{URL: link, Depth: task.Depth + 1})
					c.Stats.FoundCh <- struct{}{}
				}
			} else {
			c.Stats.DuplicateCh <- struct{}{} 
			}
		}
	c.Stats.QueueSizeCh <- len(c.Queue)

		time.Sleep(c.Delay)

}

func (c *Crawler) isVisitedInMemory(url string) bool {
	c.VisitedM.Lock()
	defer c.VisitedM.Unlock()
	return c.Visited[url]
}

func (c *Crawler) markInMemoryVisited(url string) {
	c.VisitedM.Lock()
	defer c.VisitedM.Unlock()
	c.Visited[url] = true
}
