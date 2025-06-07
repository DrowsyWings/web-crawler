package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
	"web-crawler/pkg/models"

	"github.com/boltdb/bolt"
)

var alreadyVisited = errors.New("visited")

func Init() error {
	db, err := bolt.Open("storage.db", 0600, &bolt.Options{Timeout: 1 * time.Second})

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
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
		timestamp := time.Now().Format(time.RFC3339)
		return a.Put([]byte(url), []byte(timestamp))

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

		if a != nil {
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
