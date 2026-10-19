# OpenSpec: Runtime Sandbox Isolation

> **Status 2026-10-19 (fourth pass) — PARTIAL, and the sandbox reported capability it never measured — a green control over a path that could not run once.** The readiness probe (`probeSandbox`) and the readiness observer both answered a *declaration* question and published it as a *measurement*. The probe ran `LookPath` over `unshare`/`podman`/`docker` and returned `active` when any was found; the observer read `cfg.Enabled` and returned `reachable`. Measured on the UCS03 host: `unshare` is in `/usr/bin` and cannot create a namespace at all (`unshare: write failed /proc/self/uid_map: Operation not permitted`, exit 1), while every shipped tool policy is **L1** and `ExecuteFor` **floors at L1** — so every policy-driven execution was refused (`L1 setup failed and falling back would run at L0, below the required floor L1`) on the same process that reported `active`. Since `active` is the **only** status a measured compliance report counts as covered, three requirements scored the sandbox control as **COVERED** on a host where it could not run once: `eu_ai_act art_9`, `eu_ai_act art_15`, `nist_rmf manage`. **The inference was the defect, not the host:** a namespace is a kernel grant, not a file on PATH, and a runtime binary is not a reachable daemon. Capability is now **measured** — the probe execs the same argv `execL1` builds and records what happened, and the container runtime must answer `info` — and one decision feeds the probe, the readiness observer and the report, so three surfaces cannot disagree about one process. The probe judges the **strongest** declared policy (not the first, not the weakest), because a deployment with one L2 policy is only as capable as L2. Present-but-unusable is **`partial`, never `active`**. The decision lives in its own file, `sandbox_capability.go`, **declared** in the `sandbox-isolation` component's `source_paths` — while this logic sat in a shared probe file no component digest covered, the published compliance verdict could have changed without the manifest noticing. Manifest `1.0.2 → 1.0.3`, digest recomputed from the repo's own `controlplane.SourceDigest` output. **Evidence:** `sandbox_capability_test.go`, 8 tests over the real probe/manager, **red-on-drift proven three ways** (restore the `LookPath` inference → 4 tests fail, including the exact host premise; select the weakest policy → the strongest-policy test fails; make `Supports` fail-open → 4 tests fail). Named limit: the L2 measurement proves the daemon answers, not that a container boots — pulling an image inside a report would cost seconds and could fail on a registry outage unrelated to this host. Spec 03 remains PARTIAL: L1 still enforces no path set, hot reload is unimplemented, and the Sec. 5.x harnesses are unwritten.
>
> **Status 2026-10-19 (third pass) — PARTIAL, and the violation DETECTOR was fabricating breaches.** With the two switches now binding (second pass), the remaining output-derived control was the network-violation detector, and it was a false-positive generator of the first order: it found "hosts" with a bare `//` and a bare `to:` alternation, so a Go source comment (`main.go:12: // increment the counter`), a UNC path (`//server/share/file.txt`) and ordinary prose (`next step to: nothing-here`) each produced `network access to non-allowed host: <word>`. Because any non-empty violation set becomes `ErrSandboxViolation`, a **successful benign command was reported to the caller as a security breach** — measured through the real manager with the shipped `git` policy (network enabled, hosts `github.com`+`gitlab.com`). The slash half was the worse of the two: `//` is a comment in every C-family language and a UNC root on Windows, so the detector fired on the most common line of output a build produces. Both alternations are removed; a candidate now requires an explicit scheme or an explicit connection/resolution phrase **and** must look like a hostname, and the branch is gated on `egressPermitted` — the same decision the execution path uses, so a `network.enabled: false` run (which had `--network=none`) is not scanned for egress violations it could not commit. Red-on-drift proven four ways. Spec 03 remains PARTIAL: L1 still enforces no path set, hot reload is unimplemented, and the Sec. 5.x harnesses are unwritten.
>
> **Status 2026-10-19 (second pass) — PARTIAL, wired, and two more declared switches now bind.** The previous pass made every numeric limit bind and made the documented config file load at startup. This pass closed the two *boolean* switches that were still declaration-only: **`network.enabled: false` had zero readers on any execution path** — `execL2` decided egress from `len(allowed_hosts) > 0` alone, so the exact shape Sec. 3.2 documents (`enabled: false` above a whitelist) opened a provisioned egress bridge and a CONNECT proxy to those hosts; and **`sandbox.enabled: false` was read by exactly one place, the readiness probe**, so readiness reported the manager as not enabled while every execution still ran through it. Both now bind at the manager, and a process-wide `network_mode` can no longer hand a bridge to a policy that declared network disabled. Residual gaps are named in Sec. 8 — the largest being that L1 still enforces no path set. Sec. 4.3 fallback text remains wrong — see the amendment note.
>
> **Status 2026-10-19 (first pass) — PARTIAL, the declared limits bind, and the documented config file is now read.** Both tools exist, are registered, dispatched, classified and audited, and the declared per-tool policies are applied at execution (`ExecuteFor`). This pass fixed a class of defect the previous status missed: **three limits declared in the shipped default config were consumed by nothing.** `swap_allowed: false`, `max_wall_seconds` and `max_write_mb` were read by no code path, so a reviewer saw three controls and the sandbox applied none of them — and `ResourceLimits{}` (the engine's fallback for a tool with no policy) produced a zero deadline, killing every command before it started and reporting the result as `ErrSandboxViolation`. All four are fixed at the cause. **The configuration surface is now real:** `DefaultSandboxConfig()` had been the only `SandboxConfig` in the tree — no file, no environment variable, no caller — so an operator who wrote the Sec. 3.2/3.3 YAML changed nothing. `SANDBOX_CONFIG_FILE` now loads it at startup, the documented `sandbox_policies:` spelling and the policy-level path placement both load, an unknown key is an error, and a configured-but-unusable file stops startup.

**Gap:** Important — Pre-execution validation only; no OS-level isolation
**Priority:** 🟡 Important (Phase 2)
**Depends on:** None
**Blocks:** None

---

## 1. Problem Statement

The agent-guardrails-template validates tool calls **before execution** (checking command patterns, file paths, git operations) but does not enforce **runtime isolation** — the ability to contain a tool's execution within a restricted environment so it cannot escape or access unauthorized resources.

**Current gap:**
- `guardrail_validate_bash` checks command patterns but cannot prevent command escape
- No OS-level sandboxing (containers, namespaces, seccomp)
- A validated command could still access arbitrary files or network resources
- No equivalent to NeMo's execution rails or container-based sandboxing

---

## 2. Proposed Solution

Add **Runtime Sandbox Execution** that wraps tool calls in isolated environments with configurable resource limits and access controls.

### 2.1 Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Tool Call   │────▶│ Sandbox Manager   │────▶│  Isolated Exec  │
│  (validated) │     │ (namespace/cgroup)│     │  (restricted)   │
└─────────────┘     └──────────────────┘     └─────────────────┘
                           │                         │
                    ┌──────▼──────┐           ┌──────▼──────┐
                    │  Audit Log  │           │  Resource   │
                    │  + Metrics  │           │  Monitor    │
                    └─────────────┘           └─────────────┘
```

### 2.2 Isolation Levels

| Level | Method | Protection | Performance |
|-------|--------|------------|-------------|
| **L0: None** | Direct execution | Pre-validation only | 0ms overhead |
| **L1: Namespace** | Linux namespaces (pid, net, mnt, uts) | Process/network/mount isolation | ~5ms |
| **L2: Container** | Docker/podman rootless container | Full filesystem + resource isolation | ~50ms |
| **L3: VM** | Firecracker/microVM | Hardware-level isolation | ~200ms |

Default: **L1** for most tool calls, **L2** for untrusted operations, **L3** not required for coding agents.

---

## 3. Technical Requirements

### 3.1 New MCP Tools

```go
// guardrail_sandbox_execute — runs a command in a sandboxed environment
// Input:  { command: string, sandbox_level: enum("L0","L1","L2"), resource_limits?: ResourceLimits, allowed_paths?: []string }
// Output: { exit_code: int, stdout: string, stderr: string, sandbox_violations: []string }
guardrail_sandbox_execute(command, sandbox_level, resource_limits?, allowed_paths?) → SandboxResult

// guardrail_sandbox_config — configures sandbox policies per tool type
// Input:  { tool_type: string, sandbox_level: string, resource_limits: ResourceLimits }
// Output: { configured: bool }
guardrail_sandbox_config(tool_type, sandbox_level, resource_limits) → ConfigResult
```

### 3.2 Resource Limits

```yaml
resource_limits:
  cpu:
    max_percent: 50          # percentage of available CPU
    max_cores: 2             # hard cap
  memory:
    max_mb: 1024             # maximum RAM
    swap_allowed: false
  disk:
    max_write_mb: 500        # maximum disk writes
    read_only_paths:         # paths that cannot be written
      - "/etc"
      - "/usr"
      - "~/.ssh"
    read_write_paths:        # paths that CAN be written
      - "/tmp"
      - "/workspace"
  network:
    enabled: false           # no network by default
    allowed_hosts: []        # whitelist if network needed
  time:
    max_seconds: 300         # execution timeout (5 min default)
    max_wall_seconds: 600    # wall clock timeout
```

### 3.3 Sandbox Policies per Tool Type

```yaml
sandbox_policies:
  bash:
    default_level: "L1"
    resource_limits:
      cpu:
        max_percent: 50
      memory:
        max_mb: 512
      network:
        enabled: false
    read_write_paths:
      - "/tmp"
      - "/workspace"
    read_only_paths:
      - "/etc"
      - "/usr"
  git:
    default_level: "L1"
    resource_limits:
      cpu:
        max_percent: 25
      memory:
        max_mb: 256
      network:
        enabled: true
        allowed_hosts: ["github.com", "gitlab.com"]
    read_write_paths:
      - "/workspace/.git"
      - "/workspace"
  file_edit:
    default_level: "L0"  # file edits use existing validation
    resource_limits:
      cpu:
        max_percent: 10
      memory:
        max_mb: 128
```

### 3.4 Violation Detection

The sandbox monitors for:
- **Path traversal:** Attempts to access files outside allowed paths
- **Network violations:** Connections to non-whitelisted hosts
- **Resource exhaustion:** CPU/memory/time limits exceeded
- **Privilege escalation:** Attempts to use setuid, change users, or modify system files
- **Fork bombs:** Excessive process creation

---

## 4. Implementation Notes

### 4.1 Linux Namespace Approach (L1)

```go
// Using unshare(2) for namespace isolation
func sandboxExecL1(command string, limits ResourceLimits) (*SandboxResult, error) {
    cmd := exec.Command("unshare",
        "--pid",     // new PID namespace
        "--mount",   // new mount namespace
        "--net",     // new network namespace
        "--uts",     // new UTS namespace
        "sh", "-c", command,
    )
    // Apply cgroup limits
    // Set read-only mounts
    // Execute and capture output
}
```

### 4.2 Container Approach (L2)

```go
// Using Docker/podman rootless containers
func sandboxExecL2(command string, limits ResourceLimits) (*SandboxResult, error) {
    dockerArgs := []string{
        "run", "--rm", "--network=none",
        "--memory=" + strconv.Itoa(limits.Memory.MaxMB) + "m",
        "--cpus=" + fmt.Sprintf("%.1f", float64(limits.CPU.MaxPercent)/100),
        "--read-only",
        "-v", "/workspace:/workspace:rw",
        "guardrails-sandbox:latest",
        "sh", "-c", command,
    }
    cmd := exec.Command("docker", dockerArgs...)
    // Execute and capture output
}
```

### 4.3 Fallback Strategy

- If namespaces are unavailable → fall back to pre-validation only (L0)
- If Docker is unavailable → fall back to L1 namespaces
- Always log the actual isolation level used

---

## 5. Testing Criteria

### 5.1 Unit Tests
- [ ] Namespace isolation correctly blocks file access outside allowed paths
- [ ] Resource limits are enforced (CPU, memory, time)
- [ ] Network isolation prevents external connections
- [ ] Violation detection catches path traversal attempts

### 5.2 Integration Tests
- [ ] Tool call → sandbox → execution → audit log
- [ ] Sandbox configuration hot-reload
- [ ] Fallback: L2 unavailable → L1 used
- [ ] Fallback: L1 unavailable → L0 with warning

### 5.3 Security Tests
- [ ] Container escape attempts are blocked
- [ ] Fork bomb is contained
- [ ] Network exfiltration is prevented
- [ ] Privilege escalation is prevented

---

## 6. Dependencies

- **Internal:** Existing MCP server, tool validation pipeline
- **External:** Linux kernel (namespaces), Docker/podman (optional for L2)
- **System:** Root or unprivileged user namespaces enabled

---

## 7. References

- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html) — isolation primitives
- [Docker Security](https://docs.docker.com/engine/security/) — container isolation
- [NeMo Execution Rails](https://github.com/NVIDIA/NeMo-Guardrails) — tool-use sandboxing concept
- [Firecracker](https://firecracker-microvm.github.io/) — microVM isolation

---

## 8. Implementation Status (reconciled 2026-10-18)

Evidence: `docs/specs/guardrail-gaps-2026/STATUS.md`; tests
`mcp-server/internal/mcp/tools_sandbox_test.go` and
`mcp-server/internal/guardrails/sandbox_policy_test.go`.

| Sec. | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 3.1 | `guardrail_sandbox_execute` | **Implemented 2026-10-18** | registered in `toolList()`, routed in `dispatchToolCall`, classified `mutate` + audit-before-effect; `tools_sandbox.go` |
| 3.1 | `guardrail_sandbox_config` | **Implemented 2026-10-18** | registered, routed, classified `admin` + audit-before-effect; whole config validated before the mutation is applied |
| 3.1 | Tool cannot become a host-execution path | **Implemented 2026-10-18** | effective L0 refused outright even when a policy declares it; test asserts the marker file is never created |
| 3.2 | Per-request `resource_limits` | **Implemented, narrowing only** | a wider value is refused naming the field, never clamped; `NarrowLimits` |
| 3.3 | Per-tool policies **applied** | **Implemented 2026-10-18** | `ExecuteFor`/`ExecuteForLimits` resolve level + limits from `tool_policies`; before this the table had **no consumer at all** |
| 3.3 | Per-tool policy **validated** | **Implemented 2026-10-18** | empty or unknown `DefaultLevel` is a config error; previously `L9` validated clean and resolved to L0 at use time |
| 2.x | L0 / L1 (unshare) / L2 (podman→docker) | Implemented | `sandbox.go` exec funcs (`--read-only`, `--cap-drop=ALL`, `no-new-privileges`) |
| 2.x | L3 Firecracker | Not implemented | the spec itself marks this optional |
| 3.2 | `resource_limits` shape | Implemented field-for-field | `sandbox_config.go` |
| 3.3 | Policy key name | **Both accepted 2026-10-19** | the Go type is `tool_policies`; the spec documents `sandbox_policies`. The file loader accepts **either** and refuses a file declaring both, so the documented spelling now loads |
| 3.3 | Spec's own YAML shape loads | **Implemented 2026-10-19** | Sec. 3.3 puts `read_write_paths`/`read_only_paths` at the **policy** level, beside `resource_limits`; a `resource_limits`-only decoder refused that as an unknown field. Both placements load; declaring one set in both places is refused (no correct merge order). `TestDocumentedSandboxShapeLoads` drives a verbatim copy of the documented YAML |
| 4.x | Network isolation | Implemented, **stronger than spec** | `--network=none` fail-closed plus CONNECT-filtering proxy and provisioned bridge network (`sandbox_network.go`) |
| 3.4 | Network-violation **detection** | **Repaired 2026-10-19** | the detector found hosts with a bare `//` and a bare `to:` alternation, so ordinary output — a source comment (`main.go:12: // increment`), a UNC path (`//server/share/f`), documentation prose (`next step to: nothing-here`) — was reported as `network access to non-allowed host: <word>`, and any non-empty violation set becomes `ErrSandboxViolation`. A successful benign command was handed to the caller as a security breach, which is the fastest way to teach an operator to disable the detector. Both alternations removed; a candidate now needs an explicit scheme (`https://`, `ssh://`, `git://`, …) or an explicit connection/resolution phrase, **and** must look like a hostname (contains a dot, hostname alphabet) — so `docs`, `server` and `localhost` are no longer hosts. The branch is also gated on `egressPermitted`, the same decision the execution path uses: with `network.enabled: false` the run had `--network=none`, so a "network access" verdict about it would describe a control that was never reachable. Pinned by `sandbox_network_falsepositive_test.go` (8 benign shapes, 4 positive controls, allowlist negative control, the egress gate, and 12 extractor shapes); red-on-drift proven four ways. Named limit: protocol-relative URLs (`//host/path`, an HTML idiom) are no longer detected — they are indistinguishable from a comment or a UNC path in free-form output |
| 4.x | Resource-exhaustion detection | Implemented | exit 137 / deadline mapping |
| 4.x | Path-traversal **detection** | **Not implemented** | only mount-path syntax validation; L2 relies on the runtime's mounts, L1 does not enforce a path set at all |
| 3.2 | `max_wall_seconds` **applied** | **Implemented 2026-10-19** | `effectiveDeadline` binds both time fields and takes the tighter one; before this only `max_seconds` was ever read, so a config declaring a 60 s wall clock had no 60 s bound at all |
| 3.2 | `swap_allowed: false` **applied** | **Implemented 2026-10-19** | `containerResourceArgs` emits `--memory-swap` equal to `--memory` when swap is disallowed; without it the memory ceiling was advisory and the container swapped past it |
| 3.2 | `max_write_mb` **applied** | **Implemented 2026-10-19** | the writable `/tmp` tmpfs is sized from the declared bound instead of a hardcoded 64 MB |
| 3.2 | Absent limits use the documented default | **Implemented 2026-10-19** | `ResourceLimits{}` used to mean a **zero deadline**: every command was killed before it ran and reported as `ErrSandboxViolation`. `effectiveDeadline` now falls back to `DefaultTimeoutSeconds`, and `Engine.Evaluate` uses `SandboxManager.DefaultLimits()` |
| 4.x | Fork-bomb **prevention** | **Implemented at L2 2026-10-19** | `--pids-limit` from the declared `max_processes` — a cgroup bound scoped to the container |
| 4.x | Fork-bomb **detection** | **Implemented, gated on an enforced bound** | `detectViolations` reports the class only where a process bound is actually in force (`processLimitEnforcedFor` → L2), so it cannot describe a control that does not exist |
| 4.x | Privilege-escalation detection | **Deliberately not a text match** | prevention at the boundary (`--cap-drop=ALL`, `no-new-privileges`, user namespace). Grepping stderr for "Operation not permitted" would flag ordinary permission errors — the same false-positive generator Law 2 became in gap-spec 04. An assertion pins that an ordinary permission error is **not** reported as a violation |
| 5.1 | Unit tests for isolation behaviour | **Partial, improved 2026-10-19** | policy resolution, level resolution, override narrowing, L0 refusal, fail-closed-without-engine, and now every declared limit bound to a real value (`sandbox_limits_test.go`); "namespace blocks file access outside allowed paths" is **not** covered because L1 does not implement a path set |
| 5.2 | Config hot-reload | **File source IMPLEMENTED 2026-10-19; hot reload NOT implemented** | `SANDBOX_CONFIG_FILE` → `LoadSandboxConfigFromYAML` at startup; a configured-but-unusable file **stops startup**. The file is read once — an edit still requires a restart, so the Sec. 5.2 *reload* checkbox stays open. Named along with the trigger question: a reload of a policy file changes what is *permitted*, which is a governance decision (who may raise isolation, who may lower it), not a plumbing gap |
| 3.x | Operator can supply the policy at all | **Implemented 2026-10-19** | before this, `DefaultSandboxConfig()` was the **only** `SandboxConfig` in the tree: no file, no env var, no caller, so the documented YAML described a path nothing read. Now the loader is wired at startup and the loaded table is what `ExecuteForLimits` resolves |
| 3.x | Sandbox image validation | **Fixed 2026-10-19 (latent defect)** | `Validate()` rejected **every** valid OCI reference — `alpine:latest`, `ghcr.io/org/img:v1`, `alpine@sha256:…` — because the ':' separator was treated as an argv split. Invisible while no file could set `image:`; the loader would have made a correct value stop startup. `validateImageRef` now refuses only what changes the command line (whitespace, leading `-`, shell metacharacters) |
| 3.2 | `network.enabled: false` **applied** | **Implemented 2026-10-19** | the switch had **zero readers on any execution path**. `execL2` decided egress from `len(AllowedHosts) > 0` alone, so the exact shape Sec. 3.2 documents — `enabled: false` above `allowed_hosts:` — got a provisioned egress bridge and a CONNECT proxy to the listed hosts: the field that says "no" was ignored and the field that says "these hosts, if permitted" was read as permission. The switch now binds (`egressPermitted`, `containerNetworkPlan`), and a process-wide `network_mode` no longer hands a bridge to a policy that declared network disabled |
| 3.x | `sandbox.enabled: false` **applied** | **Implemented 2026-10-19** | `Enabled` was read by exactly one place in the tree — the readiness probe — so readiness reported "sandbox manager is not enabled" while every execution still went through `Execute`. A deployment whose operator turned the sandbox off got a report saying so and commands that ran anyway. The manager now refuses to execute when disabled (`ErrSandboxViolation`, naming the cause), so the report and the behaviour cannot disagree |
| 5.3 | Container escape / fork bomb / exfiltration | **Not implemented** | no harness exists; this is the named blocker for the Sec. 5.x checkboxes |

**Sec. 4.3 fallback is wrong and should be rewritten.** Sec. 4.3 specifies
unconditional L2→L1→L0 fallback. The implementation deliberately downgrades
**only** on genuine setup errors (`isSetupError`); a command denied while
running under isolation is treated as a security breach and reported. That is
the correct fail-closed behaviour — silently re-running a denied command at
lower isolation would execute it on the host with none. The spec text should
be amended to match the code, not the reverse.

**Net position after 2026-10-19.** The library is reachable, the declared
policies bind, and **every limit the config declares now has a consumer**. That
last point was the defect this pass existed to fix: `swap_allowed`,
`max_wall_seconds` and `max_write_mb` were declared in `DefaultSandboxConfig()`
and read by nothing, which is worse than an absent control — a reviewer sees a
bound in the file and the sandbox applies none of it. Two defects surfaced only
while proving the fix:

1. **`ResourceLimits{}` was a zero deadline.** `Engine.Evaluate` passed the zero
   value for any tool with no declared policy, so every command was killed
   before it started and the caller received `ErrSandboxViolation` — a security
   verdict about work that never ran. That teaches operators to ignore the
   violation signal.
2. **The first mechanism I used was itself a no-op.** A `ulimit -u` shell prefix
   silently failed under `dash` (`sh` on Debian has no `-u`), so the
   "fix" ran the command unbounded while printing nothing. `prlimit` replaced
   it. Then measurement killed the `--nproc` form too: `RLIMIT_NPROC` is counted
   **per real UID across the whole host**, not per process tree, so
   `--nproc=128` made every fork fail on an account already running 837
   processes. That is an outage shaped like a control. The process bound is now
   `--pids-limit` at L2, which is a cgroup bound scoped to the container, and
   detection is gated on that bound actually being in force.

What remains is not reachability: it is detection breadth (path traversal), a
config *reload*, and the Sec. 5.x harnesses. Each is named above rather than
stubbed.

**Configuration file, 2026-10-19.** The loader landed because the absence was
not theoretical. `DefaultSandboxConfig()` was the only `SandboxConfig` value in
the tree; `LoadInjectionConfigFromYAML`'s Spec 01 seam had exactly the same
shape one spec over, and both artifacts described a configuration path nothing
read. Two defects surfaced only while wiring it:

1. **The spec's own YAML did not load.** Sec. 3.3 places `read_write_paths` and
   `read_only_paths` at the **policy** level, beside `resource_limits`. A
   decoder built from the Go type — where those live under `resource_limits.disk`
   — refused the documented shape as an unknown field. Both placements are now
   accepted, and declaring one set in both places is refused because two
   spellings of one path set has no correct merge order. `TestDocumentedSandboxShapeLoads`
   drives a verbatim copy of the spec's own example.
2. **`Validate()` refused every valid image reference.** The check was
   `strings.ContainsAny(image, " \t\n:;")`, which rejects `alpine:latest`,
   `alpine:3.19`, `ghcr.io/org/img:v1` and `alpine@sha256:…` — every reference a
   deployment would actually use — while accepting only the empty string. That
   was invisible for as long as no file could set `image:`; the moment one can,
   a correct value would have stopped startup. The ':' in a tag and in a
   registry port is part of the reference, not a second argument.

Two rules keep the file from becoming a false assurance. **Unknown keys are an
error** (`KnownFields`), so a typo cannot leave a bound at its zero value and
load clean. And every boolean is a **pointer** in the YAML mirror, so an omitted
switch inherits the shipped default while an explicit `false` wins — decoding
straight into the Go struct collapses both, which would let a file naming only
`max_mb` silently disable `fallback_enabled`.

**Still-open decisions:**

1. Path-traversal detection: enforce a path set at L1, or amend Sec. 4.x to
   scope detection to L2 mounts only. (Unchanged: L1 opens a mount namespace
   but mounts nothing, so there is no path set to violate.)
2. ~~The two missing violation classes.~~ **Resolved 2026-10-19:** fork bombs
   are prevented by `--pids-limit` and detected only where that bound exists;
   privilege escalation is prevention at the boundary, not a text match, and
   the reason is recorded in the table above.
3. ~~A YAML loader for policies.~~ **Resolved 2026-10-19:** `SANDBOX_CONFIG_FILE`
   loads the Sec. 3.2/3.3 file at startup; `sandbox_policies:` and
   `tool_policies:` are both accepted. Hot *reload* (an edit picked up without a
   restart) is still unimplemented and is the remaining Sec. 5.2 item.
4. Sec. 5.x security criteria need a defined harness (container escape, fork
   bomb) before they can be checked at all. The fork-bomb half now has a
   mechanism to test; the harness itself is still unwritten.
5. Whether `max_processes` should default non-zero for L0/L1 callers at all.
   It currently defaults to 256 in `DefaultSandboxConfig()`, but that bound only
   binds at L2 — at L0/L1 there is no per-tree mechanism that does not also
   break every other process under the same UID. Recording the limit as
   L2-only, rather than silently pretending it applies everywhere, is the
   honest disposition; making it real at L1 needs a cgroup-per-command
   supervisor, which is a design decision, not a bug fix.
