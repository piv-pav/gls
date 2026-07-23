# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.2.4] - 2026-06-29

### Changed
- Search results now always show line number of first match and snippet from matching line
- Previously snippet was taken from document start; now extracted from the matched line
- Output format: `Score: X | Matches: Y/Z | Line: N` followed by matching line content

## [0.2.3] - 2026-05-27

### Changed
- **Repository migration**: Moved to Codeberg (github.com/piv-pav/gls)
- **Module path**: Updated to `github.com/piv-pav/gls`
- **Install command**: `go install github.com/piv-pav/gls@latest`
- Updated all import paths in source code
- Updated README and CHANGELOG URLs

## [0.2.0] - 2026-05-25

### Added
- **Version command**: `gls --version` / `gls -v` / `gls version` to display version
- Version information embedded via ldflags during build

### Changed
- **Simplified install**: Moved main.go to repo root for cleaner `go install github.com/piv-pav/gls@latest`
- **Repo renamed**: `go-local-search` → `gls` on git server
- **Module path**: Updated to `github.com/piv-pav/gls`

## [0.1.0] - 2026-05-25

### Added
- **Multiple named indexes**: Index different directories separately (`gls work index ~/Work`)
- **CLI shorthand syntax**: `gls "query"` as shorthand for `gls search "query"`
- **Index management**: `gls list` to show all indexes, `gls stats` for statistics
- **Delete command**: `gls delete <index_name>` to remove indexes
- **Search result limiting**: `--limit`/`-l` flag (default: 10 results)
- **Smart re-indexing**: `gls index` without path re-indexes existing paths from config
- **XDG-compliant storage**: Indexes in `~/.cache/gls/`, config in `~/.config/gls/config.json`
- **Binary rename**: `search` → `gls` for cleaner CLI UX
- **Task automation**: Added `justfile` alongside Makefile

### Changed
- **Config structure**: `StoragePath` → `StorageDir` + `Indexes map[string][]string`
- **CLI parser**: Detects index name vs command vs query automatically
- **Storage layout**: Each index = separate `.db` file

### Removed
- **HTTP server**: Removed entire server component (CLI-only now)
- **Server endpoints**: No more `/search`, `/index`, `/stats` HTTP endpoints
- **Server configuration**: Removed server-related config options

### Fixed
- Awkward CLI syntax (`search search "query"` → `gls "query"`)
- Single global index limitation (now supports multiple named indexes)
- Non-standard config paths (now XDG-compliant)

## [1.0.0] - 2024-12-19 (Upstream)

Initial upstream release by BaseMax/go-local-search with:
- Inverted index with TF-IDF scoring
- Fuzzy matching (Levenshtein distance)
- BoltDB persistent storage
- HTTP API server
- CLI interface
- Multiple file type support

---

**Fork source**: https://github.com/BaseMax/go-local-search  
**Fork repository**: https://github.com/piv-pav/gls.git
