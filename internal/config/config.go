package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Search  SearchConfig  `toml:"search"`
	History HistoryConfig `toml:"history"`
	UI      UIConfig      `toml:"ui"`
}

type SearchConfig struct {
	MaxDepth        int      `toml:"max_depth"`
	GlobalRoot      string   `toml:"global_root"`
	ExcludePatterns []string `toml:"exclude_patterns"`
}

type HistoryConfig struct {
	MaxEntries int    `toml:"max_entries"`
	Sort       string `toml:"sort"`
}

type UIConfig struct {
	Color       bool   `toml:"color"`
	FuzzyFinder string `toml:"fuzzy_finder"`
}

func defaults() *Config {
	return &Config{
		Search: SearchConfig{
			MaxDepth:        5,
			GlobalRoot:      "~",
			ExcludePatterns: []string{"node_modules", ".git", "dist", ".cache"},
		},
		History: HistoryConfig{
			MaxEntries: 1000,
			Sort:       "frecency",
		},
		UI: UIConfig{
			Color:       true,
			FuzzyFinder: "internal",
		},
	}
}

// ConfigDir returns the ofdir config directory (XDG_CONFIG_HOME/ofdir).
func ConfigDir() string {
	return appDir("XDG_CONFIG_HOME", filepath.Join(".config"))
}

// DataDir returns the ofdir data directory (XDG_DATA_HOME/ofdir).
func DataDir() string {
	return appDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

func appDir(envVar, fallback string) string {
	base := os.Getenv(envVar)
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, fallback)
	}
	return filepath.Join(base, "ofdir")
}

// ConfigFile returns the path to config.toml.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

// BookmarksFile returns the path to bookmarks.json.
func BookmarksFile() string {
	return filepath.Join(ConfigDir(), "bookmarks.json")
}

// HistoryDB returns the path to history.db.
func HistoryDB() string {
	return filepath.Join(DataDir(), "history.db")
}

// StackFile returns the path to the stack file.
func StackFile() string {
	return filepath.Join(DataDir(), "stack")
}

// Load reads config from file and applies environment variable overrides.
func Load() (*Config, error) {
	cfg := defaults()

	path := ConfigFile()
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, err
		}
	}

	applyEnv(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("OFDIR_MAX_DEPTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Search.MaxDepth = n
		}
	}
	if v := os.Getenv("NO_COLOR"); v != "" {
		cfg.UI.Color = false
	}
}
