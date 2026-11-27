#!/usr/bin/env bash
# Authorized host-side acceptance harness. Never run this from an untrusted PR.
set -euo pipefail
umask 077
usage() {
  cat <<'EOF'
Usage: run_mcp_image_codex_harness.sh --receipt FILE [--run-codex]
Builds and probes a disposable rootless Podman MCP stack. The receipt is
written outside the scratch directory and contains no credentials. --run-codex
requires GUARDRAILS_SMOKE_CODEX_AUTH=1 (host authorization is not provisioned).
EOF
}
run_codex=0; receipt=
while (($#)); do
  case "$1" in
    --receipt) (($# > 1)) || { echo "error: --receipt needs a path" >&2; exit 2; }; receipt=$2; shift 2 ;;
    --run-codex) run_codex=1; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "error: unknown option $1" >&2; usage >&2; exit 2 ;;
  esac
done
[[ -n "$receipt" ]] || { echo "error: --receipt is required (prevents advertising a deleted receipt)" >&2; exit 2; }
[[ ! -e "$receipt" ]] || { echo "error: refusing to overwrite existing receipt: $receipt" >&2; exit 2; }
command -v podman >/dev/null || { echo "error: rootless Podman is required" >&2; exit 2; }
rootless=$(podman info --format '{{.Host.Security.Rootless}}' 2>/dev/null || true)
[[ "$rootless" == true ]] || { echo "error: Podman is not rootless; refusing rootful execution" >&2; exit 2; }
if ((run_codex)) && [[ ${GUARDRAILS_SMOKE_CODEX_AUTH:-} != 1 ]]; then
  echo "error: --run-codex requires GUARDRAILS_SMOKE_CODEX_AUTH=1" >&2; exit 2
fi
repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
receipt_abs=$(realpath -m "$receipt")
repo_abs=$(realpath -m "$repo")
[[ "$receipt_abs" != "$repo_abs"/* ]] || { echo "error: receipt must be outside the repository" >&2; exit 2; }
[[ -z "$(git -C "$repo" status --porcelain)" ]] || { echo "error: clean worktree required" >&2; exit 2; }
mkdir -p "$(dirname "$receipt")"
source_sha=$(git -C "$repo" rev-parse HEAD)
receipt_tmp=$(mktemp "$(dirname "$receipt")/.guardrails-receipt.XXXXXXXX")
scratch=$(mktemp -d "${TMPDIR:-/tmp}/guardrails-mcp.XXXXXXXX"); chmod 700 "$scratch"
prefix="guardrails-mcp-${source_sha:0:12}-$(date -u +%Y%m%d%H%M%S)-$$"
net="${prefix}-net"; pg="${prefix}-pg"; redis="${prefix}-redis"; app="${prefix}-app"
tag="localhost/${prefix}:runtime"; secrets="$scratch/secrets"; mkdir -p "$secrets"; chmod 700 "$secrets"
label="harness.owner=$prefix"
cleanup() {
  for c in "$app" "$redis" "$pg"; do podman rm -f "$c" >/dev/null 2>&1 || true; done
  podman network rm "$net" >/dev/null 2>&1 || true
  podman image rm "$tag" >/dev/null 2>&1 || true
  rm -rf -- "$scratch"
  rm -f -- "$receipt_tmp"
}
trap cleanup EXIT HUP INT TERM
python3 - "$secrets" <<'PY'
import pathlib, secrets, sys
d=pathlib.Path(sys.argv[1])
def put(name, value):
 p=d/name; p.write_text(value+"\n"); p.chmod(0o600)
def token(): return 'aA7' + secrets.token_urlsafe(48)
api,ide,db,red,jwt=(token() for _ in range(5))
put("mcp-key.txt",api)
put("postgres.env",f"POSTGRES_DB=guardrailsmoke\nPOSTGRES_USER=grsmoke\nPOSTGRES_PASSWORD={db}\nPGDATA=/var/lib/postgresql/data/pgdata")
put("redis.env",f"REDIS_PASSWORD={red}\nREDISCLI_AUTH={red}")
put("migration.env",f"MIGRATION_DATABASE_URL=postgres://grsmoke:{db}@pg:5432/guardrailsmoke?sslmode=disable")
put("server.env",f"""DEPLOYMENT_PROFILE=local
DEPLOYMENT_BIND_ATTESTATION=isolated-rootless-loopback
MCP_BIND_HOST=0.0.0.0
MCP_PORT=8080
WEB_ENABLED=false
DB_HOST=pg
DB_PORT=5432
DB_NAME=guardrailsmoke
DB_USER=grsmoke
DB_PASSWORD={db}
DB_SSLMODE=disable
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD={red}
MCP_API_KEY={api}
IDE_API_KEY={ide}
JWT_SECRET={jwt}
JWT_ROTATION_HOURS=168h
ENABLE_METRICS=false
VISION_ENABLED=false
LOG_LEVEL=info""")
PY
podman build --pull=missing --label "$label" --build-arg VERSION=ci --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" --build-arg GIT_COMMIT="$source_sha" \
  -f "$repo/mcp-server/deploy/Dockerfile" -t "$tag" "$repo/mcp-server" >/dev/null
podman network create --internal --label "$label" "$net" >/dev/null
podman run -d --name "$pg" --label "$label" --network "$net" --network-alias pg --env-file "$secrets/postgres.env" \
  --tmpfs /var/lib/postgresql/data:rw,noexec,nosuid,size=768m,mode=1777 --memory 1536m --cpus 1.0 --health-cmd 'pg_isready -U grsmoke -d guardrailsmoke' \
  --health-interval 2s --health-timeout 3s --health-retries 30 docker.io/library/postgres:16-alpine@sha256:721873c34ceb9f8d8fc265984940dc982404c105f19ad51be9fdc5970a6080ea >/dev/null
podman run -d --name "$redis" --label "$label" --network "$net" --network-alias redis --env-file "$secrets/redis.env" \
  --tmpfs /data:rw,noexec,nosuid,size=64m --memory 256m --cpus 0.5 --health-cmd 'redis-cli ping' \
  --health-interval 2s --health-timeout 3s --health-retries 30 \
  docker.io/library/redis:7-alpine@sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499 \
  sh -c 'printf "requirepass %s\\n" "$REDIS_PASSWORD" >/tmp/redis.conf && exec redis-server /tmp/redis.conf' >/dev/null
wait_health() {
  local c=$1 i
  for i in {1..60}; do
    [[ "$(podman inspect -f '{{.State.Health.Status}}' "$c" 2>/dev/null || true)" == healthy ]] && return
    sleep 2
  done
  podman logs "$c" >&2 || true; echo "error: $c not healthy" >&2; exit 1
}
wait_health "$pg"; wait_health "$redis"
podman run --rm --name "${prefix}-migrate" --label "$label" --network "$net" --env-file "$secrets/migration.env" \
  --entrypoint /run-migrations --workdir /app --read-only --tmpfs /tmp:rw,noexec,nosuid,size=16m "$tag" >/dev/null
podman run -d --name "$app" --label "$label" --network "$net" --env-file "$secrets/server.env" \
  --read-only --tmpfs /tmp:rw,noexec,nosuid,size=64m --tmpfs /app/vision_data:rw,noexec,nosuid,size=64m \
  --tmpfs /app/screenshots:rw,noexec,nosuid,size=64m --memory 2g --cpus 2.0 --security-opt no-new-privileges --cap-drop ALL \
  --health-cmd '/server --health-check' --health-interval 10s --health-timeout 5s --health-retries 20 \
  -p 127.0.0.1::8080 "$tag" >/dev/null
port=$(podman port "$app" 8080/tcp | sed -n 's/.*:\([0-9][0-9]*\)$/\1/p')
[[ "$port" =~ ^[0-9]+$ ]] || { echo "error: no loopback port" >&2; exit 1; }
for i in {1..60}; do
  podman exec "$app" /server --health-check >/dev/null 2>&1 && break
  ((i == 60)) && { podman logs "$app" >&2 || true; exit 1; }; sleep 2
done
key=$(<"$secrets/mcp-key.txt")
python3 - "$secrets/mcp-key.txt" "$port" <<'PY'
import json, pathlib, sys, urllib.request
key=pathlib.Path(sys.argv[1]).read_text().strip()
url="http://127.0.0.1:"+sys.argv[2]+"/mcp"
def call(method, params, ident):
 body=json.dumps({"jsonrpc":"2.0","id":ident,"method":method,"params":params}).encode()
 req=urllib.request.Request(url, data=body, method="POST", headers={"Authorization":"Bearer "+key,"Content-Type":"application/json","Accept":"application/json, text/event-stream"})
 with urllib.request.urlopen(req, timeout=10) as response:
  raw=response.read().decode()
  if "text/event-stream" in response.headers.get("Content-Type",""):
   raw=next(line[5:].strip() for line in raw.splitlines() if line.startswith("data:"))
  return response.status, json.loads(raw)
status, init=call("initialize",{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"ci-probe","version":"1"}},1)
if status != 200 or "result" not in init or not init["result"].get("protocolVersion"):
 raise SystemExit("MCP initialize failed or did not negotiate a protocol version")
status, listing=call("tools/list",{},2)
if status != 200 or "result" not in listing:
 raise SystemExit("MCP tools/list failed")
if not any(t.get("name")=="guardrail_advisor_list" for t in listing["result"]["tools"]):
 raise SystemExit("MCP advisor tool is missing")
status, called=call("tools/call",{"name":"guardrail_advisor_list","arguments":{}},3)
if status != 200 or "result" not in called or called["result"].get("isError"):
 raise SystemExit("MCP advisor tool call failed")
payload=next(json.loads(c["text"]) for c in called["result"].get("content",[]) if c.get("type")=="text")
if payload.get("count") != 9:
 raise SystemExit("MCP advisor tool returned an unexpected count")
print("MCP protocol OK: initialize, tools/list, guardrail_advisor_list=9")
PY
image_id=$(podman image inspect "$tag" -f '{{.Id}}')
manifest_digest=$(podman image inspect "$tag" -f '{{.Digest}}' 2>/dev/null || true)
[[ -n "$manifest_digest" ]] || { echo "error: local engine did not expose an immutable manifest digest; refusing unverifiable receipt" >&2; exit 1; }
{
  echo "source_sha=$source_sha"
  echo "image_tag=$tag"
  echo "image_id=$image_id"
  echo "image_manifest_digest=$manifest_digest"
  echo "network=$net"
  echo "containers=$pg,$redis,$app"
  echo "endpoint=http://127.0.0.1:$port/mcp"
  echo "mcp_protocol=passed"
  echo "codex=not-run"
} > "$receipt_tmp"
if ((run_codex)); then
  GUARDRAILS_SMOKE_MCP_KEY="$key" GUARDRAILS_SMOKE_MCP_URL="http://127.0.0.1:$port/mcp" "$repo/scripts/codex_mcp_runtime_smoke.sh"
  sed -i 's/^codex=.*/codex=passed/' "$receipt_tmp"
fi
ln -- "$receipt_tmp" "$receipt" || { echo "error: receipt destination already exists or cannot be written" >&2; exit 1; }
rm -f -- "$receipt_tmp"
echo "PASS: receipt persisted at $receipt"
