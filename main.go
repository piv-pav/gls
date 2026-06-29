package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"codeberg.org/pivpav/gls/internal/search"
	"codeberg.org/pivpav/gls/pkg/config"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

var (
	// Version is set by ldflags during build
	Version = "dev"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load configuration
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "gls", "config.json")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("%sError loading config: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Ensure storage directory exists
	if err := cfg.EnsureStorageDir(); err != nil {
		fmt.Printf("%sError creating storage directory: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Parse: gls [index_name] <command> <args...>
	// or: gls <query> (default search with default index)
	indexName := ""
	command := ""
	argOffset := 1

	// Check if first arg is a known command
	knownCommands := map[string]bool{
		"index": true, "search": true, "stats": true,
		"help": true, "--help": true, "-h": true, "list": true, "delete": true,
		"version": true, "--version": true, "-v": true,
	}

	firstArg := os.Args[1]
	
	if knownCommands[firstArg] {
		// gls <command> <args...>
		command = firstArg
		argOffset = 2
	} else if len(os.Args) >= 3 && knownCommands[os.Args[2]] {
		// gls <index_name> <command> <args...>
		indexName = firstArg
		command = os.Args[2]
		argOffset = 3
	} else if len(os.Args) >= 2 {
		// Check if first arg looks like an index name (exists in config)
		// or if it starts with a quote (likely a query)
		if strings.HasPrefix(firstArg, `"`) || strings.HasPrefix(firstArg, `'`) {
			// gls "query" = gls search "query"
			command = "search"
			argOffset = 1
		} else {
			// Check if it's a known index name
			_, indexExists := cfg.Indexes[firstArg]
			if indexExists && len(os.Args) >= 3 {
				// gls <index_name> <query>
				indexName = firstArg
				command = "search"
				argOffset = 2
			} else {
				// Default: gls <query>
				command = "search"
				argOffset = 1
			}
		}
	} else {
		// Default: gls <query> = gls search <query>
		command = "search"
		argOffset = 1
	}

	switch command {
	case "index":
		handleIndex(cfg, indexName, argOffset)
	case "search":
		handleSearch(cfg, indexName, argOffset)
	case "stats":
		handleStats(cfg, indexName)
	case "list":
		handleList(cfg)
	case "delete":
		handleDelete(cfg, indexName, argOffset)
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Printf("%sUnknown command: %s%s\n", colorRed, command, colorReset)
		printUsage()
		os.Exit(1)
	}
}

func handleIndex(cfg *config.Config, indexName string, argOffset int) {
	if indexName == "" {
		indexName = "default"
	}

	// Check if path provided
	var paths []string
	if len(os.Args) < argOffset+1 {
		// No path provided - try to re-index existing paths from config
		paths = cfg.GetIndexPaths(indexName)
		if len(paths) == 0 {
			fmt.Printf("%sUsage: gls [index_name] index <path>%s\n", colorRed, colorReset)
			fmt.Printf("%sNo existing paths for index '%s'. Please provide a path.%s\n", colorRed, indexName, colorReset)
			os.Exit(1)
		}
		fmt.Printf("%sRe-indexing existing paths for index '%s'...%s\n", colorCyan, indexName, colorReset)
	} else {
		// Path provided - validate and use it
		path := os.Args[argOffset]

		// Validate path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("%sError: path does not exist: %s%s\n", colorRed, path, colorReset)
			os.Exit(1)
		}

		// Get absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("%sError resolving path: %v%s\n", colorRed, err, colorReset)
			os.Exit(1)
		}

		paths = []string{absPath}
	}

	dbPath := cfg.GetIndexPath(indexName)
	engine, err := search.NewEngine(dbPath)
	if err != nil {
		fmt.Printf("%sError initializing engine: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	defer engine.Close()

	totalCount := 0
	for _, path := range paths {
		fmt.Printf("%sIndexing: %s%s\n", colorCyan, path, colorReset)
		count, err := engine.Index(path)
		if err != nil {
			fmt.Printf("%sError indexing %s: %v%s\n", colorRed, path, err, colorReset)
			os.Exit(1)
		}
		totalCount += count
	}

	// Save paths to config (only if new path was provided)
	if len(os.Args) >= argOffset+1 {
		for _, path := range paths {
			cfg.AddIndexPath(indexName, path)
		}
		homeDir, _ := os.UserHomeDir()
		configPath := filepath.Join(homeDir, ".config", "gls", "config.json")
		if err := cfg.SaveConfig(configPath); err != nil {
			fmt.Printf("%sWarning: failed to save config: %v%s\n", colorYellow, err, colorReset)
		}
	}

	fmt.Printf("%s✓ Indexed %d files%s\n", colorGreen, totalCount, colorReset)
}

func handleSearch(cfg *config.Config, indexName string, argOffset int) {
	if len(os.Args) < argOffset+1 {
		fmt.Printf("%sUsage: gls [index_name] search <query> [--fuzzy] [--distance N] [--limit N | -l N]%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	// Parse arguments
	query := os.Args[argOffset]
	fuzzy := cfg.FuzzySearch
	maxDistance := cfg.MaxDistance
	limit := 10 // Default limit for AI use

	for i := argOffset + 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--fuzzy":
			fuzzy = true
		case "--distance":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &maxDistance)
				i++
			}
		case "--limit", "-l":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &limit)
				i++
			}
		}
	}

	if indexName == "" {
		indexName = "default"
	}

	dbPath := cfg.GetIndexPath(indexName)
	
	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("%sError: index '%s' does not exist. Create it with: gls %s index <path>%s\n", 
			colorRed, indexName, indexName, colorReset)
		os.Exit(1)
	}

	engine, err := search.NewEngine(dbPath)
	if err != nil {
		fmt.Printf("%sError initializing engine: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	defer engine.Close()

	results := engine.Search(query, fuzzy, maxDistance)

	if len(results) == 0 {
		fmt.Printf("%sNo results found for: %s (index: %s)%s\n", colorYellow, query, indexName, colorReset)
		return
	}

	totalResults := len(results)
	if limit > 0 && limit < totalResults {
		results = results[:limit]
	}

	fmt.Printf("%sFound %d results for: %s (index: %s)%s", colorGreen, totalResults, query, indexName, colorReset)
	if limit > 0 && limit < totalResults {
		fmt.Printf(" %s(showing %d)%s", colorYellow, limit, colorReset)
	}
	fmt.Printf("\n\n")

	for i, result := range results {
		fmt.Printf("%s%d. %s%s\n", colorBlue, i+1, result.Document.Path, colorReset)
		lineNum, snippet := getSnippetWithLine(result.Document.Content, query, 300)
		snippet = highlightMatches(snippet, query)
		fmt.Printf("   Score: %.2f | Matches: %d/%d | Line: %d\n", result.Score, result.MatchCount, result.QueryTerms, lineNum)
		fmt.Printf("   %s\n\n", snippet)
	}
}

func handleStats(cfg *config.Config, indexName string) {
	if indexName == "" {
		indexName = "default"
	}

	dbPath := cfg.GetIndexPath(indexName)
	
	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("%sError: index '%s' does not exist%s\n", colorRed, indexName, colorReset)
		os.Exit(1)
	}

	engine, err := search.NewEngine(dbPath)
	if err != nil {
		fmt.Printf("%sError initializing engine: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	defer engine.Close()

	stats := engine.GetStats()

	fmt.Printf("%s=== Index Statistics (index: %s) ===%s\n", colorCyan, indexName, colorReset)
	fmt.Printf("Documents:    %d\n", stats.DocumentCount)
	fmt.Printf("Terms:        %d\n", stats.TermCount)
	fmt.Printf("Files:        %d\n", stats.FilesIndexed)
	fmt.Printf("Total Size:   %.2f MB\n", float64(stats.TotalSize)/(1024*1024))
}

func handleList(cfg *config.Config) {
	fmt.Printf("%s=== Available Indexes ===%s\n\n", colorCyan, colorReset)
	
	names := cfg.ListIndexNames()
	if len(names) == 0 {
		fmt.Printf("%sNo indexes found. Create one with: gls index <path>%s\n", colorYellow, colorReset)
		return
	}

	for _, name := range names {
		paths := cfg.GetIndexPaths(name)
		dbPath := cfg.GetIndexPath(name)
		
		fmt.Printf("%s%s%s\n", colorGreen, name, colorReset)
		
		// Check if DB exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Printf("  %sStatus: Not indexed%s\n", colorYellow, colorReset)
		} else {
			engine, err := search.NewEngine(dbPath)
			if err == nil {
				stats := engine.GetStats()
				engine.Close()
				fmt.Printf("  Docs: %d | Terms: %d | Size: %.2f MB\n", 
					stats.DocumentCount, stats.TermCount, float64(stats.TotalSize)/(1024*1024))
			}
		}
		
		if len(paths) > 0 {
			fmt.Printf("  Paths:\n")
			for _, p := range paths {
				fmt.Printf("    - %s\n", p)
			}
		}
		fmt.Println()
	}
}

func handleDelete(cfg *config.Config, indexName string, argOffset int) {
	// If indexName not set by parser, read from args
	if indexName == "" {
		if len(os.Args) < argOffset+1 {
			fmt.Printf("%sError: index name required%s\n", colorRed, colorReset)
			fmt.Printf("Usage: gls delete <index_name>\n")
			os.Exit(1)
		}
		indexName = os.Args[argOffset]
	}

	// Check if index exists
	if _, exists := cfg.Indexes[indexName]; !exists {
		fmt.Printf("%sError: index '%s' not found%s\n", colorRed, indexName, colorReset)
		fmt.Printf("Use 'gls list' to see available indexes\n")
		os.Exit(1)
	}

	// Get DB path
	dbPath := cfg.GetIndexPath(indexName)

	// Delete DB file if it exists
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			fmt.Printf("%sError deleting index file: %v%s\n", colorRed, err, colorReset)
			os.Exit(1)
		}
		fmt.Printf("%sDeleted index file: %s%s\n", colorGreen, dbPath, colorReset)
	}

	// Remove from config
	cfg.DeleteIndex(indexName)

	// Save config
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "gls", "config.json")
	if err := cfg.SaveConfig(configPath); err != nil {
		fmt.Printf("%sError saving config: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%sSuccessfully deleted index '%s'%s\n", colorGreen, indexName, colorReset)
}

func printVersion() {
	version := Version
	
	// Try to get version from build info (when installed via go install)
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
		}
	}
	
	fmt.Printf("%sgls version %s%s\n", colorCyan, version, colorReset)
}

func printUsage() {
	fmt.Printf("%sGo Local Search - A fast local file search engine%s\n\n", colorCyan, colorReset)
	fmt.Printf("Usage:\n")
	fmt.Printf("  %sgls [index_name] index <path>%s    Index files in directory\n", colorGreen, colorReset)
	fmt.Printf("  %sgls [index_name] search <query>%s  Search indexed files\n", colorGreen, colorReset)
	fmt.Printf("  %sgls [index_name] <query>%s         Search (shorthand)\n", colorGreen, colorReset)
	fmt.Printf("    %s--fuzzy%s                         Enable fuzzy matching\n", colorYellow, colorReset)
	fmt.Printf("    %s--distance N%s                    Set max edit distance (default: 2)\n", colorYellow, colorReset)
	fmt.Printf("    %s--limit N, -l N%s                 Limit results (default: 10)\n", colorYellow, colorReset)
	fmt.Printf("  %sgls [index_name] stats%s           Show index statistics\n", colorGreen, colorReset)
	fmt.Printf("  %sgls delete <index_name>%s          Delete an index\n", colorGreen, colorReset)
	fmt.Printf("  %sgls list%s                          List all indexes\n", colorGreen, colorReset)
	fmt.Printf("  %sgls help%s                          Show this help message\n", colorGreen, colorReset)
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  gls index ~/Documents              # Index to 'default'\n")
	fmt.Printf("  gls work index ~/Work              # Index to 'work'\n")
	fmt.Printf("  gls \"golang tutorial\"              # Search in 'default'\n")
	fmt.Printf("  gls work search \"function\"         # Search in 'work'\n")
	fmt.Printf("  gls work \"function\" --fuzzy        # Fuzzy search in 'work'\n")
	fmt.Printf("  gls delete work                    # Delete 'work' index\n")
	fmt.Printf("  gls list                           # List all indexes\n")
}

// getSnippetWithLine finds the first query term match, returns its 1-based line
// number and a snippet extracted from that line (trimmed to maxLen).
func getSnippetWithLine(content, query string, maxLen int) (int, string) {
	contentLower := strings.ToLower(content)
	words := strings.Fields(strings.ToLower(query))

	firstMatch := -1
	for _, word := range words {
		if idx := strings.Index(contentLower, word); idx >= 0 {
			if firstMatch < 0 || idx < firstMatch {
				firstMatch = idx
			}
		}
	}

	if firstMatch < 0 {
		// No match found — fall back to start of doc
		return 1, getSnippet(content, maxLen)
	}

	lineNum := strings.Count(content[:firstMatch], "\n") + 1

	// Find start and end of the matched line
	lineStart := strings.LastIndex(content[:firstMatch], "\n") + 1
	lineEnd := strings.Index(content[firstMatch:], "\n")
	var line string
	if lineEnd < 0 {
		line = content[lineStart:]
	} else {
		line = content[lineStart : firstMatch+lineEnd]
	}

	// Trim leading/trailing whitespace, preserve internal spacing
	line = strings.TrimSpace(line)
	if len(line) > maxLen {
		line = line[:maxLen] + "..."
	}

	return lineNum, line
}

// findLineNumber is kept for potential future use.
func findLineNumber(content, query string) int {
	lineNum, _ := getSnippetWithLine(content, query, 0)
	return lineNum
}

func getSnippet(content string, maxLen int) string {
	// Replace newlines and multiple spaces
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")
	content = strings.Join(strings.Fields(content), " ")

	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

func highlightMatches(text, query string) string {
	words := strings.Fields(strings.ToLower(query))
	result := text

	for _, word := range words {
		// Case-insensitive replacement
		idx := strings.Index(strings.ToLower(result), word)
		if idx >= 0 {
			original := result[idx : idx+len(word)]
			highlighted := colorYellow + original + colorReset
			result = result[:idx] + highlighted + result[idx+len(word):]
		}
	}

	return result
}
