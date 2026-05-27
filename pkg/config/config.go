package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	StorageDir  string              `json:"storage_dir"`
	Indexes     map[string][]string `json:"indexes"` // name -> paths
	FuzzySearch bool                `json:"fuzzy_search"`
	MaxDistance int                 `json:"max_distance"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		StorageDir:  filepath.Join(homeDir, ".cache", "gls"),
		Indexes:     make(map[string][]string),
		FuzzySearch: false,
		MaxDistance: 2,
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func (c *Config) SaveConfig(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// EnsureStorageDir ensures the storage directory exists
func (c *Config) EnsureStorageDir() error {
	return os.MkdirAll(c.StorageDir, 0755)
}

// GetIndexPath returns the database path for a named index
func (c *Config) GetIndexPath(name string) string {
	if name == "" {
		name = "default"
	}
	return filepath.Join(c.StorageDir, name+".db")
}

// AddIndexPath adds a path to a named index
func (c *Config) AddIndexPath(name, path string) {
	if name == "" {
		name = "default"
	}
	if c.Indexes == nil {
		c.Indexes = make(map[string][]string)
	}
	// Dedup: check if path already exists
	for _, p := range c.Indexes[name] {
		if p == path {
			return
		}
	}
	c.Indexes[name] = append(c.Indexes[name], path)
}

// GetIndexPaths returns paths for a named index
func (c *Config) GetIndexPaths(name string) []string {
	if name == "" {
		name = "default"
	}
	return c.Indexes[name]
}

// ListIndexNames returns all index names
func (c *Config) ListIndexNames() []string {
	names := make([]string, 0, len(c.Indexes))
	for name := range c.Indexes {
		names = append(names, name)
	}
	return names
}

// DeleteIndex removes an index from config
func (c *Config) DeleteIndex(name string) {
	if name == "" {
		name = "default"
	}
	delete(c.Indexes, name)
}
