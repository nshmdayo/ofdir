package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nshmdayo/ofdir/internal/cli"
	"github.com/nshmdayo/ofdir/internal/config"
)

// runScd sets up isolated XDG dirs, captures stdout, calls Execute with the
// given args, and returns the captured stdout and exit code.
func runScd(t *testing.T, args ...string) (stdout string, exitCode int) {
	t.Helper()

	// Isolate storage to a temp dir.
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("OFDIR_PRINT_PATH", "1")
	t.Setenv("OFDIR_PRINT_PATH", "1")

	// Capture stdout.
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Redirect stderr to discard so test output stays clean.
	origStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)

	// Restore in teardown.
	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	})

	// Inject args.
	origArgs := os.Args
	os.Args = append([]string{"ofdir"}, args...)
	t.Cleanup(func() { os.Args = origArgs })

	err := cli.Execute()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	exitCode = 0
	if err != nil {
		exitCode = cli.ExitCode(err)
	}
	return strings.TrimRight(buf.String(), "\n"), exitCode
}

func TestNoArgs_GoHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	stdout, code := runScd(t)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stdout != home {
		t.Errorf("stdout = %q, want %q", stdout, home)
	}
}

func TestVersion(t *testing.T) {
	_, code := runScd(t, "--version")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestHelp(t *testing.T) {
	_, code := runScd(t, "--help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestInitBash(t *testing.T) {
	stdout, code := runScd(t, "--init", "bash")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "function ofdir()") {
		t.Error("bash init script missing 'function ofdir()'")
	}
}

func TestInitZsh(t *testing.T) {
	stdout, code := runScd(t, "--init", "zsh")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "function ofdir()") {
		t.Error("zsh init script missing 'function ofdir()'")
	}
}

func TestInitUnknownShell(t *testing.T) {
	_, code := runScd(t, "--init", "fish")
	if code == 0 {
		t.Error("expected non-zero exit for unknown shell")
	}
}

func TestBookmarkAddListDelete(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("OFDIR_PRINT_PATH", "1")
	origStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	t.Cleanup(func() { os.Stderr = origStderr })

	// -a: add bookmark for an existing directory
	targetDir := t.TempDir()
	if err := os.Chdir(targetDir); err != nil {
		t.Fatal(err)
	}

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"ofdir", "-a", "testbm"}
	_ = cli.Execute()

	// -l: list should contain our bookmark
	os.Args = []string{"ofdir", "--list-bookmarks"}
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	_ = cli.Execute()
	w.Close()
	os.Stdout = origStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)

	if !strings.Contains(buf.String(), "@testbm") {
		t.Errorf("list output does not contain @testbm: %s", buf.String())
	}

	// @testbm: jump should return targetDir
	os.Args = []string{"ofdir", "@testbm"}
	r, w, _ = os.Pipe()
	os.Stdout = w
	err := cli.Execute()
	w.Close()
	os.Stdout = origStdout
	buf.Reset()
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("bookmark jump failed: %v", err)
	}
	got := strings.TrimRight(buf.String(), "\n")
	// Resolve symlinks for comparison (macOS: /var → /private/var).
	wantResolved, _ := filepath.EvalSymlinks(targetDir)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Errorf("bookmark jump = %q, want %q", got, targetDir)
	}

	// -d: delete
	os.Args = []string{"ofdir", "-d", "testbm"}
	_ = cli.Execute()

	// After delete, jump should fail
	os.Args = []string{"ofdir", "@testbm"}
	r, w, _ = os.Pipe()
	os.Stdout = w
	err = cli.Execute()
	w.Close()
	os.Stdout = origStdout
	if err == nil || cli.ExitCode(err) == 0 {
		t.Error("expected non-zero exit after deleting bookmark")
	}
}

func TestBookmarkJump_NotFound(t *testing.T) {
	_, code := runScd(t, "@nonexistent")
	if code == 0 {
		t.Error("expected non-zero exit for missing bookmark")
	}
}

func TestBookmarkJump_PathGone(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("OFDIR_PRINT_PATH", "1")
	origStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	t.Cleanup(func() { os.Stderr = origStderr })

	// Write a bookmark pointing to a directory that doesn't exist.
	bmFile := config.BookmarksFile()
	if err := os.MkdirAll(filepath.Dir(bmFile), 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`{"bookmarks":[{"name":"gone","path":"%s/nonexistent","created_at":"2026-01-01T00:00:00Z"}]}`, tmp)
	if err := os.WriteFile(bmFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"ofdir", "@gone"}
	err := cli.Execute()
	if err == nil || cli.ExitCode(err) == 0 {
		t.Error("expected non-zero exit when bookmark path no longer exists")
	}
}

func TestRecord(t *testing.T) {
	dir := t.TempDir()
	_, code := runScd(t, "--record", dir)
	if code != 0 {
		t.Errorf("--record exited %d, want 0", code)
	}
}

func TestClearHistory(t *testing.T) {
	_, code := runScd(t, "--clear-history")
	if code != 0 {
		t.Errorf("--clear-history exited %d, want 0", code)
	}
}

func TestSetFuzzyFinder(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("OFDIR_PRINT_PATH", "1")
	origStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	t.Cleanup(func() { os.Stderr = origStderr })

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"ofdir", "--set-fuzzy-finder", "peco"}
	if err := cli.Execute(); err != nil {
		t.Fatalf("--set-fuzzy-finder failed: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}
	if cfg.UI.FuzzyFinder != "peco" {
		t.Fatalf("FuzzyFinder = %q, want %q", cfg.UI.FuzzyFinder, "peco")
	}
}

func TestSetFuzzyFinder_Invalid(t *testing.T) {
	_, code := runScd(t, "--set-fuzzy-finder", "invalid")
	if code == 0 {
		t.Fatal("expected non-zero exit for invalid fuzzy finder")
	}
}

func TestStackPushPop(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("OFDIR_PRINT_PATH", "1")
	origStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	t.Cleanup(func() { os.Stderr = origStderr })

	dir := t.TempDir()

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	origStdout := os.Stdout

	// Push
	os.Args = []string{"ofdir", "-p", dir}
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := cli.Execute(); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	w.Close()
	os.Stdout = origStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if got := strings.TrimRight(buf.String(), "\n"); got != dir {
		t.Errorf("push stdout = %q, want %q", got, dir)
	}

	// Pop
	os.Args = []string{"ofdir", "--"}
	r, w, _ = os.Pipe()
	os.Stdout = w
	if err := cli.Execute(); err != nil {
		t.Fatalf("pop failed: %v", err)
	}
	w.Close()
	os.Stdout = origStdout
	buf.Reset()
	io.Copy(&buf, r)
	if got := strings.TrimRight(buf.String(), "\n"); got != dir {
		t.Errorf("pop stdout = %q, want %q", got, dir)
	}
}

func TestStackPopEmpty(t *testing.T) {
	_, code := runScd(t, "--")
	if code == 0 {
		t.Error("expected non-zero exit when popping empty stack")
	}
}

func TestFuzzySearch_Found(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("OFDIR_PRINT_PATH", "1")

	// Create a subdirectory to find.
	searchRoot := t.TempDir()
	target := filepath.Join(searchRoot, "myproject")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(searchRoot); err != nil {
		t.Fatal(err)
	}

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	t.Cleanup(func() { os.Stderr = origStderr })

	os.Args = []string{"ofdir", "myproject"}
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := cli.Execute()
	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	got := strings.TrimRight(buf.String(), "\n")

	if err != nil {
		t.Fatalf("fuzzy search failed: %v", err)
	}
	// Resolve symlinks for comparison (macOS: /var → /private/var).
	wantResolved, _ := filepath.EvalSymlinks(target)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Errorf("stdout = %q, want %q", got, target)
	}
}

func TestFuzzySearch_NotFound(t *testing.T) {
	searchRoot := t.TempDir()
	if err := os.Chdir(searchRoot); err != nil {
		t.Fatal(err)
	}
	_, code := runScd(t, "zzznomatch")
	if code == 0 {
		t.Error("expected non-zero exit when no match found")
	}
}
