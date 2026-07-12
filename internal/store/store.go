package store

import (
	"context"

	"github.com/DrowsyWings/web-crawler/pkg/models"
)

type Store interface {
	SaveResult(ctx context.Context, r models.CrawlResult) error
	ExportResults(ctx context.Context) ([]models.CrawlResult, error)
	Close() error
}
