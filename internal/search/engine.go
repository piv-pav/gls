package search

import (
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/piv-pav/gls/internal/index"
	"github.com/piv-pav/gls/internal/indexer"
	"github.com/piv-pav/gls/internal/storage"
	"github.com/piv-pav/gls/internal/tokenizer"
)

// Engine is the main search engine
type Engine struct {
	index     *index.InvertedIndex
	indexer   *indexer.Indexer
	storage   *storage.Storage
	tokenizer *tokenizer.Tokenizer
}

// NewEngine creates a new search engine
func NewEngine(storagePath string) (*Engine, error) {
	// Initialize storage
	store, err := storage.NewStorage(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Load or create index
	idx, err := store.LoadIndex()
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	// Create indexer
	idxr := indexer.NewIndexer(idx)

	// Load file cache if available
	cacheData, err := store.LoadMetadata("file_cache")
	if err == nil && cacheData != nil {
		cache := make(map[string]indexer.FileInfo)
		if err := gob.NewDecoder(strings.NewReader(string(cacheData))).Decode(&cache); err == nil {
			idxr.SetFileCache(cache)
		}
	}

	return &Engine{
		index:     idx,
		indexer:   idxr,
		storage:   store,
		tokenizer: tokenizer.NewTokenizer(),
	}, nil
}

// Index indexes a directory
func (e *Engine) Index(path string) (int, error) {
	count, err := e.indexer.IndexDirectory(path)
	if err != nil {
		return count, err
	}

	// Save index to disk
	if err := e.Save(); err != nil {
		return count, fmt.Errorf("failed to save index: %w", err)
	}

	return count, nil
}

// IndexFile indexes a single file
func (e *Engine) IndexFile(path string) error {
	if err := e.indexer.IndexFile(path); err != nil {
		return err
	}

	return e.Save()
}

// Search performs a search query
func (e *Engine) Search(query string, fuzzy bool, maxDistance int) []index.SearchResult {
	// Tokenize query
	queryTerms := e.tokenizer.TokenizeToStrings(query)

	if len(queryTerms) == 0 {
		return []index.SearchResult{}
	}

	// Perform search
	var results []index.SearchResult
	if fuzzy {
		results = e.index.FuzzySearch(queryTerms, maxDistance)
	} else {
		results = e.index.Search(queryTerms)
	}

	return results
}

// GetStats returns engine statistics
func (e *Engine) GetStats() Stats {
	indexStats := e.index.GetStats()
	indexerStats := e.indexer.GetStats()

	return Stats{
		DocumentCount: indexStats.DocumentCount,
		TermCount:     indexStats.TermCount,
		FilesIndexed:  indexerStats.FilesIndexed,
		TotalSize:     indexerStats.TotalSize,
	}
}

// Save saves the index and metadata to disk
func (e *Engine) Save() error {
	// Save index
	if err := e.storage.SaveIndex(e.index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	// Save file cache
	cache := e.indexer.GetFileCache()
	var buf strings.Builder
	if err := gob.NewEncoder(&buf).Encode(cache); err != nil {
		return fmt.Errorf("failed to encode cache: %w", err)
	}
	if err := e.storage.SaveMetadata("file_cache", []byte(buf.String())); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// Close closes the engine and releases resources
func (e *Engine) Close() error {
	return e.storage.Close()
}

// Stats contains search engine statistics
type Stats struct {
	DocumentCount int
	TermCount     int
	FilesIndexed  int
	TotalSize     int64
}
