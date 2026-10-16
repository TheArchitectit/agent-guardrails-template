package secretfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestPermitsRestrictedOnly pins the mode rule: an owner-only file passes, any
// group or other bit fails.
func TestPermitsRestrictedOnly(t *testing.T) {
	tests := []struct {
		mode os.FileMode
		want bool
	}{
		{0o600, true},
		{0o400, true},
		{0o700, true},
		{0o644, false},
		{0o640, false},
		{0o660, false},
		{0o606, false},
		{0o000, true},
	}
	for _, tc := range tests {
		if got := PermitsRestrictedOnly(tc.mode); got != tc.want {
			t.Errorf("PermitsRestrictedOnly(%#o) = %v, want %v", uint32(tc.mode), got, tc.want)
		}
	}
}

// TestReadAcceptsRestrictedFile proves the positive path returns the file bytes.
func TestReadAcceptsRestrictedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("value-from-a-mounted-secret\n"), Mode); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(data) != "value-from-a-mounted-secret\n" {
		t.Fatalf("Read = %q, want the file contents", data)
	}
}

// TestReadRejectsNonRegularAndPermissive covers the two fail-closed rules.
func TestReadRejectsNonRegularAndPermissive(t *testing.T) {
	dir := t.TempDir()

	if _, err := Read(dir); err == nil {
		t.Error("a directory source must be rejected")
	}
	if _, err := Read(filepath.Join(dir, "absent")); err == nil {
		t.Error("an absent source must be rejected")
	}

	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode check is not enforced on Windows")
	}
	path := filepath.Join(dir, "permissive")
	if err := os.WriteFile(path, []byte("value"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Read(path); err == nil {
		t.Error("a group/world-readable source must be rejected")
	}
}
