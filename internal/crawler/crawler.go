package crawler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"web-crawler/internal/parser"
	"web-crawler/internal/stats"
	"web-crawler/internal/storage"
	"web-crawler/pkg/models"

	"github.com/boltdb/bolt"
)

type Task struct {
	URL   string
	Depth int
}

type Crawler struct {
	Config   models.CrawlConfig
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
}

func NewCrawler(config models.CrawlConfig, db *bolt.DB, stats *stats.Stats) *Crawler {
	parsedURL, _ := url.Parse(config.SeedUrl)

	depth := 2
	fmt.Sscanf(config.Depth, "%d", &depth)

	workers := 4
	fmt.Sscanf(config.RateLimits, "%d", &workers)

	delay := 0 * time.Second
	if d, err := time.ParseDuration(config.DomainRestrictions); err == nil {
		delay = d
	}

	return &Crawler{
		Config:   config,
		DB:       db,
		Visited:  make(map[string]bool),
		Queue:    make(chan Task, 1000),
		Domain:   parsedURL.Host,
		MaxDepth: depth,
		Workers:  workers,
		Delay:    delay,
		Stats:    stats,
	}
}

func (c *Crawler) Start() {
	c.Queue <- Task{URL: c.Config.SeedUrl, Depth: 0}
	c.Stats.QueuedCh <- struct{}{}

	go c.Stats.StartReporting()
	defer func() { c.Stats.DoneCh <- struct{}{} }()

	for i := 0; i < c.Workers; i++ {
		c.WG.Add(1)
		go c.runWorker()
	}

	c.WG.Wait()
	close(c.Queue)
	fmt.Println("Crawling complete.")
}

func (c *Crawler) runWorker() {
	defer c.WG.Done()

	for task := range c.Queue {
		if task.Depth > c.MaxDepth {
			continue
		}

		if isVisited, _ := storage.IsVisited(c.DB, task.URL); isVisited || c.isVisitedInMemory(task.URL) {
			continue
		}

		res, err := http.Get(task.URL)
		if err != nil || res.StatusCode != 200 {
			c.Stats.ErrorCh <- struct{}{}
			continue
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			c.Stats.ErrorCh <- struct{}{}
			continue
		}

		result, err := parser.ParseHTML(task.URL, string(body))
		if err != nil {
			c.Stats.ErrorCh <- struct{}{}
			continue
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
				continue
			}
			if !c.isVisitedInMemory(link) {
				c.Queue <- Task{URL: link, Depth: task.Depth + 1}
				c.Stats.QueuedCh <- struct{}{}
			}
		}

		time.Sleep(c.Delay)
	}
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
