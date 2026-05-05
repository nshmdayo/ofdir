package history_test

import (
	"path/filepath"
	"testing"

	"github.com/nshmdayo/ofdir/internal/history"
)

func TestRecord(t *testing.T) {
	db, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Record("/home/user/dev/project"); err != nil {
		t.Fatal(err)
	}

	entries, err := db.List(history.SortFrecency, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Path != "/home/user/dev/project" {
		t.Errorf("unexpected path %q", entries[0].Path)
	}
}

func TestGetByIndex(t *testing.T) {
	db, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	paths := []string{"/a", "/b", "/c"}
	for _, p := range paths {
		if err := db.Record(p); err != nil {
			t.Fatal(err)
		}
	}

	entry, err := db.GetByIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
}

func TestClear(t *testing.T) {
	db, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_ = db.Record("/some/path")
	if err := db.Clear(); err != nil {
		t.Fatal(err)
	}
	entries, _ := db.List(history.SortFrecency, 10)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func BenchmarkRecord(b *testing.B) {
	db, _ := history.Open(filepath.Join(b.TempDir(), "history.db"))
	defer db.Close()

	for b.Loop() {
		_ = db.Record("/home/user/dev/project")
	}
}
