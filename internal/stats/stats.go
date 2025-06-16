package stats

import (
	"log"
	"time"
)

type Stats struct {
	Crawled int
	InProgress int
	QueueSize  int
	Errors  int

	TotalFound int
	Duplicates int
	Filtered   int
	Start   time.Time

	CrawledCh chan struct{}
	FoundCh   chan struct{}
	DuplicateCh  chan struct{}
	FilteredCh   chan struct{}
	InProgressCh chan struct{}
	CompletedCh  chan struct{}
	ErrorCh   chan struct{}
	QueueSizeCh  chan int
	DoneCh    chan struct{}
}

func NewStats() *Stats {
	return &Stats{
		CrawledCh: make(chan struct{}, 100),
		FoundCh:      make(chan struct{}, 100),
		DuplicateCh:  make(chan struct{}, 100),
		FilteredCh:   make(chan struct{}, 100),
		InProgressCh: make(chan struct{}, 100),
		CompletedCh:  make(chan struct{}, 100),
		ErrorCh:   make(chan struct{}, 100),
		QueueSizeCh:  make(chan int, 100),
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
		case <-s.FoundCh:
			s.TotalFound++
		case <-s.DuplicateCh:
			s.Duplicates++
		case <-s.FilteredCh:
			s.Filtered++
		case <-s.InProgressCh:
			s.InProgress++
		case <-s.CompletedCh:
			s.InProgress--
		case <-s.ErrorCh:
			s.Errors++
		case size := <-s.QueueSizeCh:
			s.QueueSize = size
		case <-ticker.C:
			s.printStats()
		case <-s.DoneCh:
			ticker.Stop()
			s.printFinalStats()
			log.Println("Stats reporting stopped.")
			return
		}
	}
}

func (s *Stats) printStats() {
	elapsed := time.Since(s.Start)
	log.Printf("[CRAWL] Processing: %d | Queue: %d | Crawled: %d | Found: %d | Errors: %d | Time: %s",
		s.InProgress, s.QueueSize, s.Crawled, s.TotalFound, s.Errors, elapsed.Truncate(time.Second))
}

func (s *Stats) printFinalStats() {
	elapsed := time.Since(s.Start)
	log.Printf("\n"+
		"=== CRAWL COMPLETE ===\n"+
		"✓ Pages crawled:     %d\n"+
		"✓ URLs discovered:   %d\n"+
		"• Duplicates skipped: %d\n"+
		"• Filtered out:      %d\n"+
		"• Errors:            %d\n"+
		"• Total time:        %s\n"+
		"======================",
		s.Crawled, s.TotalFound, s.Duplicates, s.Filtered, s.Errors, elapsed.Truncate(time.Second))
}
