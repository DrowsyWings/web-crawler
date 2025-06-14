package stats

import (
	"log"
	"time"
)

type Stats struct {
	Crawled int
	Queued  int
	Errors  int
	Start   time.Time

	CrawledCh chan struct{}
	QueuedCh  chan struct{}
	ErrorCh   chan struct{}
	DoneCh    chan struct{}
}

func NewStats() *Stats {
	return &Stats{
		CrawledCh: make(chan struct{}, 100),
		QueuedCh:  make(chan struct{}, 100),
		ErrorCh:   make(chan struct{}, 100),
		DoneCh:    make(chan struct{}),
	}
}

func (s *Stats) StartReporting() {
	s.Start = time.Now()
	ticker := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-s.CrawledCh:
			s.Crawled++
		case <-s.QueuedCh:
			s.Queued++
		case <-s.ErrorCh:
			s.Errors++
		case <-ticker.C:
			s.printStats()
		case <-s.DoneCh:
			ticker.Stop()
			s.printStats()
			log.Println("Stats reporting stopped.")
			return
		}
	}
}

func (s *Stats) printStats() {
	elapsed := time.Since(s.Start)
	log.Printf("[STATS] Crawled: %d | Queued: %d | Errors: %d | Elapsed: %s\n",
		s.Crawled, s.Queued, s.Errors, elapsed.Truncate(time.Second))
}
