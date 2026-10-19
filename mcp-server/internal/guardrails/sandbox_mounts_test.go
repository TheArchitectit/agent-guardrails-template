package guardrails

// Spec 03 Sec. 3.4 / Sec. 5.1 — the L2 mount plan.
//
// The defect these tests pin was not a wrong branch; it was two correct-looking
// halves that met at one destination:
//
//	containerHardeningArgs: --tmpfs /tmp:rw,noexec,size=<n>m   (always)
//	execL2:                 -v /tmp:/tmp:rw                    (when /tmp is declared writable)
//
// `DefaultSandboxConfig()` declares `/tmp` writable, so the SHIPPED DEFAULT was
// a container the runtime refuses to create:
//
//	podman: Error: /tmp: duplicate mount destination       (exit 125)
//	docker: Duplicate mount point: /tmp                    (exit 125)
//
// and exit 125 is `isSetupError`, so Execute downgraded to L1, logged a
// warning, and returned a successful result. Every default deployment silently
// ran one isolation level below the one it asked for, and nothing in the result
// said so.
//
// The central test is therefore a POST-CONDITION ON THE GENERATED ARGV, not an
// assertion about which function emitted which flag. A per-flag assertion would
// have passed against the broken tree — each half was correct in isolation.
// Asserting that no destination appears twice cannot.

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// mountDestinations extracts every destination the argv mounts, from both
// `-v <src>:<dst>[:<mode>]` and `--tmpfs <dst>:<opts>`. It is deliberately
// derived from the argv rather than from the plan's intermediate state: the
// runtime only ever sees the argv.
func mountDestinations(args []string) []string {
	var dests []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-v", "--volume":
			if i+1 >= len(args) {
				continue
			}
			parts := strings.Split(args[i+1], ":")
			if len(parts) >= 2 {
				dests = append(dests, parts[1])
			}
		case "--tmpfs":
			if i+1 >= len(args) {
				continue
			}
			parts := strings.SplitN(args[i+1], ":", 2)
			dests = append(dests, parts[0])
		}
	}
	return dests
}

// TestNoDestinationIsMountedTwice is the post-condition. It runs against the
// SHIPPED configuration, because that is the configuration that was broken.
func TestNoDestinationIsMountedTwice(t *testing.T) {
	cfg := DefaultSandboxConfig()

	// Every policy, resolved as the execution path resolves it, plus the
	// globals: a defect in any one of them is the same defect.
	limits := []ResourceLimits{cfg.GlobalDefaults}
	for _, name := range cfg.ToolPolicyNames() {
		limits = append(limits, cfg.ResolveToolPolicyLimits(name))
	}

	for i, l := range limits {
		args, err := containerMountArgs(l)
		if err != nil {
			t.Fatalf("limits[%d]: containerMountArgs: %v", i, err)
		}
		seen := make(map[string]int)
		for _, d := range mountDestinations(args) {
			seen[d]++
		}
		for d, n := range seen {
			if n > 1 {
				t.Errorf("limits[%d]: destination %q is mounted %d times (argv: %v); "+
					"the runtime refuses this with exit 125, which the fallback chain "+
					"reads as a setup error and silently downgrades", i, d, n, args)
			}
		}
	}
}

// TestDocumentedDefaultExampleHasNoDuplicateDestination pins the exact input
// that failed: the shipped defaults declare /tmp writable and the tmpfs is
// always emitted.
func TestDocumentedDefaultExampleHasNoDuplicateDestination(t *testing.T) {
	def := DefaultSandboxConfig().GlobalDefaults
	if len(def.Disk.ReadWritePaths) == 0 {
		t.Fatal("the shipped defaults declare no writable path; this test's premise is gone and it must be rewritten, not deleted")
	}
	hasTmp := false
	for _, p := range def.Disk.ReadWritePaths {
		if p == "/tmp" {
			hasTmp = true
		}
	}
	if !hasTmp {
		t.Fatalf("shipped read_write_paths = %v, expected /tmp to be the failing case", def.Disk.ReadWritePaths)
	}

	args, err := containerMountArgs(def)
	if err != nil {
		t.Fatalf("containerMountArgs: %v", err)
	}
	dests := mountDestinations(args)
	seen := make(map[string]int)
	for _, d := range dests {
		seen[d]++
	}
	if seen["/tmp"] > 1 {
		t.Fatalf("/tmp mounted %d times: %v", seen["/tmp"], args)
	}
}

// TestDeclaredScratchPathIsSatisfiedByTheBoundedTmpfs is the security half.
//
// Binding the host's /tmp into the container would satisfy the declaration
// while REMOVING the write bound: `max_write_mb` is enforced by the tmpfs size,
// and a host bind has none. The declared intent (/tmp writable inside the
// sandbox) is met by the tmpfs; the host directory stays outside.
func TestDeclaredScratchPathIsSatisfiedByTheBoundedTmpfs(t *testing.T) {
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Disk.MaxWriteMB = 200

	args, err := containerMountArgs(limits)
	if err != nil {
		t.Fatalf("containerMountArgs: %v", err)
	}
	for i, a := range args {
		if a != "-v" || i+1 >= len(args) {
			continue
		}
		if strings.HasPrefix(args[i+1], "/tmp:") || strings.HasPrefix(args[i+1], "/var/tmp:") {
			t.Fatalf("host scratch is bind-mounted (%q); that replaces the bounded tmpfs with an unbounded host directory", args[i+1])
		}
	}
	// Non-vacuous: the bound must still be present and sized from the policy.
	var tmpfs string
	for i, a := range args {
		if a == "--tmpfs" && i+1 < len(args) && strings.HasPrefix(args[i+1], "/tmp:") {
			tmpfs = args[i+1]
		}
	}
	if !strings.Contains(tmpfs, "size=200m") {
		t.Fatalf("/tmp tmpfs = %q, want size=200m from max_write_mb", tmpfs)
	}
}

// TestDeclaredPathAppearsExactlyOnce proves a genuine (non-scratch) declared
// path is still mounted, so removing the duplicate did not remove the feature.
func TestDeclaredPathAppearsExactlyOnce(t *testing.T) {
	limits := ResourceLimits{}
	limits.Disk.ReadWritePaths = []string{"/var/lib/sandbox-work"}
	limits.Disk.ReadOnlyPaths = []string{"/etc"}

	args, err := containerMountArgs(limits)
	if err != nil {
		t.Fatalf("containerMountArgs: %v", err)
	}
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-v /var/lib/sandbox-work:/var/lib/sandbox-work:rw") {
		t.Errorf("declared rw path not mounted: %v", args)
	}
	if !strings.Contains(joined, "-v /etc:/etc:ro") {
		t.Errorf("declared ro path not mounted: %v", args)
	}
	for _, d := range []string{"/var/lib/sandbox-work", "/etc"} {
		n := 0
		for _, got := range mountDestinations(args) {
			if got == d {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%q mounted %d times, want 1: %v", d, n, args)
		}
	}
}

// TestPathDeclaredBothWaysIsRefused: one destination, two contradictory modes.
// There is no order in which both mounts agree, so this is refused rather than
// resolved — resolving it would make the effective policy depend on argument
// order.
func TestPathDeclaredBothWaysIsRefused(t *testing.T) {
	limits := ResourceLimits{}
	limits.Disk.ReadWritePaths = []string{"/srv/data"}
	limits.Disk.ReadOnlyPaths = []string{"/srv/data/"}

	_, err := containerMountArgs(limits)
	if err == nil {
		t.Fatal("a path declared both writable and read-only was accepted")
	}
	if !strings.Contains(err.Error(), "both") {
		t.Errorf("error = %q; it should name the contradiction", err)
	}
}

// TestRelativeBindPathIsRefused: a relative path resolves against the server's
// working directory, so which host directory a command receives would depend on
// where the process was started.
func TestRelativeBindPathIsRefused(t *testing.T) {
	limits := ResourceLimits{}
	limits.Disk.ReadWritePaths = []string{"workspace"}

	_, err := containerMountArgs(limits)
	if err == nil {
		t.Fatal("a relative bind path was accepted")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("error = %q; it should say the path must be absolute", err)
	}
}

// TestMountPathSyntaxStillRefused keeps the pre-existing guard: ':' and ','
// corrupt the -v spec.
func TestMountPathSyntaxStillRefused(t *testing.T) {
	for _, bad := range []string{"/tmp:rw", "/a,b", ""} {
		limits := ResourceLimits{}
		limits.Disk.ReadWritePaths = []string{bad}
		if _, err := containerMountArgs(limits); err == nil {
			t.Errorf("mount path %q was accepted", bad)
		}
	}
}

// TestL2ActuallyRunsWithTheShippedDefaults drives the REAL execL2 path.
//
// Every assertion above is about argv shape, and argv shape was never the
// question — the question is whether a container starts. The defect survived
// because no test executed L2 with the default configuration; the shipped
// config could not start a container at all. This test skips (rather than
// passing) when no runtime is present, because a green result on a host that
// cannot run containers would be exactly the inference `sandbox_capability.go`
// exists to remove.
func TestL2ActuallyRunsWithTheShippedDefaults(t *testing.T) {
	runtime := "podman"
	if _, err := exec.LookPath(runtime); err != nil {
		runtime = "docker"
		if _, err := exec.LookPath(runtime); err != nil {
			t.Skip("no container runtime on this host; L2 capability is unmeasurable here")
		}
	}

	// Pre-flight the image. `isSetupError` classifies exit 125 as a setup
	// failure, and a duplicate mount destination ALSO exits 125 — the wrapped
	// error keeps only "exit status 125", so the two are indistinguishable from
	// the error alone. Deciding image availability up front is what lets the
	// execution below be a real assertion: if the image is present and the run
	// still fails 125, it is the mount defect.
	image := DefaultSandboxConfig().Image
	if image == "" {
		image = "alpine:latest"
	}
	if err := exec.Command(runtime, "image", "exists", image).Run(); err != nil {
		pull := exec.Command(runtime, "pull", "--quiet", image)
		if perr := pull.Run(); perr != nil {
			t.Skipf("image %q is not available on this host and could not be pulled: %v", image, perr)
		}
	}

	mgr := NewSandboxManager(DefaultSandboxConfig(), slog.Default())
	limits := DefaultSandboxConfig().GlobalDefaults
	// Keep only writable paths that exist on any host. The shipped default also
	// declares /workspace, which is not present everywhere; a missing bind
	// source is a real limitation of that default but a DIFFERENT one from this
	// test's subject. /tmp is retained because it is precisely the destination
	// that collided with the always-emitted tmpfs.
	var present []string
	for _, p := range limits.Disk.ReadWritePaths {
		if _, err := os.Stat(p); err == nil {
			present = append(present, p)
		}
	}
	if len(present) == 0 {
		t.Skip("none of the default writable paths exist on this host")
	}
	limits.Disk.ReadWritePaths = present
	limits.Disk.ReadOnlyPaths = nil
	// Keep the probe cheap: the point is that the container is CREATED, not
	// that an image was pulled inside a unit test.
	limits.Time.MaxSeconds = 60
	limits.Time.MaxWallSeconds = 90

	res, err := mgr.ExecuteWithFloor(context.Background(), "echo mount-plan-ok", LevelL2, limits, LevelL2)
	if err != nil {
		t.Fatalf("L2 execution failed with the image present: %v — exit 125 here means the "+
			"runtime refused the container, which is the duplicate-destination defect this "+
			"test exists to catch", err)
	}
	if res.ActualIsolationLvl != LevelL2 {
		t.Fatalf("executed at %s, not L2 — the requested level was silently downgraded", res.ActualIsolationLvl)
	}
	if !strings.Contains(res.Stdout, "mount-plan-ok") {
		t.Fatalf("stdout = %q, want the command's output", res.Stdout)
	}
}
