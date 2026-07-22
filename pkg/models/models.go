package models

import "time"

type CrawlResult struct {
	Url         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
	Timestamp   string `json:"timestamp"`
	Status      string `json:"status"`
}

type Config struct {
	SeedURL  string
	MaxDepth int
	Workers  int
	Delay    time.Duration
	Timeout  time.Duration
	JobID    string
}
