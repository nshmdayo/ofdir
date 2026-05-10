package fuzzy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nshmdayo/opfd/internal/config"
	"github.com/nshmdayo/opfd/internal/fuzzy"
)

func TestSearch(t *testing.T) {
	root := t.TempDir()
	// Create test directories
	dirs := []string{
		"my-project/src",
		"my-project/dist",
		"other",
		"node_modules/dep",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{
		Search: config.SearchConfig{
			MaxDepth:        5,
			ExcludePatterns: []string{"node_modules", "dist"},
		},
	}

	results, err := fuzzy.Search(root, "project", cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}
	if results[0].Path != filepath.Join(root, "my-project") {
		t.Errorf("unexpected top result: %s", results[0].Path)
	}

	// node_modules should be excluded
	for _, r := range results {
		if filepath.Base(r.Path) == "node_modules" {
			t.Error("node_modules should be excluded")
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	root := b.TempDir()
	// Create 100 directories
	for i := range 100 {
		_ = os.MkdirAll(filepath.Join(root, "project"+string(rune('a'+i%26)), "src"), 0o755)
	}
	cfg := &config.Config{
		Search: config.SearchConfig{
			MaxDepth:        5,
			ExcludePatterns: []string{"node_modules"},
		},
	}
	for b.Loop() {
		_, _ = fuzzy.Search(root, "proj", cfg, nil)
	}
}
