# Go Local Search (gls)

A fast, lightweight local full-text search engine for indexing and searching through your files (Markdown, text, and code). Built entirely in Go with inverted index data structures and persistent storage using BoltDB.

**Forked from**: [BaseMax/go-local-search](https://github.com/BaseMax/go-local-search)

**Fork changes**: 
- Renamed binary from `search` to `gls`
- Added support for multiple named indexes
- Removed HTTP server (CLI-only now)
- XDG-compliant storage paths
- Default limit of 10 search results (configurable via `--limit`)
- Added `justfile` for task automation

## Features

- 🚀 **Fast Indexing**: Efficiently indexes files using inverted index data structures
- 🔍 **Instant Search**: Sub-second search across thousands of files
- 🎯 **TF-IDF Ranking**: Intelligent relevance scoring using Term Frequency-Inverse Document Frequency
- 🔤 **Fuzzy Matching**: Find results even with typos using Levenshtein distance
- 📊 **Incremental Indexing**: Only re-indexes modified files
- 💾 **Persistent Storage**: Indexes stored on disk using BoltDB
- 🖥️ **CLI Interface**: Fast command-line tool
- 📝 **Multiple File Types**: Supports Markdown, text, and code files (.md, .txt, .go, .py, .js, .ts, .java, .c, .cpp, .rs, etc.)
- 🎨 **Colored CLI Output**: Beautiful, colorized terminal output
- 📚 **Multiple Named Indexes**: Organize indexes by project or category
- 📁 **XDG-Compliant Storage**: Config in `~/.config/gls/`, indexes in `~/.cache/gls/`

## Installation

### Prerequisites
- Go 1.18 or higher

### Build from Source

```bash
git clone https://github.com/piv-pav/gls.git
cd gls
make build  # Creates bin/gls
```

### Install Globally

```bash
go install github.com/piv-pav/gls@latest
```

## Usage

### CLI Commands

#### Index Files

Index to the default index:
```bash
gls index /path/to/directory
```

Create named indexes for different projects:
```bash
gls work index ~/Work
gls docs index ~/Documents
gls notes index ~/Notes
```

#### Search

Search in the default index:
```bash
gls search "your query"
gls "your query"  # shorthand
```

Search in a named index:
```bash
gls work search "function"
gls work "function"  # shorthand
```

Fuzzy search (tolerates typos):
```bash
gls "pythn" --fuzzy
gls work "handleRequest" --fuzzy --distance 2
```

#### List Indexes

Show all available indexes:
```bash
gls list
```

#### Delete Index

Remove an index (deletes config entry and .db file):
```bash
gls delete work         # Delete 'work' index
gls delete <name>       # Delete any named index
```

#### Statistics

View index statistics:
```bash
gls stats           # default index
gls work stats      # named index
```

Limit search results:
```bash
gls search "query" --limit 20   # Show max 20 results
gls "query" -l 5               # Show max 5 results
```
Default limit: 10 results

Re-index without specifying paths (uses stored config):
```bash
gls work index                  # Re-indexes all paths configured for 'work'
```

#### View Statistics

Show index statistics:
```bash
gls stats           # default index
gls work stats      # named index
```

Output:
```
=== Index Statistics (index: work) ===
Documents:    1234
Terms:        5678
Files:        1234
Total Size:   45.67 MB
```

## Architecture

### Components

1. **Tokenizer** (`internal/tokenizer`): 
   - Text tokenization and normalization
   - Stop word filtering
   - Basic stemming
   - Levenshtein distance calculation for fuzzy matching

2. **Inverted Index** (`internal/index`):
   - Efficient inverted index data structure
   - TF-IDF scoring for relevance ranking
   - Positional information tracking
   - Thread-safe operations

3. **Storage** (`internal/storage`):
   - BoltDB integration for persistent storage
   - Index serialization/deserialization
   - Metadata storage

4. **Indexer** (`internal/indexer`):
   - Recursive directory scanning
   - File type detection
   - Incremental indexing (detects file changes)
   - SHA-256 hashing for change detection

5. **Search Engine** (`internal/search`):
   - Main search engine orchestration
   - Query processing
   - Result ranking
   - Fuzzy search implementation

### How It Works

1. **Indexing Phase**:
   - Files are scanned recursively
   - Content is tokenized into terms
   - Terms are normalized (lowercase, stemming)
   - Inverted index is built: term → list of (document, frequency, positions)
   - Index is persisted to BoltDB

2. **Search Phase**:
   - Query is tokenized and normalized
   - Relevant documents are retrieved from inverted index
   - TF-IDF scoring calculates relevance
   - Results are ranked by score and number of matching terms
   - For fuzzy search, similar terms are found using Levenshtein distance

3. **Incremental Indexing**:
   - File modification times and hashes are tracked
   - Only changed files are re-indexed
   - Removed files are automatically cleaned from index

## Technical Details

### Supported File Types

- **Markdown**: .md
- **Text**: .txt
- **Go**: .go
- **Python**: .py
- **JavaScript**: .js, .ts
- **Java**: .java
- **C/C++**: .c, .cpp, .h
- **Rust**: .rs
- **Ruby**: .rb
- **PHP**: .php
- **Shell**: .sh
- **YAML**: .yml, .yaml
- **JSON**: .json
- **XML**: .xml
- **HTML**: .html
- **CSS**: .css
- **SQL**: .sql
- **README** files (no extension)

### TF-IDF Scoring

The search engine uses TF-IDF (Term Frequency-Inverse Document Frequency) for ranking:

- **TF (Term Frequency)**: Number of times a term appears in a document
- **IDF (Inverse Document Frequency)**: log(total_documents / documents_containing_term)
- **Score**: TF × IDF

Documents with higher scores are more relevant to the query.

### Fuzzy Matching

Fuzzy search uses Levenshtein distance to find similar terms:
- Default maximum edit distance: 2
- Finds terms within the specified edit distance
- Useful for handling typos and variations

## Configuration

Configuration is stored in `~/.config/gls/config.json`:

```json
{
  "storage_dir": "~/.cache/gls",
  "indexes": {
    "default": ["/path/to/directory"],
    "work": ["/path/to/work"],
    "notes": ["/path/to/notes"]
  },
  "fuzzy_search": false,
  "max_distance": 2
}
```

Indexes are stored as `.db` files in `~/.cache/gls/`.

## Performance

- Indexing speed: ~1000 files/second (depends on file size and disk speed)
- Search speed: Sub-millisecond for most queries
- Memory usage: Efficient with lazy loading from BoltDB
- Disk usage: Index size is typically 10-20% of original file size

## Examples

### Index your projects

```bash
gls index ~/projects
gls docs index ~/Documents
gls notes index ~/notes
```

### Search examples

```bash
# Find Go tutorials
gls search "golang tutorial"

# Find function definitions (with fuzzy matching)
gls search "handleRequest" --fuzzy

# Search for algorithms with limited results
gls search "binary search algorithm" --limit 5

# Find Python code in specific index
gls docs "python class definition"
```

### Working with Indexes

```bash
# Create multiple indexes for different purposes
gls work index ~/Work
gls notes index ~/Notes
gls docs index ~/Documents

# Re-index without specifying path (uses config)
gls work index
gls notes index

# List all indexes
gls list

# View statistics
gls work stats
```

## Dependencies

- [BoltDB](https://github.com/etcd-io/bbolt) - Embedded key/value database for persistent storage

## Project Structure

```
.
├── cmd/
│   └── gls/             # Main CLI application
├── internal/
│   ├── tokenizer/       # Text tokenization and normalization
│   ├── index/           # Inverted index implementation
│   ├── storage/         # BoltDB storage layer
│   ├── indexer/         # File indexing logic
│   ├── search/          # Search engine
│   └── util/            # Utilities
├── pkg/
│   └── config/          # Configuration management
└── bin/                 # Compiled binaries

```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the GPL-3.0 License - see the LICENSE file for details.

## Author

- **Max Base** - [@BaseMax](https://github.com/BaseMax)

## Acknowledgments

- Built with Go
- Uses BoltDB for efficient storage
- Inspired by modern search engines and information retrieval techniques
