package models

import "time"

type CrawlResult struct {
	Url       string
	Title     string
	Timestamp string
	Status    string
}

type CrawlConfig struct {
	SeedUrl            string
	Depth              string
	DomainRestrictions string
	RateLimits         string
}

type CrawlStats struct {
	PagesCrawled int16
	Queued       int16
	Errors       string
	ElapsedTime  time.Time
}
