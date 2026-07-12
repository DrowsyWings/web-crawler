package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/DrowsyWings/web-crawler/internal/frontier"
	"github.com/DrowsyWings/web-crawler/internal/parser"
	"github.com/DrowsyWings/web-crawler/internal/stats"
	"github.com/DrowsyWings/web-crawler/internal/store"
	"github.com/DrowsyWings/web-crawler/pkg/models"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Crawler struct {
	Config     models.Config
	Frontier   frontier.Frontier
	Store      store.Store
	WG         sync.WaitGroup
	Domain     string
	MaxDepth   int
	Workers    int
	Delay      time.Duration
	Stats      *stats.Stats
	HTTPClient HTTPClient
}

func NewCrawler(config models.Config, f frontier.Frontier, s store.Store, st *stats.Stats, client HTTPClient) *Crawler {
	parsedURL, _ := url.Parse(config.SeedURL)

	return &Crawler{
		Config:     config,
		Frontier:   f,
		Store:      s,
		Domain:     parsedURL.Host,
		MaxDepth:   config.MaxDepth,
		Workers:    config.Workers,
		Delay:      config.Delay,
		Stats:      st,
		HTTPClient: client,
	}
}

func (c *Crawler) Start(ctx context.Context) {
	go c.Stats.StartReporting()
	defer func() { c.Stats.DoneCh <- struct{}{} }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c.Frontier.Push(ctx, frontier.Task{URL: c.Config.SeedURL, Depth: 0})

	for i := 0; i < c.Workers; i++ {
		c.WG.Add(1)
		go c.runWorker(ctx)
	}

	go c.monitorCompletion(ctx, cancel)

	c.WG.Wait()
	fmt.Println("Crawling complete.")
}

func (c *Crawler) runWorker(ctx context.Context) {
	defer c.WG.Done()

	for {
		if ctx.Err() != nil {
			return
		}
		task, ack, ok, err := c.Frontier.Claim(ctx)
		if err != nil {
			return
		}
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		c.processTask(ctx, task)
		ack(ctx) // release the lease only after children are enqueued
	}
}

// monitorCompletion ends the crawl once nothing is queued or in flight.
func (c *Crawler) monitorCompletion(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queued, inflight, err := c.Frontier.Pending(ctx)
			if err != nil {
				return
			}
			if queued == 0 && inflight == 0 {
				cancel()
				return
			}
		}
	}
}

func (c *Crawler) processTask(ctx context.Context, task frontier.Task) {
	c.Stats.InProgressCh <- struct{}{}
	defer func() { c.Stats.CompletedCh <- struct{}{} }()

	if task.Depth > c.MaxDepth {
		c.Stats.FilteredCh <- struct{}{}
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

	c.Store.SaveResult(ctx, models.CrawlResult{
		Url:       task.URL,
		Title:     result.Title,
		Timestamp: time.Now().Format(time.RFC3339),
		Status:    res.Status,
	})
	c.Stats.CrawledCh <- struct{}{}

	for _, link := range result.Links {
		linkURL, _ := url.Parse(link)
		if linkURL.Host != c.Domain {
			c.Stats.FilteredCh <- struct{}{}
			continue
		}
		added, _ := c.Frontier.Push(ctx, frontier.Task{URL: link, Depth: task.Depth + 1})
		if added {
			c.Stats.FoundCh <- struct{}{}
		} else {
			c.Stats.DuplicateCh <- struct{}{}
		}
	}

	if queued, _, err := c.Frontier.Pending(ctx); err == nil {
		c.Stats.QueueSizeCh <- int(queued)
	}

	time.Sleep(c.Delay)
}
