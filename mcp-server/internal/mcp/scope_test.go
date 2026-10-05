package mcp

import (
	"os"
	"path/filepath"
	"runtime"
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

	cases := []struct {
		name, path string
		want       bool
	}{
		{"inside", filepath.Join(src, "a.go"), true},
		{"scope itself", src, true},
		{"nested new file", filepath.Join(src, "x", "y.go"), true},
		{"sibling with shared prefix", filepath.Join(evil, "a.go"), false},
		{"dotdot escape", filepath.Join(src, "..", "secret", "k"), false},
		{"outside", filepath.Join(outside, "k"), false},
	}

	// The symlink-escape case needs an unprivileged os.Symlink. On Windows
	// that requires Administrator or Developer Mode, so on Windows the case is
	// NOT_EXERCISED (skipped) rather than a hard failure; Unix exercises it and
	// the assertion is not weakened.
	if runtime.GOOS == "windows" {
		t.Log("symlink escape case NOT_EXERCISED on Windows: os.Symlink requires elevated privilege")
	} else {
		if err := os.Symlink(outside, filepath.Join(src, "link")); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct {
			name, path string
			want       bool
		}{"symlink escape", filepath.Join(src, "link", "k"), false})
	}

	for _, c := range cases {
		if got := pathWithinScope(c.path, src); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
