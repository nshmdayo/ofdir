package fuzzy

import (
	"cmp"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nshmdayo/opfd/internal/config"
)

// SearchResult holds a candidate directory and its score.
type SearchResult struct {
	Path  string
	Score int
	Depth int
}

// Search walks root recursively and returns directories matching query,
// scored by name similarity and depth. frecencyMap (path→score) is optional.
func Search(root, query string, cfg *config.Config, frecencyMap map[string]float64) ([]SearchResult, error) {
	return walk(root, query, cfg, frecencyMap)
}

// SearchGlobal searches from the configured global root (default: ~).
func SearchGlobal(query string, cfg *config.Config, frecencyMap map[string]float64) ([]SearchResult, error) {
	root := cfg.Search.GlobalRoot
	if root == "~" || strings.HasPrefix(root, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		if root == "~" {
			root = home
		} else {
			root = filepath.Join(home, root[2:])
		}
	}
	return walk(root, query, cfg, frecencyMap)
}

func walk(root, query string, cfg *config.Config, frecencyMap map[string]float64) ([]SearchResult, error) {
	lowerQuery := strings.ToLower(query)
	exclude := cfg.Search.ExcludePatterns
	maxDepth := cfg.Search.MaxDepth

	var results []SearchResult

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable directories
		}
		if !d.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		depth := strings.Count(rel, string(filepath.Separator)) + 1

		name := d.Name()

		// Skip excluded directories.
		for _, pat := range exclude {
			if matched, _ := filepath.Match(pat, name); matched {
				return filepath.SkipDir
			}
		}

		// Stop recursing beyond max depth.
		if depth > maxDepth {
			return filepath.SkipDir
		}

		score := scoreDir(name, path, lowerQuery, depth, maxDepth, frecencyMap)
		if score > 0 {
			results = append(results, SearchResult{Path: path, Score: score, Depth: depth})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(results, func(a, b SearchResult) int {
		return cmp.Compare(b.Score, a.Score) // descending
	})
	return results, nil
}

func scoreDir(name, path, query string, depth, maxDepth int, frecencyMap map[string]float64) int {
	lowerName := strings.ToLower(name)

	var score int
	switch {
	case lowerName == query:
		score += 100
	case strings.HasPrefix(lowerName, query):
		score += 50
	case strings.Contains(lowerName, query):
		score += 20
	default:
		return 0 // no match
	}

	// Depth bonus: shallower is better.
	score += max(0, (maxDepth-depth)*5)

	// Frecency bonus (capped at 30).
	if frecencyMap != nil {
		score += min(30, int(frecencyMap[path]))
	}

	return score
}
