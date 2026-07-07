package models

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
