package guardrails

// Spec 03 Sec. 4.x — the captured-output bound.
//
// Every execution path (L0, L1, L2) piped the child's stdout and stderr into a
// plain bytes.Buffer. Nothing bounded that buffer, and nothing capped it before
// the bytes reached the MCP result. The declared memory limit therefore applied
// to the *container*, not to the server collecting its output: `max_mb: 1024`
// says the sandboxed process may not use more than a gigabyte, while a command
// printing unbounded output grew the HOST process' heap with no bound at all.
//
// The shape is worse than an ordinary missing limit because the memory bound is
// the control an operator reads when deciding the deployment is safe. A command
// like `yes` inside a 1 GiB container is bounded; the same command's output
// captured by the server is not.
//
// The cap is a published constant rather than a per-policy knob. A knob would
// be a second limit that most deployments never set, and the repository has
// already paid for that pattern twice (swap_allowed, max_wall_seconds and
// max_write_mb were declared and consumed by nothing). One bound, applied on
// every path, with the truncation visible in the result.
const (
	// MaxCapturedOutputBytes is the most stdout (and, separately, stderr) that
	// one sandboxed execution may contribute to the result. It matches the
	// result cap the Rust server publishes for a single tool result
	// (RADICAL_TOOL_MAX_RESULT_BYTES, 1 MiB), so the two products agree on what
	// one tool result is allowed to weigh.
	MaxCapturedOutputBytes = 1 << 20

	// truncationMarker is appended to a stream that hit the cap. A caller that
	// reads only the text — not the boolean flags — must still be unable to
	// mistake a truncated stream for a complete one.
	truncationMarker = "…[output truncated at cap]"
)

// boundedCapture is an io.Writer that keeps at most `limit` bytes and records
// that it dropped the rest.
//
// Write ALWAYS reports the full len(p) as written. That is deliberate: exec's
// copy goroutine treats a short write as an error and stops draining the pipe,
// which would block or SIGPIPE the child. Truncation is a reporting decision,
// never a reason to stop the process under observation.
type boundedCapture struct {
	buf       []byte
	limit     int
	truncated bool
}

func newBoundedCapture(limit int) *boundedCapture {
	if limit <= 0 {
		limit = MaxCapturedOutputBytes
	}
	return &boundedCapture{limit: limit}
}

func (b *boundedCapture) Write(p []byte) (int, error) {
	remaining := b.limit - len(b.buf)
	if remaining <= 0 {
		if len(p) > 0 {
			b.truncated = true
		}
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf = append(b.buf, p[:remaining]...)
		b.truncated = true
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// String returns the captured text, with the marker appended when the stream
// was truncated. An untruncated stream is byte-identical to the child's output.
func (b *boundedCapture) String() string {
	if !b.truncated {
		return string(b.buf)
	}
	return string(b.buf) + truncationMarker
}

// Truncated reports whether any bytes were dropped.
func (b *boundedCapture) Truncated() bool { return b.truncated }
