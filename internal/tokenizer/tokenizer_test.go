package tokenizer

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tokenizer := NewTokenizer()
	
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple text",
			input:    "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with stop words",
			input:    "The quick brown fox",
			expected: []string{"quick", "brown", "fox"},
		},
		{
			name:     "code example",
			input:    "func main() { fmt.Println() }",
			expected: []string{"func", "main", "fmt", "println"},
		},
		{
			name:     "with numbers",
			input:    "Go version 1.20 released",
			expected: []string{"go", "version", "20", "releas"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.TokenizeToStrings(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(result))
				return
			}
			for i, token := range result {
				if token != tt.expected[i] {
					t.Errorf("token %d: expected %s, got %s", i, tt.expected[i], token)
				}
			}
		})
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"hello", "hello", 0},
		{"hello", "hallo", 1},
		{"python", "pythn", 1},
		{"golang", "java", 5},
		{"test", "", 4},
	}
	
	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			result := CalculateSimilarity(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("expected distance %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tokenizer := NewTokenizer()
	
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "hello"},
		{"WORLD", "world"},
		{"testing", "test"},
		{"Running", "runn"},
		{"Go", "go"},  // Short words are kept
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := tokenizer.normalize(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
