package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Spec 17 R17-09 requires that pprof stay disabled by default and that, when it
// is enabled for diagnosis, it bind a local-only administrative endpoint with
// no public/tailnet proxy route and a bounded access window. This file holds
// the single decision for that posture. It is enforced from Config.Validate at
// startup and consulted again when the debug listener is actually started, so
// an expired window stops the endpoint instead of being a one-time check.

// defaultPProfBindHost is the reviewed loopback default for the debug listener.
const defaultPProfBindHost = "127.0.0.1"

// PProfBindAddr is the declared debug-listener address. The host is a reviewed
// loopback default; ValidateDebugEndpoint refuses anything that is not
// loopback, so this never produces a non-local administrative endpoint.
func (c *Config) PProfBindAddr() string {
	if c == nil {
		return net.JoinHostPort(defaultPProfBindHost, "6060")
	}
	host := strings.TrimSpace(c.PProfBindHost)
	if host == "" {
		host = defaultPProfBindHost
	}
	port := c.PProfPort
	if port <= 0 || port > 65535 {
		port = 6060
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// PProfDeadline parses PPROF_UNTIL. An unset value returns (zero, false, nil):
// no access window is configured. A malformed value is a hard error, never a
// silent "unbounded".
func (c *Config) PProfDeadline() (time.Time, bool, error) {
	if c == nil || strings.TrimSpace(c.PProfUntil) == "" {
		return time.Time{}, false, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(c.PProfUntil))
	if err != nil {
		return time.Time{}, false, fmt.Errorf("PPROF_UNTIL must be an RFC3339 timestamp, got %q", c.PProfUntil)
	}
	return t, true, nil
}

// DebugEndpointActive reports whether the debug endpoint may serve as of now,
// and the deadline it was judged against. A disabled endpoint, an unconfigured
// or malformed window, or a window that has passed all return false — the
// failure mode of an unread posture must not be a serving debug endpoint.
func (c *Config) DebugEndpointActive(now time.Time) (bool, time.Time) {
	if c == nil || !c.PProfEnabled {
		return false, time.Time{}
	}
	deadline, configured, err := c.PProfDeadline()
	if err != nil || !configured {
		return false, time.Time{}
	}
	return now.Before(deadline), deadline
}

// ValidateDebugEndpoint enforces the R17-09 debug posture: a local-only bind, a
// required time-box, and — outside the local profile — an explicit attestation
// that no public/tailnet proxy route reaches the loopback endpoint. A disabled
// endpoint validates to nil.
func (c *Config) ValidateDebugEndpoint() error {
	if c == nil || !c.PProfEnabled {
		return nil
	}
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		return err
	}

	host := strings.TrimSpace(c.PProfBindHost)
	if host == "" {
		host = defaultPProfBindHost
	}
	if !IsLoopbackBindHost(host) {
		return fmt.Errorf("PPROF_BIND_HOST must be a loopback address (Spec 17 R17-09: the debug endpoint is local-only), got %q", c.PProfBindHost)
	}
	// Port 0 means "use the documented default" (matching PProfBindAddr); an
	// explicitly set out-of-range port is a configuration error.
	if c.PProfPort < 0 || c.PProfPort > 65535 {
		return fmt.Errorf("PPROF_PORT must be between 1 and 65535, got %d", c.PProfPort)
	}

	deadline, configured, err := c.PProfDeadline()
	if err != nil {
		return err
	}
	if !configured {
		return fmt.Errorf("PPROF_ENABLED requires PPROF_UNTIL (Spec 17 R17-09: debug access must be time-boxed); set an RFC3339 deadline")
	}
	if now := time.Now(); !now.Before(deadline) {
		return fmt.Errorf("PPROF_UNTIL %q is not in the future (Spec 17 R17-09: the debug window is expired); raise the deadline or disable PPROF_ENABLED", deadline.UTC().Format(time.RFC3339))
	}
	if profile != DeploymentProfileLocal && strings.TrimSpace(c.PProfNoProxyRouteAttestation) == "" {
		return fmt.Errorf("profile %q enables the debug endpoint (Spec 17 R17-09): set PPROF_NO_PROXY_ROUTE_ATTESTATION to attest there is no public/tailnet proxy route to the loopback debug endpoint, or disable PPROF_ENABLED", profile)
	}
	return nil
}
