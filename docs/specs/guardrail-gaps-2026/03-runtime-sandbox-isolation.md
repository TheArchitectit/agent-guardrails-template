# OpenSpec: Runtime Sandbox Isolation

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
