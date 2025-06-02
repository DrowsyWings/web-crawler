package storage

import (
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

func init() error {
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
