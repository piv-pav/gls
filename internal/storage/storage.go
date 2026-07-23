package storage

import (
	"fmt"
	"time"

	"github.com/piv-pav/gls/internal/index"
	bolt "go.etcd.io/bbolt"
)

const (
	indexBucket = "index"
	metaBucket  = "meta"
)

// Storage handles persistent storage using BoltDB
type Storage struct {
	db   *bolt.DB
	path string
}

// NewStorage creates a new storage instance
func NewStorage(path string) (*Storage, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(indexBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(metaBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &Storage{
		db:   db,
		path: path,
	}, nil
}

// SaveIndex saves the inverted index to disk
func (s *Storage) SaveIndex(idx *index.InvertedIndex) error {
	data, err := idx.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize index: %w", err)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(indexBucket))
		return bucket.Put([]byte("main"), data)
	})
}

// LoadIndex loads the inverted index from disk
func (s *Storage) LoadIndex() (*index.InvertedIndex, error) {
	idx := index.NewInvertedIndex()

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(indexBucket))
		data := bucket.Get([]byte("main"))
		
		if data == nil {
			// No index stored yet
			return nil
		}

		return idx.Deserialize(data)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return idx, nil
}

// SaveMetadata saves metadata to disk
func (s *Storage) SaveMetadata(key string, value []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(metaBucket))
		return bucket.Put([]byte(key), value)
	})
}

// LoadMetadata loads metadata from disk
func (s *Storage) LoadMetadata(key string) ([]byte, error) {
	var value []byte
	
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(metaBucket))
		data := bucket.Get([]byte(key))
		if data != nil {
			value = make([]byte, len(data))
			copy(value, data)
		}
		return nil
	})

	return value, err
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Stats returns storage statistics
func (s *Storage) Stats() (bolt.Stats, error) {
	return s.db.Stats(), nil
}
