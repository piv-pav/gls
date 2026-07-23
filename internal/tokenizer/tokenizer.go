package tokenizer

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/piv-pav/gls/internal/util"
)

// Token represents a processed token with its position
type Token struct {
	Text     string
	Position int
}

// Tokenizer handles text tokenization and normalization
type Tokenizer struct {
	minWordLength int
	stopWords     map[string]bool
	wordPattern   *regexp.Regexp
}

// NewTokenizer creates a new tokenizer instance
func NewTokenizer() *Tokenizer {
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "has": true, "he": true,
		"in": true, "is": true, "it": true, "its": true, "of": true, "on": true,
		"that": true, "the": true, "to": true, "was": true, "will": true, "with": true,
	}

	return &Tokenizer{
		minWordLength: 2,
		stopWords:     stopWords,
		wordPattern:   regexp.MustCompile(`\w+`),
	}
}

// Tokenize splits text into tokens with positions
func (t *Tokenizer) Tokenize(text string) []Token {
	var tokens []Token
	position := 0

	// Find all words
	matches := t.wordPattern.FindAllStringIndex(text, -1)
	for _, match := range matches {
		word := text[match[0]:match[1]]
		normalized := t.normalize(word)
		
		if normalized != "" && !t.stopWords[normalized] {
			tokens = append(tokens, Token{
				Text:     normalized,
				Position: position,
			})
			position++
		}
	}

	return tokens
}

// TokenizeToStrings returns just the token strings without positions
func (t *Tokenizer) TokenizeToStrings(text string) []string {
	tokens := t.Tokenize(text)
	result := make([]string, len(tokens))
	for i, token := range tokens {
		result[i] = token.Text
	}
	return result
}

// normalize converts a word to lowercase and applies basic stemming
func (t *Tokenizer) normalize(word string) string {
	// Convert to lowercase
	word = strings.ToLower(word)

	// Remove non-alphanumeric characters except underscores
	var builder strings.Builder
	for _, r := range word {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			builder.WriteRune(r)
		}
	}
	word = builder.String()

	// Check minimum length
	if len(word) < t.minWordLength {
		return ""
	}

	// Simple stemming - remove common suffixes
	word = t.simpleStem(word)

	return word
}

// simpleStem applies basic stemming rules
func (t *Tokenizer) simpleStem(word string) string {
	suffixes := []string{"ing", "ed", "ly", "es", "s"}
	
	for _, suffix := range suffixes {
		if len(word) > len(suffix)+2 && strings.HasSuffix(word, suffix) {
			return word[:len(word)-len(suffix)]
		}
	}
	
	return word
}

// CalculateSimilarity calculates Levenshtein distance between two strings
func CalculateSimilarity(s1, s2 string) int {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	if s1 == s2 {
		return 0
	}

	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = util.MinInt3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}
