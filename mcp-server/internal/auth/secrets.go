package auth

import (
	"fmt"
	"os"
	"runtime"
)

// SecretFileMode is the POSIX mode a credential secret file must have: owner
// read/write only, no group or world access (Spec 17 R17-06). Restrictive
// Windows ACL equivalents cannot be observed through os.FileMode, so the mode
// check is enforced on non-Windows hosts and documented for Windows.
const SecretFileMode os.FileMode = 0o600

// permitsRestrictedOnly reports whether a file mode grants no group or world
// access. An owner-only file (0600, 0400, …) passes; any group/other bits
// (0644, 0640, 0660, …) fail.
func permitsRestrictedOnly(mode os.FileMode) bool {
	return mode.Perm()&0o077 == 0
}

// readSecretFile reads a read-only mounted secret file after rejecting
// non-regular files and, on POSIX hosts, any group/world-accessible mode. The
// caller treats any error as fail-closed.
func readSecretFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("secret source must be a regular file")
	}
	if runtime.GOOS != "windows" && !permitsRestrictedOnly(info.Mode()) {
		return nil, fmt.Errorf("secret source is group/world accessible; require POSIX mode %#o", uint32(SecretFileMode))
	}
	return os.ReadFile(path)
}
