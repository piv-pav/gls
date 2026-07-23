package index

import (
	"encoding/gob"
	"io"
	"math"
	"sort"
	"sync"

	"github.com/piv-pav/gls/internal/util"
)

// Document represents an indexed document
type Document struct {
	ID       string
	Path     string
	Content  string
	ModTime  int64
	Size     int64
}

// Posting represents a document that contains a term
type Posting struct {
	DocID     string
	Frequency int
	Positions []int
}

// InvertedIndex stores the inverted index structure
type InvertedIndex struct {
	mu        sync.RWMutex
	index     map[string][]Posting  // term -> postings list
	documents map[string]*Document  // docID -> document
	docCount  int
}

// NewInvertedIndex creates a new inverted index
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		index:     make(map[string][]Posting),
		documents: make(map[string]*Document),
		docCount:  0,
	}
}

// AddDocument adds or updates a document in the index
func (idx *InvertedIndex) AddDocument(doc *Document, tokens []string, positions []int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Remove old document if exists
	if oldDoc, exists := idx.documents[doc.ID]; exists {
		idx.removeDocumentLocked(oldDoc)
	}

	// Add new document
	idx.documents[doc.ID] = doc
	idx.docCount++

	// Build term frequency map
	termFreq := make(map[string]int)
	termPositions := make(map[string][]int)
	
	for i, token := range tokens {
		termFreq[token]++
		termPositions[token] = append(termPositions[token], positions[i])
	}

	// Update inverted index
	for term, freq := range termFreq {
		posting := Posting{
			DocID:     doc.ID,
			Frequency: freq,
			Positions: termPositions[term],
		}
		idx.index[term] = append(idx.index[term], posting)
	}
}

// RemoveDocument removes a document from the index
func (idx *InvertedIndex) RemoveDocument(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if doc, exists := idx.documents[docID]; exists {
		idx.removeDocumentLocked(doc)
	}
}

// removeDocumentLocked removes a document (caller must hold lock)
func (idx *InvertedIndex) removeDocumentLocked(doc *Document) {
	delete(idx.documents, doc.ID)
	idx.docCount--

	// Remove postings from inverted index
	for term, postings := range idx.index {
		newPostings := make([]Posting, 0)
		for _, posting := range postings {
			if posting.DocID != doc.ID {
				newPostings = append(newPostings, posting)
			}
		}
		if len(newPostings) > 0 {
			idx.index[term] = newPostings
		} else {
			delete(idx.index, term)
		}
	}
}

// Search performs a search query and returns ranked results
func (idx *InvertedIndex) Search(queryTerms []string) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Find documents containing query terms
	docScores := make(map[string]float64)
	docMatches := make(map[string]int)

	for _, term := range queryTerms {
		if postings, exists := idx.index[term]; exists {
			idf := idx.calculateIDF(len(postings))
			
			for _, posting := range postings {
				tf := float64(posting.Frequency)
				score := tf * idf
				docScores[posting.DocID] += score
				docMatches[posting.DocID]++
			}
		}
	}

	// Convert to results and sort by score
	results := make([]SearchResult, 0, len(docScores))
	for docID, score := range docScores {
		if doc, exists := idx.documents[docID]; exists {
			results = append(results, SearchResult{
				Document:    doc,
				Score:       score,
				MatchCount:  docMatches[docID],
				QueryTerms:  len(queryTerms),
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		// Sort by match count first, then by score
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		return results[i].Score > results[j].Score
	})

	return results
}

// FuzzySearch performs fuzzy matching search
func (idx *InvertedIndex) FuzzySearch(queryTerms []string, maxDistance int) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Find similar terms using Levenshtein distance
	expandedTerms := make(map[string]bool)
	for _, queryTerm := range queryTerms {
		expandedTerms[queryTerm] = true
		
		for indexTerm := range idx.index {
			distance := levenshteinDistance(queryTerm, indexTerm)
			if distance <= maxDistance {
				expandedTerms[indexTerm] = true
			}
		}
	}

	// Convert to slice and search
	terms := make([]string, 0, len(expandedTerms))
	for term := range expandedTerms {
		terms = append(terms, term)
	}

	return idx.Search(terms)
}

// GetDocument retrieves a document by ID
func (idx *InvertedIndex) GetDocument(docID string) (*Document, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	
	doc, exists := idx.documents[docID]
	return doc, exists
}

// GetStats returns index statistics
func (idx *InvertedIndex) GetStats() Stats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return Stats{
		DocumentCount: idx.docCount,
		TermCount:     len(idx.index),
	}
}

// calculateIDF calculates inverse document frequency
func (idx *InvertedIndex) calculateIDF(docFreq int) float64 {
	if idx.docCount == 0 || docFreq == 0 {
		return 0
	}
	return math.Log(float64(idx.docCount) / float64(docFreq))
}

// SearchResult represents a search result with scoring
type SearchResult struct {
	Document   *Document
	Score      float64
	MatchCount int
	QueryTerms int
}

// Stats contains index statistics
type Stats struct {
	DocumentCount int
	TermCount     int
}

// Serialize encodes the index for storage
func (idx *InvertedIndex) Serialize() ([]byte, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var buf []byte
	enc := gob.NewEncoder(&gobWriter{data: &buf})
	
	if err := enc.Encode(idx.docCount); err != nil {
		return nil, err
	}
	if err := enc.Encode(idx.documents); err != nil {
		return nil, err
	}
	if err := enc.Encode(idx.index); err != nil {
		return nil, err
	}

	return buf, nil
}

// Deserialize decodes the index from storage
func (idx *InvertedIndex) Deserialize(data []byte) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	dec := gob.NewDecoder(&gobReader{data: data})
	
	if err := dec.Decode(&idx.docCount); err != nil {
		return err
	}
	if err := dec.Decode(&idx.documents); err != nil {
		return err
	}
	if err := dec.Decode(&idx.index); err != nil {
		return err
	}

	return nil
}

// gobWriter implements io.Writer for gob encoding
type gobWriter struct {
	data *[]byte
}

func (w *gobWriter) Write(p []byte) (n int, err error) {
	*w.data = append(*w.data, p...)
	return len(p), nil
}

// gobReader implements io.Reader for gob decoding
type gobReader struct {
	data []byte
	pos  int
}

func (r *gobReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Use only two rows for space optimization
	prev := make([]int, len(s2)+1)
	curr := make([]int, len(s2)+1)

	for j := 0; j <= len(s2); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(s1); i++ {
		curr[0] = i
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			curr[j] = util.MinInt3(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(s2)]
}
