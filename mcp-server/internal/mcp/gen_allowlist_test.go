package mcp

import (
	"encoding/json"
	"os"
	"testing"
)

// TestGenerateDriftAllowlist is a maintenance helper. It is skipped unless
// GR_GEN_DRIFT_ALLOWLIST=1, and it writes the current measured drift to stdout
// as the body of testdata/tool_contract_drift_allowlist.json. It never runs as
// part of the normal gate.
func TestGenerateDriftAllowlist(t *testing.T) {
	if os.Getenv("GR_GEN_DRIFT_ALLOWLIST") != "1" {
		t.Skip("generation helper; set GR_GEN_DRIFT_ALLOWLIST=1 to run")
	}
	_, files := parsePackageSource(t)
	schemas := enumerateRegisteredTools(files)
	reads := handlerReads(files, schemas)
	drift := computeDrift(schemas, reads)
	out, _ := json.MarshalIndent(drift, "", "  ")
	t.Logf("DRIFTJSON=%s", string(out))
}
