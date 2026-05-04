package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/nshmdayo/sd/internal/bookmark"
	"github.com/nshmdayo/sd/internal/config"
	"github.com/nshmdayo/sd/internal/fuzzy"
	"github.com/nshmdayo/sd/internal/history"
	"github.com/nshmdayo/sd/internal/output"
	"github.com/nshmdayo/sd/internal/pathutil"
	"github.com/nshmdayo/sd/internal/selector"
	"github.com/nshmdayo/sd/internal/stack"
)

const version = "0.1.0"

// Execute is the main entry point for the sd binary.
func Execute() error {
	cfg, err := config.Load()
	if err != nil {
		output.Errorf("failed to load config: %v", err)
		return err
	}
	output.SetColor(cfg.UI.Color)

	args := os.Args[1:]
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
		output.Path(home)
		return nil
	}

	first := args[0]

	// --- version / help ---
	if first == "--version" || first == "-v" {
		fmt.Fprintf(os.Stderr, "sd version %s\n", version)
		return nil
	}
	if first == "--help" || first == "-h" {
		printHelp()
		return nil
	}

	// --- shell init script ---
	if first == "--init" {
		if len(args) < 2 {
			return outputError("usage: sd --init <bash|zsh>", "")
		}
		return PrintInitScript(args[1])
	}

	// --- record history (called from shell wrapper) ---
	if first == "--record" {
		if len(args) < 2 {
			return outputError("usage: sd --record <path>", "")
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
				return outputError("usage: sd -d <name>", "run 'cd -l' to list available bookmarks")
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
				return outputError("usage: sd -g <query>", "")
			}
			return fuzzyGlobal(args[1], cfg)
		}
		// -p <path>: stack push
		if suffix == "p" {
			if len(args) < 2 {
				return outputError("usage: sd -p <path>", "")
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
		output.Hintf("run 'cd -l' to list available bookmarks")
		return exitCodeError(1)
	}
	if !pathutil.Exists(bm.Path) {
		output.Errorf("path no longer exists: %s", bm.Path)
		output.Hintf("run 'cd -d %s' to remove this bookmark", name)
		return exitCodeError(1)
	}
	output.Path(bm.Path)
	return nil
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
		output.Hintf("run 'cd -l' to list available bookmarks")
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
	output.Path(entry.Path)
	return nil
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
	output.Path(chosen)
	return nil
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
	output.Path(resolved)
	return nil
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
	output.Path(path)
	return nil
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
		output.Path(results[0].Path)
		return nil
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
	output.Path(chosen)
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

func parseShellWords(input string) ([]string, error) {
	var (
		parts   []string
		current strings.Builder
		quote   rune
		escape  bool
	)

	for _, r := range input {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\' && quote != '\'':
			escape = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
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
	return parts, nil
}
		default:
			current.WriteRune(r)
		}
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
	const defaultTOML = `[search]
max_depth        = 5
global_root      = "~"
exclude_patterns = ["node_modules", ".git", "dist", ".cache"]

[history]
max_entries = 1000
sort        = "frecency"   # frecency | time | alpha

[ui]
color        = true
fuzzy_finder = "fzf"       # fzf | peco | internal
`
	return os.WriteFile(path, []byte(defaultTOML), 0o644)
}

// ---- help ----

func printHelp() {
	fmt.Fprint(os.Stderr, `sd - smart cd

Usage:
  cd [query]         Fuzzy search in current directory
  cd @<name>         Jump to bookmark
  cd -N              Jump to history entry N (e.g. cd -1)
  cd -H              Browse history interactively
  cd -a [name]       Add current directory as bookmark
  cd -d <name>       Delete bookmark
  cd -l              List bookmarks
  cd -e              Edit bookmarks file
  cd -g <query>      Global fuzzy search (from home)
  cd -p <path>       Push path onto stack and jump to it
  cd --              Pop from stack (go back)
  cd -s              Show stack
  cd --clear-history Delete all history
  cd --config        Edit config file
  cd --version       Show version
  cd --help          Show this help

Installation:
  eval "$(sd --init bash)"   # add to ~/.bashrc
  eval "$(sd --init zsh)"    # add to ~/.zshrc
`)
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
