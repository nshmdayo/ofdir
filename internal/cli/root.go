package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nshmdayo/ofdir/internal/bookmark"
	"github.com/nshmdayo/ofdir/internal/config"
	"github.com/nshmdayo/ofdir/internal/fuzzy"
	"github.com/nshmdayo/ofdir/internal/history"
	"github.com/nshmdayo/ofdir/internal/output"
	"github.com/nshmdayo/ofdir/internal/pathutil"
	"github.com/nshmdayo/ofdir/internal/selector"
	"github.com/nshmdayo/ofdir/internal/stack"
)

const version = "0.1.0"

var fuzzyFinderOptions = []string{"internal", "fzf", "peco"}

func formatFuzzyFinderOptions(sep string) string {
	return strings.Join(fuzzyFinderOptions, sep)
}

// Execute is the main entry point for the ofdir binary.
func Execute() error {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--set-fuzzy-finder" {
		if len(args) < 2 {
			return outputError(fmt.Sprintf("usage: ofdir --set-fuzzy-finder <%s>", formatFuzzyFinderOptions("|")), "")
		}
		return setFuzzyFinder(args[1])
	}

	cfg, err := config.Load()
	if err != nil {
		output.Errorf("failed to load config: %v", err)
		return err
	}
	output.SetColor(cfg.UI.Color)

	return route(args, cfg)
}

// errorf creates a formatted error (used internally; not printed).
func errorf(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}

// route dispatches args to the appropriate handler.
func route(args []string, cfg *config.Config) error {
	// --- no arguments: go home ---
	if len(args) == 0 {
		home, _ := os.UserHomeDir()

		return enterDirectory(home, cfg)
	}

	first := args[0]

	// --- version / help ---
	if first == "--version" || first == "-v" {
		fmt.Fprintf(os.Stderr, "ofdir version %s\n", version)
		return nil
	}
	if first == "--help" || first == "-h" {
		printHelp()
		return nil
	}

	// --- shell init script ---
	if first == "--init" {
		if len(args) < 2 {
			return outputError("usage: ofdir --init <bash|zsh>", "")
		}
		return PrintInitScript(args[1])
	}

	// --- record history (called from shell wrapper) ---
	if first == "--record" {
		if len(args) < 2 {
			return outputError("usage: ofdir --record <path>", "")
		}
		return recordHistory(args[1], cfg)
	}

	// --- list bookmarks for shell completion ---
	if first == "--list-bookmarks" {
		return listBookmarkNames(cfg)
	}

	// --- clear history ---
	if first == "--clear-history" {
		return clearHistory(cfg)
	}

	// --- config edit ---
	if first == "--config" {
		return editConfig()
	}
	if first == "--set-fuzzy-finder" {
		if len(args) < 2 {
			return outputError(fmt.Sprintf("usage: ofdir --set-fuzzy-finder <%s>", formatFuzzyFinderOptions("|")), "")
		}
		return setFuzzyFinder(args[1])
	}

	// --- bookmark jump: @name ---
	if strings.HasPrefix(first, "@") {
		name := first[1:]
		return bookmarkJump(name, cfg)
	}

	// --- history jump: -N (numeric) ---
	if strings.HasPrefix(first, "-") && len(first) > 1 {
		suffix := first[1:]

		// -H: interactive history
		if suffix == "H" {
			return historyInteractive(cfg)
		}
		// -a [name]: add bookmark
		if suffix == "a" {
			name := ""
			if len(args) >= 2 {
				name = args[1]
			}
			return bookmarkAdd(name, cfg)
		}
		// -d <name>: delete bookmark
		if suffix == "d" {
			if len(args) < 2 {
				return outputError("usage: ofdir -d <name>", "run 'ofdir -l' to list available bookmarks")
			}
			return bookmarkDelete(args[1], cfg)
		}
		// -l: list bookmarks
		if suffix == "l" {
			return bookmarkList(cfg)
		}
		// -e: edit bookmarks file
		if suffix == "e" {
			return bookmarkEdit(cfg)
		}
		// -g <query>: global fuzzy search
		if suffix == "g" {
			if len(args) < 2 {
				return outputError("usage: ofdir -g <query>", "")
			}
			return fuzzyGlobal(args[1], cfg)
		}
		// -p <path>: stack push
		if suffix == "p" {
			if len(args) < 2 {
				return outputError("usage: ofdir -p <path>", "")
			}
			return stackPush(args[1], cfg)
		}
		// -s: stack list
		if suffix == "s" {
			return stackList(cfg)
		}
		// --: stack pop
		if first == "--" {
			return stackPop(cfg)
		}
		// -N: history jump by index
		if n, err := strconv.Atoi(suffix); err == nil && n > 0 {
			return historyJumpN(n, cfg)
		}
	}

	// --- default: fuzzy search from cwd ---
	cwd, err := os.Getwd()
	if err != nil {
		return outputError(fmt.Sprintf("cannot determine working directory: %v", err), "")
	}
	return fuzzySearch(cwd, first, cfg)
}

// ---- bookmark operations ----

func bookmarkJump(name string, cfg *config.Config) error {
	store, err := bookmark.Load(config.BookmarksFile())
	if err != nil {
		return exitError(fmt.Sprintf("failed to load bookmarks: %v", err), 2)
	}
	bm, err := store.Find(name)
	if err != nil {
		output.Errorf("bookmark %q not found", name)
		output.Hintf("run 'ofdir -l' to list available bookmarks")
		return exitCodeError(1)
	}
	if !pathutil.Exists(bm.Path) {
		output.Errorf("path no longer exists: %s", bm.Path)
		output.Hintf("run 'ofdir -d %s' to remove this bookmark", name)
		return exitCodeError(1)
	}
	return enterDirectory(bm.Path, cfg)
}

func bookmarkAdd(name string, cfg *config.Config) error {
	cwd, err := os.Getwd()
	if err != nil {
		return outputError(fmt.Sprintf("cannot determine working directory: %v", err), "")
	}
	if name == "" {
		name = strings.ReplaceAll(cwd[strings.LastIndex(cwd, "/")+1:], " ", "-")
	}

	bmFile := config.BookmarksFile()
	store, err := bookmark.Load(bmFile)
	if err != nil {
		return exitError(fmt.Sprintf("failed to load bookmarks: %v", err), 2)
	}
	if err := store.Add(name, cwd); err != nil {
		return outputError(err.Error(), "")
	}
	if err := store.Save(bmFile); err != nil {
		return exitError(fmt.Sprintf("failed to save bookmarks: %v", err), 2)
	}
	output.Successf("Bookmark %q added → %s", name, cwd)
	return nil
}

func bookmarkDelete(name string, cfg *config.Config) error {
	bmFile := config.BookmarksFile()
	store, err := bookmark.Load(bmFile)
	if err != nil {
		return exitError(fmt.Sprintf("failed to load bookmarks: %v", err), 2)
	}
	if err := store.Delete(name); err != nil {
		output.Errorf("bookmark %q not found", name)
		output.Hintf("run 'ofdir -l' to list available bookmarks")
		return exitCodeError(1)
	}
	if err := store.Save(bmFile); err != nil {
		return exitError(fmt.Sprintf("failed to save bookmarks: %v", err), 2)
	}
	output.Successf("Bookmark %q deleted", name)
	return nil
}

func bookmarkList(cfg *config.Config) error {
	store, err := bookmark.Load(config.BookmarksFile())
	if err != nil {
		return exitError(fmt.Sprintf("failed to load bookmarks: %v", err), 2)
	}
	bms := store.List()
	if len(bms) == 0 {
		output.Infof("No bookmarks. Add one with 'cd -a [name]'")
		return nil
	}
	for i, bm := range bms {
		output.Infof("  %d  %-20s  %s", i+1, bm.Name, bm.Path)
	}
	return nil
}

func listBookmarkNames(cfg *config.Config) error {
	store, err := bookmark.Load(config.BookmarksFile())
	if err != nil {
		return nil // silently ignore for completion
	}
	for _, name := range store.Names() {
		fmt.Println("@" + name)
	}
	return nil
}

func bookmarkEdit(cfg *config.Config) error {
	bmFile := config.BookmarksFile()
	// Ensure file exists before editing.
	if _, err := os.Stat(bmFile); errors.Is(err, os.ErrNotExist) {
		store := &bookmark.Store{}
		_ = store.Save(bmFile)
	}
	return openWithEditor(bmFile)
}

// ---- history operations ----

func historyJumpN(n int, cfg *config.Config) error {
	db, err := history.Open(config.HistoryDB())
	if err != nil {
		return exitError(fmt.Sprintf("failed to open history db: %v", err), 2)
	}
	defer db.Close()

	entry, err := db.GetByIndex(n)
	if err != nil {
		output.Errorf("%v", err)
		output.Hintf("run 'cd -H' to browse history interactively")
		return exitCodeError(1)
	}
	if !pathutil.Exists(entry.Path) {
		output.Errorf("path no longer exists: %s", entry.Path)
		return exitCodeError(1)
	}
	return enterDirectory(entry.Path, cfg)
}

func historyInteractive(cfg *config.Config) error {
	db, err := history.Open(config.HistoryDB())
	if err != nil {
		return exitError(fmt.Sprintf("failed to open history db: %v", err), 2)
	}
	defer db.Close()

	sort := history.SortFrecency
	if cfg.History.Sort == "time" {
		sort = history.SortTime
	} else if cfg.History.Sort == "alpha" {
		sort = history.SortAlpha
	}

	entries, err := db.List(sort, cfg.History.MaxEntries)
	if err != nil {
		return exitError(fmt.Sprintf("failed to list history: %v", err), 2)
	}
	if len(entries) == 0 {
		output.Infof("No history entries yet.")
		return exitCodeError(1)
	}

	candidates := make([]string, len(entries))
	for i, e := range entries {
		candidates[i] = e.Path
	}

	chosen, err := selectPath(candidates, cfg, "history> ")
	if err != nil {
		return err
	}
	if !pathutil.Exists(chosen) {
		output.Errorf("path no longer exists: %s", chosen)
		return exitCodeError(1)
	}
	return enterDirectory(chosen, cfg)
}

func clearHistory(cfg *config.Config) error {
	db, err := history.Open(config.HistoryDB())
	if err != nil {
		return exitError(fmt.Sprintf("failed to open history db: %v", err), 2)
	}
	defer db.Close()
	if err := db.Clear(); err != nil {
		return outputError(fmt.Sprintf("failed to clear history: %v", err), "")
	}
	output.Successf("History cleared.")
	return nil
}

func recordHistory(path string, cfg *config.Config) error {
	resolved, err := pathutil.Resolve(path)
	if err != nil || !pathutil.IsSafe(resolved) {
		return nil // silently ignore unsafe paths
	}
	db, err := history.Open(config.HistoryDB())
	if err != nil {
		return nil // best-effort; don't break the shell
	}
	defer db.Close()
	_ = db.Record(resolved)
	_ = db.Prune(cfg.History.MaxEntries)
	return nil
}

// ---- stack operations ----

func stackPush(path string, cfg *config.Config) error {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return outputError(fmt.Sprintf("cannot resolve path: %v", err), "")
	}
	if !pathutil.IsSafe(resolved) {
		return outputError("unsafe path rejected", "")
	}
	if !pathutil.Exists(resolved) {
		output.Errorf("directory does not exist: %s", resolved)
		return exitCodeError(1)
	}

	sf := config.StackFile()
	s, err := stack.Load(sf)
	if err != nil {
		return exitError(fmt.Sprintf("failed to load stack: %v", err), 2)
	}
	_ = s.Push(resolved)
	if err := s.Save(sf); err != nil {
		return exitError(fmt.Sprintf("failed to save stack: %v", err), 2)
	}
	return enterDirectory(resolved, cfg)
}

func stackPop(cfg *config.Config) error {
	sf := config.StackFile()
	s, err := stack.Load(sf)
	if err != nil {
		return exitError(fmt.Sprintf("failed to load stack: %v", err), 2)
	}
	path, err := s.Pop()
	if err != nil {
		output.Errorf("%v", err)
		output.Hintf("use 'cd -p <path>' to push a directory")
		return exitCodeError(1)
	}
	if err := s.Save(sf); err != nil {
		return exitError(fmt.Sprintf("failed to save stack: %v", err), 2)
	}
	if !pathutil.Exists(path) {
		output.Errorf("path no longer exists: %s", path)
		return exitCodeError(1)
	}
	return enterDirectory(path, cfg)
}

func stackList(cfg *config.Config) error {
	sf := config.StackFile()
	s, err := stack.Load(sf)
	if err != nil {
		return exitError(fmt.Sprintf("failed to load stack: %v", err), 2)
	}
	entries := s.List()
	if len(entries) == 0 {
		output.Infof("Stack is empty. Use 'cd -p <path>' to push a directory.")
		return nil
	}
	for i, entry := range slices.Backward(entries) {
		output.Infof("  %d  %s", len(entries)-i, entry)
	}
	return nil
}

// ---- fuzzy search ----

func fuzzySearch(root, query string, cfg *config.Config) error {
	frecencyMap := loadFrecencyMap(cfg)

	results, err := fuzzy.Search(root, query, cfg, frecencyMap)
	if err != nil {
		return outputError(fmt.Sprintf("search failed: %v", err), "")
	}
	return handleResults(results, cfg)
}

func fuzzyGlobal(query string, cfg *config.Config) error {
	frecencyMap := loadFrecencyMap(cfg)

	results, err := fuzzy.SearchGlobal(query, cfg, frecencyMap)
	if err != nil {
		return outputError(fmt.Sprintf("global search failed: %v", err), "")
	}
	return handleResults(results, cfg)
}

func handleResults(results []fuzzy.SearchResult, cfg *config.Config) error {
	if len(results) == 0 {
		output.Errorf("no matching directory found")
		return exitCodeError(1)
	}
	if len(results) == 1 {
		return enterDirectory(results[0].Path, cfg)
	}

	candidates := make([]string, len(results))
	for i, r := range results {
		candidates[i] = r.Path
	}

	return selectAndOutputPath(candidates, cfg, "cd> ")
}

func selectPath(candidates []string, cfg *config.Config, prompt string) (string, error) {
	sel := selector.New(cfg)
	chosen, err := sel.Select(candidates, prompt)
	if err != nil {
		if errors.Is(err, selector.ErrCancelled) {
			return "", exitCodeError(130)
		}
		return "", outputError(err.Error(), "")
	}
	return chosen, nil
}

func selectAndOutputPath(candidates []string, cfg *config.Config, prompt string) error {
	chosen, err := selectPath(candidates, cfg, prompt)
	if err != nil {
		return err
	}
	return enterDirectory(chosen, cfg)
}

func enterDirectory(path string, cfg *config.Config) error {
	output.Path(path)
	_ = recordHistory(path, cfg)
	return nil
}

func loadFrecencyMap(cfg *config.Config) map[string]float64 {
	db, err := history.Open(config.HistoryDB())
	if err != nil {
		return nil
	}
	defer db.Close()
	m, _ := db.FrecencyMap()
	return m
}

// ---- config edit ----

func editConfig() error {
	cfgFile := config.ConfigFile()
	if _, err := os.Stat(cfgFile); errors.Is(err, os.ErrNotExist) {
		if err := writeDefaultConfig(cfgFile); err != nil {
			return outputError(fmt.Sprintf("failed to create config: %v", err), "")
		}
	}
	return openWithEditor(cfgFile)
}

func setFuzzyFinder(name string) error {
	if !slices.Contains(fuzzyFinderOptions, name) {
		return outputError(fmt.Sprintf("invalid fuzzy finder (must be one of: %s)", formatFuzzyFinderOptions(", ")), "")
	}

	cfgFile := config.ConfigFile()
	if _, err := os.Stat(cfgFile); errors.Is(err, os.ErrNotExist) {
		if err := writeDefaultConfig(cfgFile); err != nil {
			return outputError(fmt.Sprintf("failed to create config: %v", err), "")
		}
	}

	cfg := config.Defaults()
	if _, err := toml.DecodeFile(cfgFile, cfg); err != nil {
		// If existing config is malformed, recover by writing defaults + requested value.
		output.Infof("config is malformed, rewriting with defaults: %v", err)
	}
	cfg.UI.FuzzyFinder = name

	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(cfg); err != nil {
		return outputError(fmt.Sprintf("failed to encode config: %v", err), "")
	}
	tmpFile := cfgFile + ".tmp"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return outputError(fmt.Sprintf("failed to create temp config: %v", err), "")
	}
	if _, err := f.Write(b.Bytes()); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return outputError(fmt.Sprintf("failed to close temp config after write error: %v", closeErr), "")
		}
		return outputError(fmt.Sprintf("failed to write temp config: %v", err), "")
	}
	if err := f.Sync(); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return outputError(fmt.Sprintf("failed to close temp config after sync error: %v", closeErr), "")
		}
		return outputError(fmt.Sprintf("failed to sync temp config: %v", err), "")
	}
	if err := f.Close(); err != nil {
		return outputError(fmt.Sprintf("failed to close temp config: %v", err), "")
	}
	if err := os.Rename(tmpFile, cfgFile); err != nil {
		return outputError(fmt.Sprintf("failed to replace config: %v", err), "")
	}
	output.Successf("fuzzy_finder set to %q", name)
	return nil
}

func parseShellWords(input string) ([]string, error) {
	var (
		parts   []string
		current strings.Builder
		quote   rune
		escape  bool
		inToken bool
	)

	for _, r := range input {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
			inToken = true
		case r == '\\' && quote != '\'':
			escape = true
			inToken = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			inToken = true
		case r == '\'' || r == '"':
			quote = r
			inToken = true
		case r == ' ' || r == '\t' || r == '\n':
			if inToken {
				parts = append(parts, current.String())
				current.Reset()
				inToken = false
			}
		default:
			current.WriteRune(r)
			inToken = true
		}
	}

	if inToken {
		parts = append(parts, current.String())
	}
	if escape {
		return nil, fmt.Errorf("unterminated escape in editor command")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in editor command")
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts, nil
}

func openWithEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	parts, err := parseShellWords(editor)
	if err != nil || len(parts) == 0 || parts[0] == "" {
		parts = []string{"vi"}
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeDefaultConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	defaultTOML := fmt.Sprintf(`[search]
max_depth        = 5
global_root      = "~"
exclude_patterns = ["node_modules", ".git", "dist", ".cache"]

[history]
max_entries = 1000
sort        = "frecency"   # frecency | time | alpha

[ui]
color        = true
fuzzy_finder = "internal"  # %s
`, formatFuzzyFinderOptions(" | "))
	return os.WriteFile(path, []byte(defaultTOML), 0o644)
}

// ---- help ----

func printHelp() {
	fmt.Fprintf(os.Stderr, `ofdir - smart directory CLI

Usage:
  ofdir [query]         Fuzzy search in current directory and resolve destination path
  ofdir @<name>         Jump to bookmark
  ofdir -N              Jump to history entry N (e.g. ofdir -1)
  ofdir -H              Browse history interactively
  ofdir -a [name]       Add current directory as bookmark
  ofdir -d <name>       Delete bookmark
  ofdir -l              List bookmarks
  ofdir -e              Edit bookmarks file
  ofdir -g <query>      Global fuzzy search (from home)
  ofdir -p <path>       Push path onto stack and jump to it
  ofdir --              Pop from stack (go back)
  ofdir -s              Show stack
  ofdir --clear-history Delete all history
  ofdir --config        Edit config file
  ofdir --set-fuzzy-finder <%s>
                      Set fuzzy finder in config
  ofdir --version       Show version
  ofdir --help          Show this help

Environment:
`, formatFuzzyFinderOptions("|"))
}

// ---- error helpers ----

// exitCodeError signals a specific exit code without printing anything.
type exitCodeErr struct{ code int }

func (e exitCodeErr) Error() string { return fmt.Sprintf("exit %d", e.code) }

func exitCodeError(code int) error { return exitCodeErr{code: code} }

// ExitCode extracts exit code from an error (default 1).
func ExitCode(err error) int {
	var e exitCodeErr
	if errors.As(err, &e) {
		return e.code
	}
	return 1
}

func outputError(msg, hint string) error {
	output.Errorf("%s", msg)
	if hint != "" {
		output.Hintf("%s", hint)
	}
	return exitCodeError(1)
}

func exitError(msg string, code int) error {
	output.Errorf("%s", msg)
	return exitCodeError(code)
}
