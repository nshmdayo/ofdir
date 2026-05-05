package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nshmdayo/ofdir/internal/config"
)

func TestDefaults(t *testing.T) {
	// No config file, no env vars → defaults apply.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OFDIR_MAX_DEPTH", "")
	t.Setenv("NO_COLOR", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Search.MaxDepth != 5 {
		t.Errorf("MaxDepth = %d, want 5", cfg.Search.MaxDepth)
	}
	if cfg.History.MaxEntries != 1000 {
		t.Errorf("MaxEntries = %d, want 1000", cfg.History.MaxEntries)
	}
	if cfg.UI.FuzzyFinder != "fzf" {
		t.Errorf("FuzzyFinder = %q, want fzf", cfg.UI.FuzzyFinder)
	}
	if !cfg.UI.Color {
		t.Error("Color should default to true")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "ofdir")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `
[search]
max_depth = 10

[history]
max_entries = 500
sort = "time"

[ui]
fuzzy_finder = "peco"
color = false
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Search.MaxDepth != 10 {
		t.Errorf("MaxDepth = %d, want 10", cfg.Search.MaxDepth)
	}
	if cfg.History.MaxEntries != 500 {
		t.Errorf("MaxEntries = %d, want 500", cfg.History.MaxEntries)
	}
	if cfg.History.Sort != "time" {
		t.Errorf("Sort = %q, want time", cfg.History.Sort)
	}
	if cfg.UI.FuzzyFinder != "peco" {
		t.Errorf("FuzzyFinder = %q, want peco", cfg.UI.FuzzyFinder)
	}
	if cfg.UI.Color {
		t.Error("Color should be false")
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OFDIR_MAX_DEPTH", "3")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Search.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3 (from env)", cfg.Search.MaxDepth)
	}
}

func TestNoColorEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NO_COLOR", "1")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UI.Color {
		t.Error("Color should be false when NO_COLOR is set")
	}
}

func TestXDGPaths(t *testing.T) {
	cfgHome := t.TempDir()
	dataHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("XDG_DATA_HOME", dataHome)

	if got, want := config.ConfigFile(), filepath.Join(cfgHome, "ofdir", "config.toml"); got != want {
		t.Errorf("ConfigFile() = %q, want %q", got, want)
	}
	if got, want := config.BookmarksFile(), filepath.Join(cfgHome, "ofdir", "bookmarks.json"); got != want {
		t.Errorf("BookmarksFile() = %q, want %q", got, want)
	}
	if got, want := config.HistoryDB(), filepath.Join(dataHome, "ofdir", "history.db"); got != want {
		t.Errorf("HistoryDB() = %q, want %q", got, want)
	}
	if got, want := config.StackFile(), filepath.Join(dataHome, "ofdir", "stack"); got != want {
		t.Errorf("StackFile() = %q, want %q", got, want)
	}
}
