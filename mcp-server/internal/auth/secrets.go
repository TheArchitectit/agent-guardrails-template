package auth

import (
	"os"

	"github.com/thearchitectit/guardrail-mcp/internal/secretfile"
)

// SecretFileMode is the POSIX mode a credential secret file must have: owner
// read/write only, no group or world access (Spec 17 R17-06). Restrictive
// Windows ACL equivalents cannot be observed through os.FileMode, so the mode
// check is enforced on non-Windows hosts and documented for Windows.
//
// The rule itself lives in internal/secretfile so the credential-registry
// loader and the configuration loader cannot drift apart; this alias keeps the
// existing callers and tests reading the same name.
const SecretFileMode os.FileMode = secretfile.Mode

// permitsRestrictedOnly reports whether a file mode grants no group or world
// access. An owner-only file (0600, 0400, …) passes; any group/other bits
// (0644, 0640, 0660, …) fail.
func permitsRestrictedOnly(mode os.FileMode) bool {
	return secretfile.PermitsRestrictedOnly(mode)
}

// readSecretFile reads a read-only mounted secret file after rejecting
// non-regular files and, on POSIX hosts, any group/world-accessible mode. The
// caller treats any error as fail-closed.
func readSecretFile(path string) ([]byte, error) {
	return secretfile.Read(path)
}
