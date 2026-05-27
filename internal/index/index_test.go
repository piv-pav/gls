package index

import (
	"testing"
)

func TestInvertedIndex(t *testing.T) {
	idx := NewInvertedIndex()
	
	// Add a document
	doc := &Document{
		ID:      "doc1",
		Path:    "/test/file.txt",
		Content: "hello world golang programming",
		ModTime: 0,
		Size:    100,
	}
	
	tokens := []string{"hello", "world", "golang", "programming"}
	positions := []int{0, 1, 2, 3}
	
	idx.AddDocument(doc, tokens, positions)
	
	// Test retrieval
	retrieved, exists := idx.GetDocument("doc1")
	if !exists {
		t.Fatal("document should exist")
	}
	if retrieved.ID != doc.ID {
		t.Errorf("expected ID %s, got %s", doc.ID, retrieved.ID)
	}
	
	// Test search
	results := idx.Search([]string{"golang"})
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Document.ID != "doc1" {
		t.Errorf("expected doc1, got %s", results[0].Document.ID)
	}
	
	// Test stats
	stats := idx.GetStats()
	if stats.DocumentCount != 1 {
		t.Errorf("expected 1 document, got %d", stats.DocumentCount)
	}
	if stats.TermCount != 4 {
		t.Errorf("expected 4 terms, got %d", stats.TermCount)
	}
}

func TestMultipleDocuments(t *testing.T) {
	idx := NewInvertedIndex()
	
	doc1 := &Document{ID: "doc1", Path: "/test/file1.txt", Content: "golang programming"}
	doc2 := &Document{ID: "doc2", Path: "/test/file2.txt", Content: "python programming"}
	
	idx.AddDocument(doc1, []string{"golang", "programming"}, []int{0, 1})
	idx.AddDocument(doc2, []string{"python", "programming"}, []int{0, 1})
	
	// Search for common term
	results := idx.Search([]string{"programming"})
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	
	// Search for specific term
	results = idx.Search([]string{"golang"})
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Document.ID != "doc1" {
		t.Errorf("expected doc1, got %s", results[0].Document.ID)
	}
}

func TestRemoveDocument(t *testing.T) {
	idx := NewInvertedIndex()
	
	doc := &Document{ID: "doc1", Path: "/test/file.txt", Content: "test content"}
	idx.AddDocument(doc, []string{"test", "content"}, []int{0, 1})
	
	// Verify document exists
	if _, exists := idx.GetDocument("doc1"); !exists {
		t.Fatal("document should exist")
	}
	
	// Remove document
	idx.RemoveDocument("doc1")
	
	// Verify document is gone
	if _, exists := idx.GetDocument("doc1"); exists {
		t.Fatal("document should not exist")
	}
	
	// Verify stats updated
	stats := idx.GetStats()
	if stats.DocumentCount != 0 {
		t.Errorf("expected 0 documents, got %d", stats.DocumentCount)
	}
}

func TestFuzzySearch(t *testing.T) {
	idx := NewInvertedIndex()
	
	doc := &Document{ID: "doc1", Path: "/test/file.txt", Content: "python programming"}
	idx.AddDocument(doc, []string{"python", "programming"}, []int{0, 1})
	
	// Fuzzy search with typo
	results := idx.FuzzySearch([]string{"pythn"}, 2)
	if len(results) == 0 {
		t.Error("fuzzy search should find results")
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"hello", "hello", 0},
		{"hello", "hallo", 1},
		{"python", "pythn", 1},
		{"golang", "java", 5},
	}
	
	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			result := levenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("expected distance %d, got %d", tt.expected, result)
			}
		})
	}
}
