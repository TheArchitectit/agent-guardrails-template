package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathWithinScope(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	evil := filepath.Join(root, "src-evil")
	outside := filepath.Join(root, "secret")
	for _, d := range []string{src, evil, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(src, "link")); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, path string
		want       bool
	}{
		{"inside", filepath.Join(src, "a.go"), true},
		{"scope itself", src, true},
		{"nested new file", filepath.Join(src, "x", "y.go"), true},
		{"sibling with shared prefix", filepath.Join(evil, "a.go"), false},
		{"dotdot escape", filepath.Join(src, "..", "secret", "k"), false},
		{"symlink escape", filepath.Join(src, "link", "k"), false},
		{"outside", filepath.Join(outside, "k"), false},
	}
	for _, c := range cases {
		if got := pathWithinScope(c.path, src); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
