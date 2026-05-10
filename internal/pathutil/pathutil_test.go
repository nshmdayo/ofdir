package pathutil_test

import (
	"os"
	"testing"

	"github.com/nshmdayo/opfd/internal/pathutil"
)

func TestResolve(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		in   string
		want string
	}{
		{"~", home},
		{"~/foo", home + "/foo"},
		{"/tmp", "/tmp"},
	}
	for _, tt := range tests {
		got, err := pathutil.Resolve(tt.in)
		if err != nil {
			t.Errorf("Resolve(%q) error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsSafe(t *testing.T) {
	tests := []struct {
		path string
		safe bool
	}{
		{"/home/user/dev", true},
		{"/home/user/../etc", false},
		{"relative/path", true},
	}
	for _, tt := range tests {
		if got := pathutil.IsSafe(tt.path); got != tt.safe {
			t.Errorf("IsSafe(%q) = %v, want %v", tt.path, got, tt.safe)
		}
	}
}

func TestExists(t *testing.T) {
	tmp := t.TempDir()
	if !pathutil.Exists(tmp) {
		t.Errorf("Exists(%q) should be true", tmp)
	}
	if pathutil.Exists(tmp + "/nonexistent") {
		t.Errorf("Exists(nonexistent) should be false")
	}
}
