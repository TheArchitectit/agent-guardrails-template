// Package secretfile is the single decision for reading a read-only mounted
// secret file (Spec 17 R17-06 / R17-08).
//
// R17-06 makes a managed secret store or a runtime read-only mounted secret
// file the first-choice production source for deployment secrets, and requires
// file reads to be restricted to the service identity with no group or world
// access. R17-08 requires that secret material never reach argv, logs or error
// output.
//
// Two loaders read secret files — the credential-registry loader
// (internal/auth) and the configuration loader (internal/config) — and both
// must apply the same rule: a regular file, no group/world access on POSIX
// hosts, and any error is fail-closed. Keeping one implementation here is what
// stops the two call sites from drifting apart.
package secretfile

import (
	"fmt"
	"os"
	"runtime"
)

// Mode is the POSIX mode a secret file must have: owner read/write only, no
// group or world access (Spec 17 R17-06). Restrictive Windows ACL equivalents
// cannot be observed through os.FileMode, so the mode check is enforced on
// non-Windows hosts and documented for Windows.
const Mode os.FileMode = 0o600

// PermitsRestrictedOnly reports whether a file mode grants no group or world
// access. An owner-only file (0600, 0400, …) passes; any group/other bits
// (0644, 0640, 0660, …) fail.
func PermitsRestrictedOnly(mode os.FileMode) bool {
	return mode.Perm()&0o077 == 0
}

// Read reads a read-only mounted secret file after rejecting non-regular files
// and, on POSIX hosts, any group/world-accessible mode. The caller treats any
// error as fail-closed.
//
// The error never includes the file's contents; it may name the path, because
// its only consumers are operator-facing startup failures (the same convention
// the credential-registry loader and the migration runner already follow).
func Read(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("secret source must be a regular file")
	}
	if runtime.GOOS != "windows" && !PermitsRestrictedOnly(info.Mode()) {
		return nil, fmt.Errorf("secret source is group/world accessible; require POSIX mode %#o", uint32(Mode))
	}
	return os.ReadFile(path)
}
