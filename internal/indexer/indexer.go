package indexer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"codeberg.org/pivpav/gls/internal/index"
	"codeberg.org/pivpav/gls/internal/tokenizer"
)

// Indexer handles file indexing operations
type Indexer struct {
	index     *index.InvertedIndex
	tokenizer *tokenizer.Tokenizer
	mu        sync.Mutex
	fileCache map[string]FileInfo // path -> file info for incremental indexing
}

// FileInfo stores metadata about indexed files
type FileInfo struct {
	Path    string
	ModTime int64
	Hash    string
}

// NewIndexer creates a new indexer instance
func NewIndexer(idx *index.InvertedIndex) *Indexer {
	return &Indexer{
		index:     idx,
		tokenizer: tokenizer.NewTokenizer(),
		fileCache: make(map[string]FileInfo),
	}
}

// IndexDirectory recursively indexes all files in a directory
func (i *Indexer) IndexDirectory(rootPath string) (int, error) {
	indexed := 0
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() && strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file should be indexed
		if !i.shouldIndex(path) {
			return nil
		}

		// Check if file needs reindexing (incremental indexing)
		if !i.needsReindexing(path, info) {
			return nil
		}

		// Index the file
		if err := i.IndexFile(path); err != nil {
			// Log error but continue indexing other files
			fmt.Printf("Warning: failed to index %s: %v\n", path, err)
			return nil
		}

		indexed++
		return nil
	})

	return indexed, err
}

// IndexFile indexes a single file
func (i *Indexer) IndexFile(path string) error {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Calculate file hash
	hash := i.calculateHash(content)

	// Create document ID from path
	docID := i.generateDocID(path)

	// Tokenize content
	tokens := i.tokenizer.Tokenize(string(content))
	tokenStrings := make([]string, len(tokens))
	positions := make([]int, len(tokens))
	for idx, token := range tokens {
		tokenStrings[idx] = token.Text
		positions[idx] = token.Position
	}

	// Create document
	doc := &index.Document{
		ID:      docID,
		Path:    path,
		Content: string(content),
		ModTime: info.ModTime().Unix(),
		Size:    info.Size(),
	}

	// Add to index
	i.index.AddDocument(doc, tokenStrings, positions)

	// Update file cache
	i.mu.Lock()
	i.fileCache[path] = FileInfo{
		Path:    path,
		ModTime: info.ModTime().Unix(),
		Hash:    hash,
	}
	i.mu.Unlock()

	return nil
}

// RemoveFile removes a file from the index
func (i *Indexer) RemoveFile(path string) {
	docID := i.generateDocID(path)
	i.index.RemoveDocument(docID)
	
	i.mu.Lock()
	delete(i.fileCache, path)
	i.mu.Unlock()
}

// shouldIndex determines if a file should be indexed based on its extension
func (i *Indexer) shouldIndex(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Supported extensions
	supportedExts := map[string]bool{
		".md":   true,
		".txt":  true,
		".go":   true,
		".py":   true,
		".js":   true,
		".ts":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".rs":   true,
		".rb":   true,
		".php":  true,
		".sh":   true,
		".yml":  true,
		".yaml": true,
		".json": true,
		".xml":  true,
		".html": true,
		".css":  true,
		".sql":  true,
		"":      true, // Files without extension (like README)
	}

	return supportedExts[ext]
}

// needsReindexing checks if a file needs to be reindexed
func (i *Indexer) needsReindexing(path string, info os.FileInfo) bool {
	i.mu.Lock()
	cached, exists := i.fileCache[path]
	i.mu.Unlock()

	if !exists {
		return true
	}

	// Check if modification time changed
	if cached.ModTime != info.ModTime().Unix() {
		return true
	}

	return false
}

// calculateHash computes SHA-256 hash of content
func (i *Indexer) calculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// generateDocID generates a unique document ID from path
func (i *Indexer) generateDocID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", hash)
}

// GetFileCache returns the current file cache
func (i *Indexer) GetFileCache() map[string]FileInfo {
	i.mu.Lock()
	defer i.mu.Unlock()

	cache := make(map[string]FileInfo)
	for k, v := range i.fileCache {
		cache[k] = v
	}
	return cache
}

// SetFileCache sets the file cache (for loading from storage)
func (i *Indexer) SetFileCache(cache map[string]FileInfo) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.fileCache = cache
}

// Stats returns indexer statistics
type Stats struct {
	FilesIndexed int
	TotalSize    int64
}

// GetStats returns indexer statistics
func (i *Indexer) GetStats() Stats {
	i.mu.Lock()
	defer i.mu.Unlock()

	stats := Stats{
		FilesIndexed: len(i.fileCache),
	}

	for _, info := range i.fileCache {
		file, err := os.Stat(info.Path)
		if err == nil {
			stats.TotalSize += file.Size()
		}
	}

	return stats
}
