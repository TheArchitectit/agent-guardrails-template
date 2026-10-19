package guardrails

// Spec 03 Sec. 3.4 / Sec. 5.1 — the L2 mount plan, as ONE decision.
//
// This file exists because the mount set was assembled in two places that did
// not know about each other, and the shipped default configuration could not
// run at L2 at all:
//
//	containerHardeningArgs  always emitted  --tmpfs /tmp:rw,noexec,size=<max_write_mb>m
//	execL2                  also emitted    -v /tmp:/tmp:rw     (from Disk.ReadWritePaths)
//
// `--tmpfs /tmp` and `-v /tmp:/tmp:rw` are two mounts at the same destination.
// The runtime refuses the second one before the container exists:
//
//	podman: Error: /tmp: duplicate mount destination            (exit 125)
//	docker: Docker daemon: Duplicate mount point: /tmp          (exit 125)
//
// Exit 125 is classified by isSetupError, so the failure was silent: Execute
// downgraded L2 -> L1, logged a warning, and returned a successful result. The
// caller received a result and had no way to see that the isolation level it
// asked for had never been available. `DefaultSandboxConfig()` declares
// `/tmp` writable and every documented example repeats it, so this was not a
// corner case — it was every default configuration on every host.
//
// The class is the one this spec has already fixed three times (a declared
// control with no consumer): a declared writable path whose effect was either a
// crashed runtime or a silently different isolation level. The fix is not "move
// one line" — it is to make the whole mount set one function with one set of
// rules, so a bound cannot be satisfied on one side and contradicted on the
// other.

import (
	"fmt"
	"path/filepath"
)

// scratchTmpfs are the destinations the sandbox always provides as a bounded
// writable tmpfs. They are reserved: a policy may declare them writable (and
// the shipped defaults do), and that declaration is satisfied BY the tmpfs —
// it never becomes a host bind mount.
//
// This is the security-relevant half of the decision. Binding the host's /tmp
// into the container would hand the command the host's writable scratch space
// with no size bound at all, replacing an enforced `max_write_mb` with nothing.
// The declared intent ("/tmp is writable inside the sandbox") is met by the
// tmpfs; the host directory stays outside.
var scratchTmpfs = []string{"/tmp", "/var/tmp"}

// isReservedScratch reports whether dest is a reserved tmpfs destination.
func isReservedScratch(dest string) bool {
	clean := filepath.Clean(dest)
	for _, r := range scratchTmpfs {
		if clean == r {
			return true
		}
	}
	return false
}

// containerMountArgs returns the mount arguments for one L2 execution.
//
// Order is deliberate: bind mounts first, tmpfs last. A runtime applies mounts
// in argument order, so a later tmpfs at /tmp shadows a nested earlier bind
// (a policy declaring "/" writable cannot hand the command an unbounded /tmp
// through a broad bind). The bounded surface wins by construction rather than
// by whichever flag happened to be appended first.
//
// Errors are configuration defects, not setup failures: the caller must not
// treat them as licence to fall back to a lower isolation level, because the
// fallback would run with NO path set at all.
func containerMountArgs(limits ResourceLimits) ([]string, error) {
	rw := limits.Disk.ReadWritePaths
	ro := limits.Disk.ReadOnlyPaths

	// A path declared both writable and read-only cannot be either: there is
	// no order in which the two mounts agree. Refused rather than resolved,
	// because picking one silently would make the effective policy depend on
	// argument order.
	seenRW := make(map[string]bool, len(rw))
	for _, p := range rw {
		seenRW[filepath.Clean(p)] = true
	}
	for _, p := range ro {
		if seenRW[filepath.Clean(p)] {
			return nil, fmt.Errorf("sandbox path %q is declared both read_write_paths and read_only_paths; declare one", p)
		}
	}

	var args []string
	// Destinations already mounted, so a bind can never be emitted twice and a
	// bind can never land on a reserved scratch destination.
	destinations := make(map[string]bool, len(rw)+len(ro)+len(scratchTmpfs))
	var bindOrder []string

	addBind := func(path, mode string) error {
		if err := validateMountPath(path); err != nil {
			return err
		}
		abs := filepath.Clean(path)
		if !filepath.IsAbs(abs) {
			return fmt.Errorf("invalid bind-mount path %q: must be absolute", path)
		}
		if isReservedScratch(abs) {
			// The declaration is satisfied by the bounded tmpfs. Returning
			// here (rather than emitting -v) is the fix for the duplicate
			// destination: two mounts cannot both own /tmp.
			return nil
		}
		if destinations[abs] {
			// Same destination from two spellings of one path (or a repeated
			// entry). One mount owns a destination; a second one is the
			// runtime error that started this.
			return nil
		}
		destinations[abs] = true
		bindOrder = append(bindOrder, fmt.Sprintf("%s:%s:%s", abs, abs, mode))
		return nil
	}

	for _, p := range rw {
		if err := addBind(p, "rw"); err != nil {
			return nil, err
		}
	}
	for _, p := range ro {
		if err := addBind(p, "ro"); err != nil {
			return nil, err
		}
	}

	for _, spec := range bindOrder {
		args = append(args, "-v", spec)
	}

	// The bounded scratch, last so it wins any nested destination.
	args = append(args,
		"--tmpfs", fmt.Sprintf("/tmp:rw,noexec,size=%dm", tmpfsSizeMB(limits, 64)),
		"--tmpfs", "/var/tmp:rw,noexec,size=32m",
	)
	return args, nil
}
