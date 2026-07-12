package store

import (
	"context"
	"encoding/json"

	"github.com/DrowsyWings/web-crawler/pkg/models"
	bolt "go.etcd.io/bbolt"
)

var _ Store = (*Bolt)(nil)

var resultsBucket = []byte("results")

type Bolt struct {
	db *bolt.DB
}

func OpenBolt(path string) (*Bolt, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(resultsBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Bolt{db: db}, nil
}

func (b *Bolt) SaveResult(ctx context.Context, r models.CrawlResult) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return b.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(resultsBucket).Put([]byte(r.Url), data)
	})
}

func (b *Bolt) ExportResults(ctx context.Context) ([]models.CrawlResult, error) {
	var results []models.CrawlResult
	err := b.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(resultsBucket).ForEach(func(k, v []byte) error {
			var r models.CrawlResult
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			results = append(results, r)
			return nil
		})
	})
	return results, err
}

func (b *Bolt) Close() error { return b.db.Close() }
