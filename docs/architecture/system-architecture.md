# System Architecture

How the Agent Guardrails Template fits together — the layers, data flow, and
deployment topology. All backend services are Go (`mcp-server/internal/`).

## High-level architecture

```mermaid
flowchart TB
    subgraph Client["Client Layer"]
        CLI["CLI / Scripts"]
        IDE["IDE Integration"]
        CI["CI/CD Pipeline"]
    end
    subgraph MCP["MCP Layer"]
        Server["MCP Server<br/>:8080 POST /mcp"]
        Validator["Input Validator"]
    end
    subgraph Tools["Tool Layer"]
        TeamTools["Team Tools"]
        GuardrailTools["Guardrail Tools"]
        AgentTools["Agent Tools"]
    end
    subgraph Backend["Backend Layer (Go)"]
        TeamMgr["Team Manager<br/>internal/team/"]
        RuleEngine["Rule Engine<br/>internal/validation/"]
        AuditLog["Audit Logger<br/>internal/audit/"]
    end
    subgraph Storage["Storage Layer"]
        DB[("PostgreSQL")]
        Cache[("Redis")]
        Files[".teams/ .guardrails/"]
    end
    CLI -->|JSON-RPC| Server
    IDE -->|JSON-RPC| Server
    CI -->|JSON-RPC| Server
    Server --> Validator
    Validator --> TeamTools & GuardrailTools & AgentTools
    TeamTools --> TeamMgr
    GuardrailTools --> RuleEngine
    AgentTools --> TeamMgr
    TeamMgr --> DB & Cache & Files
    RuleEngine --> Files
    AuditLog --> DB
```

| Layer | Responsibility | Components |
|-------|---------------|------------|
| Client | User interface | CLI, IDE extensions, CI/CD |
| MCP | Protocol handling | Stateless StreamableHTTP server, validation |
| Tools | Business logic | Team, guardrail, agent operations (35 tools) |
| Backend | Core services | Team manager, rule engine, audit logger |
| Storage | Persistence | PostgreSQL, Redis, JSON configs |

## Transport

As of v3.3.0 the server speaks **stateless StreamableHTTP**: a single
`POST /mcp` endpoint, no session IDs, each request independent. This replaces
the old SSE two-step flow and makes the server trivial to run behind a load
balancer. The web UI and REST API run on a separate port (`:8081`).

## Data flow

### Tool execution

```mermaid
sequenceDiagram
    participant Client
    participant Server as MCP Server
    participant Validator
    participant Tool as Tool Handler
    participant Backend
    participant Storage
    Client->>Server: POST /mcp (JSON-RPC)
    Server->>Validator: Validate input
    alt Invalid
        Validator-->>Client: 400 ValidationError
    else Valid
        Validator->>Tool: Route to handler
        Tool->>Backend: Execute logic
        Backend->>Storage: Read/write
        Storage-->>Backend: Data
        Backend-->>Tool: Result
        Tool-->>Client: 200 + result
    end
```

### Team assignment

```mermaid
sequenceDiagram
    participant Client
    participant Server as MCP Server
    participant TeamTool
    participant TeamMgr as Team Manager
    participant File as .teams/{project}.json
    Client->>Server: guardrail_team_assign
    Server->>TeamTool: Route
    TeamTool->>TeamMgr: assign_role(project, team, role, person)
    TeamMgr->>File: Read config
    alt Team full
        TeamMgr-->>Client: Error TEAM-005
    else Role occupied
        TeamMgr-->>Client: Error TEAM-004
    else OK
        TeamMgr->>File: Write config
        TeamMgr->>TeamMgr: Audit event
        TeamMgr-->>Client: Confirmed
    end
```

### Phase gate check

```mermaid
sequenceDiagram
    participant Client
    participant Server as MCP Server
    participant GateTool as Phase Gate Tool
    participant TeamMgr as Team Manager
    participant Rules as team-layout-rules.json
    participant Config as .teams/{project}.json
    Client->>Server: guardrail_phase_gate_check
    Server->>GateTool: Route
    GateTool->>Rules: Load gate requirements
    GateTool->>TeamMgr: Get phase status
    TeamMgr->>Config: Read team config
    TeamMgr-->>GateTool: Phase status
    GateTool->>GateTool: Compare requirements vs actual
    alt Met
        GateTool-->>Client: Approved
    else Not met
        GateTool-->>Client: Missing deliverables
    end
```

## Team structure

Teams are organized into 5 phases of the delivery lifecycle, each with a gate
before the next phase begins:

```mermaid
flowchart LR
    P1["Phase 1<br/>Strategy"] --> G1["Gate 1<br/>Architecture review"] --> P2["Phase 2<br/>Platform"]
    P2 --> G2["Gate 2<br/>Environment ready"] --> P3["Phase 3<br/>Build"]
    P3 --> G3["Gate 3<br/>Feature complete"] --> P4["Phase 4<br/>Validate"]
    P4 --> G4["Gate 4<br/>Security + QA sign-off"] --> P5["Phase 5<br/>Deliver"]
```

| Phase | Teams | Focus |
|-------|-------|-------|
| 1 — Strategy | Business & Product Strategy, Enterprise Architecture, GRC | Planning |
| 2 — Platform | Infrastructure, Platform Engineering, Data Governance | Foundation |
| 3 — Build | Core Feature Squad, Middleware & Integration | Implementation |
| 4 — Validate | Cybersecurity/AppSec, Quality Engineering | Hardening |
| 5 — Deliver | SRE, IT Operations & Support | Sustainment |

See [team-structure.md](../teams/team-structure.md) for role definitions and
[team-tools.md](../teams/team-tools.md) for the MCP tools that manage this.

## Deployment

### Single node

```mermaid
flowchart TB
    subgraph Host["Deployment host"]
        subgraph App["Container"]
            MCP["MCP Server<br/>:8080 /mcp, :8081 web"]
        end
        DB[("PostgreSQL :5432")]
        Cache[("Redis :6379")]
        Config[".teams/ .guardrails/"]
    end
    Client["Client"] -->|HTTP| MCP
    MCP --> DB & Cache & Config
```

By default ports bind to `127.0.0.1` via `BIND_ADDR`. Set `BIND_ADDR` to expose
on another interface (e.g. a tailnet IP). See the
[deployment guide](../../mcp-server/deployment-guide.md).

### Production (scaled)

```mermaid
flowchart TB
    Clients["CLI / IDE / CI-CD"] -->|HTTPS| LB["Load balancer"]
    LB --> S1["MCP Server 1"] & S2["MCP Server 2"] & S3["MCP Server 3"]
    S1 & S2 & S3 --> Postgres[("PostgreSQL")] & Prometheus["Prometheus"]
    S1 & S2 & S3 --> NFS["NFS (team configs, rules)"]
```

Stateless transport means any server can serve any request — no sticky sessions.

## Integration points

```mermaid
flowchart LR
    subgraph AGT["This repo"]
        MCP["MCP Server"]
    end
    Git["Git provider<br/>GitHub/GitLab"]
    CI["CI/CD<br/>Actions/Jenkins"]
    Claude["Claude Code"] & OpenCode["OpenCode"] & Cursor["Cursor"]
    MCP <-->|Git ops| Git
    MCP <-->|Pipeline triggers| CI
    Claude & OpenCode & Cursor <-->|MCP protocol| MCP
```

| Pattern | Use for |
|---------|---------|
| Synchronous request/response | Team assignments, phase gates, validation |
| Asynchronous webhook | Violation/halt notifications, audit export |
| Batch / file-based | Bulk rule updates |

## Security

```mermaid
flowchart TB
    L1["Input validation"] --> T1["Injection attacks"]
    L2["Auth (API key + JWT)"] --> T2["Unauthorized access"]
    L3["Audit logging"] --> T3["Data tampering"]
    L4["Encryption at rest + transit"] --> T4["Data breach"]
```

- **API key auth** on write/IDE endpoints; read-only web routes public.
- **JWT** session tokens for MCP clients, 15-min expiry, rotation configurable.
- **Secrets scanning** of document content (AWS keys, GitHub tokens, private keys, etc.).
- **Rate limiting** per API key (MCP 1000/min, IDE 500/min).
- Parameterized queries, regex timeouts (ReDoS protection), strict CSP headers.

See [security audits](../security/) for the detailed reviews.

## Configuration layout

```
agent-guardrails-template/
├── mcp-server/
│   ├── internal/           # Go: team, validation, audit, database, cache, mcp, web
│   └── cmd/server/         # entry point
├── .teams/                 # team configurations (per-project JSON)
├── .guardrails/            # rules, team-layout-rules, schemas
├── pi-extension/           # TypeScript extension for the pi agent
└── docs/                   # this documentation
```

The server reads `.env` for secrets (never committed). See
[python-to-go-migration.md](../mcp-server/python-to-go-migration.md) for the
history of the backend (Python team manager retired in v2.6.0).
