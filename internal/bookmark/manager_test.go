package bookmark_test

import (
	"path/filepath"
	"testing"

	"github.com/nshmdayo/opfd/internal/bookmark"
)

func TestAddFindDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.json")

	store, err := bookmark.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Add("myproj", "/home/user/dev/myproj"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(path); err != nil {
		t.Fatal(err)
	}

	// Reload
	store, err = bookmark.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	bm, err := store.Find("myproj")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if bm.Path != "/home/user/dev/myproj" {
		t.Errorf("got path %q", bm.Path)
	}

	// Delete
	if err := store.Delete("myproj"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Find("myproj"); err == nil {
		t.Error("expected error after delete")
	}
}

func TestLoadMissing(t *testing.T) {
	store, err := bookmark.Load("/nonexistent/bookmarks.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(store.List()) != 0 {
		t.Error("expected empty store")
	}
}
