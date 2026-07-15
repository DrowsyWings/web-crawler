package store

import (
	"context"
	"encoding/json"

	"github.com/DrowsyWings/web-crawler/pkg/models"
	"github.com/redis/go-redis/v9"
)

var _ Store = (*Redis)(nil)

type Redis struct {
	client     redis.UniversalClient
	resultsKey string
}

func NewRedis(client redis.UniversalClient, jobID string) *Redis {
	return &Redis{
		client:     client,
		resultsKey: "crawl:" + jobID + ":results",
	}
}

func (r *Redis) SaveResult(ctx context.Context, res models.CrawlResult) error {
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return r.client.HSet(ctx, r.resultsKey, res.Url, data).Err()
}

func (r *Redis) ExportResults(ctx context.Context) ([]models.CrawlResult, error) {
	vals, err := r.client.HGetAll(ctx, r.resultsKey).Result()
	if err != nil {
		return nil, err
	}
	results := make([]models.CrawlResult, 0, len(vals))
	for _, v := range vals {
		var res models.CrawlResult
		if err := json.Unmarshal([]byte(v), &res); err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func (r *Redis) Close() error { return r.client.Close() }
