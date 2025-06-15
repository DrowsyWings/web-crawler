package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"web-crawler/pkg/models"

	"github.com/boltdb/bolt"
)

var alreadyVisited = errors.New("visited")

func Init(db *bolt.DB) error {

	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("visited"))
		if err != nil {
			return fmt.Errorf("could not create bucket: %v", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not setup bucket: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("queue"))
		if err != nil {
			return fmt.Errorf("could not create bucket: %v", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not setup bucket: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("results"))
		if err != nil {
			return fmt.Errorf("could not create bucket: %v", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not setup bucket: %v", err)
	}
	fmt.Println("DB intialized.")

	return nil
}

func MarkVisited(db *bolt.DB, url string) error {
	return db.Update(func(tx *bolt.Tx) error {
		a := tx.Bucket([]byte("visited"))

		if a == nil {
			return bolt.ErrBucketNotFound
		}
		return a.Put([]byte(url), []byte(strconv.FormatBool(true)))

	})

}

func IsVisited(db *bolt.DB, url string) (bool, error) {
	err := db.View(func(tx *bolt.Tx) error {
		a := tx.Bucket([]byte("visited"))

		if a == nil {
			return bolt.ErrBucketNotFound
		}
		b := a.Get([]byte(url))

		if b == nil {
			return nil
		}

		return alreadyVisited
	})

	if err != nil {
		return false, err

	} else if err == alreadyVisited {
		return true, nil
	}
	return false, nil
}

func SaveResult(db *bolt.DB, result models.CrawlResult) error {
	data, err := json.Marshal(result)

	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		a := tx.Bucket([]byte("results"))

		if a == nil {
			return bolt.ErrBucketNotFound
		}
		return a.Put([]byte(result.Url), data)
	})
}

func GetQueue(db *bolt.DB) ([]string, error) {
	var queue []string
	err := db.View(func(tx *bolt.Tx) error {
		a := tx.Bucket([]byte("queue"))

		if a == nil {
			return bolt.ErrBucketNotFound
		}

		return a.ForEach(func(k []byte, _ []byte) error {
			queue = append(queue, string(k))
			return nil
		})
	})
	return queue, err
}

func ExportResults(db *bolt.DB) ([]models.CrawlResult, error) {
	var results []models.CrawlResult

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("results"))
		if b == nil {
			return bolt.ErrBucketNotFound
		}

		return b.ForEach(func(k, v []byte) error {
			var result models.CrawlResult
			if err := json.Unmarshal(v, &result); err != nil {
				return err
			}
			results = append(results, result)
			return nil
		})
	})

	return results, err
}
