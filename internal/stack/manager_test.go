package stack_test

import (
	"path/filepath"
	"testing"

	"github.com/nshmdayo/ofdir/internal/stack"
)

func TestPushPop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stack")

	s, err := stack.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	_ = s.Push("/home/user/a")
	_ = s.Push("/home/user/b")
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}

	// Reload
	s, _ = stack.Load(path)
	top, err := s.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if top != "/home/user/b" {
		t.Errorf("expected /home/user/b, got %q", top)
	}
}

func TestPopEmpty(t *testing.T) {
	s, _ := stack.Load(filepath.Join(t.TempDir(), "stack"))
	if _, err := s.Pop(); err == nil {
		t.Error("expected error on pop from empty stack")
	}
}
