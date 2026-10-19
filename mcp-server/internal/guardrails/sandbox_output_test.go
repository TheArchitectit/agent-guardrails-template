package guardrails

// Spec 03 Sec. 4.x — the captured-output bound, exercised through the REAL
// execution paths.
//
// Before this file the sandbox piped stdout/stderr into a plain bytes.Buffer
// with no limit, so a command that produced unbounded output grew the HOST
// process' heap without bound while the declared memory limit constrained only
// the container. These tests drive the real manager: they are not a unit test
// of a writer, they are the property the execution paths must have.

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"testing"
)

func TestBoundedCaptureKeepsAtMostTheCap(t *testing.T) {
	b := newBoundedCapture(16)
	if _, err := b.Write([]byte("0123456789")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := b.Write([]byte("abcdefghij")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !b.Truncated() {
		t.Fatal("a stream longer than the cap must report truncation")
	}
	if got, want := len(b.buf), 16; got != want {
		t.Fatalf("retained %d bytes, want exactly %d", got, want)
	}
	if !strings.HasSuffix(b.String(), truncationMarker) {
		t.Fatal("a truncated stream must carry the visible marker")
	}
}

func TestBoundedCaptureWriteNeverShortWrites(t *testing.T) {
	// exec's copy goroutine stops draining on a short write, which would block
	// or SIGPIPE the child. Truncation must be a reporting decision only.
	b := newBoundedCapture(4)
	n, err := b.Write([]byte("0123456789"))
	if err != nil {
		t.Fatalf("write returned error: %v", err)
	}
	if n != 10 {
		t.Fatalf("Write reported %d bytes, want the full 10 (a short write stalls the child)", n)
	}
}

func TestBoundedCaptureUntruncatedIsByteIdentical(t *testing.T) {
	b := newBoundedCapture(64)
	if _, err := b.Write([]byte("hello\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b.Truncated() {
		t.Fatal("a stream under the cap must not be flagged")
	}
	if got := b.String(); got != "hello\n" {
		t.Fatalf("under-cap output changed: %q", got)
	}
}

func TestBoundedCaptureZeroLimitFallsBackToThePublishedCap(t *testing.T) {
	// A zero limit is a caller error, not a licence to buffer everything.
	b := newBoundedCapture(0)
	if b.limit != MaxCapturedOutputBytes {
		t.Fatalf("limit %d, want the published cap %d", b.limit, MaxCapturedOutputBytes)
	}
}

// TestExecuteBoundsUnboundedOutput is the requirement itself: the real manager,
// the real L0 path, a command whose output is far larger than the cap.
func TestExecuteBoundsUnboundedOutput(t *testing.T) {
	requireSh(t)
	mgr := NewSandboxManager(nil, slog.Default())

	// Comfortably past the cap: ~2.2 MB of digits against a 1 MiB bound.
	const bytesWanted = MaxCapturedOutputBytes*2 + 1024
	cmd := "yes 0123456789 | head -c " + strconv.Itoa(bytesWanted) + " || true"

	result, err := mgr.Execute(context.Background(), cmd, LevelL0, ResourceLimits{})
	if err != nil {
		// A command that produced output and then exited non-zero is still a
		// result we must be able to inspect; only a nil result is fatal here.
		if result == nil {
			t.Fatalf("execute returned no result: %v", err)
		}
	}
	if len(result.Stdout) > MaxCapturedOutputBytes+len(truncationMarker) {
		t.Fatalf("captured %d bytes of stdout, cap is %d", len(result.Stdout), MaxCapturedOutputBytes)
	}
	if !strings.HasSuffix(result.Stdout, truncationMarker) {
		t.Fatal("an over-cap stdout must be visibly truncated, not silently shortened")
	}
}

// A result under the cap must be untouched — the bound must not become a
// reason to mangle ordinary output.
func TestExecuteLeavesSmallOutputIntact(t *testing.T) {
	requireSh(t)
	mgr := NewSandboxManager(nil, slog.Default())
	result, _ := mgr.Execute(context.Background(), "printf 'hello'", LevelL0, ResourceLimits{})
	if result == nil {
		t.Fatal("no result")
	}
	if result.Stdout != "hello" {
		t.Fatalf("stdout %q, want %q", result.Stdout, "hello")
	}
	if strings.Contains(result.Stdout, truncationMarker) {
		t.Fatal("under-cap output must not carry the truncation marker")
	}
}
