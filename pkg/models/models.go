package models

import "time"

type CrawlResult struct {
	Url       string
	Title     string
	Timestamp string
	Status    string
}

type Config struct {
	SeedURL  string
	MaxDepth int
	Workers  int
	Delay    time.Duration
	Timeout  time.Duration
	JobID    string
}
