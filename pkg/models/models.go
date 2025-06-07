package models

import "time"

type CrawlResult struct {
	Url       string
	Title     string
	Timestamp string
	Status    string
}

type CrawlConfig struct {
	seedUrl            string
	depth              string
	domainRestrictions string
	rateLimits         string
}

type CrawlStats struct {
	pagesCrawled int16
	queued       int16
	errors       string
	elapsedTime  time.Time
}
