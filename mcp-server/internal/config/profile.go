package config

import (
	"fmt"
	"net"
	"strings"
)

// Deployment profile names (Spec 17 §2). The profile is selected explicitly at
// startup via DEPLOYMENT_PROFILE and defaults to the safest option.
const (
	DeploymentProfileLocal   = "local"
	DeploymentProfileTailnet = "tailnet"
	DeploymentProfilePublic  = "public"
)

// NormalizeDeploymentProfile maps an unset profile to the safe default
// (local) and rejects unknown names.
func NormalizeDeploymentProfile(profile string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(profile))
	if p == "" {
		return DeploymentProfileLocal, nil
	}
	switch p {
	case DeploymentProfileLocal, DeploymentProfileTailnet, DeploymentProfilePublic:
		return p, nil
	default:
		return "", fmt.Errorf("DEPLOYMENT_PROFILE must be one of local, tailnet, public, got %q", profile)
	}
}

// IsLoopbackBindHost reports whether host is a loopback-only bind target
// (localhost or a 127.0.0.0/8 / ::1 address).
func IsLoopbackBindHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// IsWildcardBindHost reports whether host binds every interface
// (0.0.0.0, ::, or an empty host as produced by Go listener syntax ":8080").
func IsWildcardBindHost(host string) bool {
	if host == "" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}

// BindHostFromAddr extracts the host portion of a Go listener address
// ("host:port", "[v6host]:port", or a bare host).
func BindHostFromAddr(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// ValidateDeploymentBinding validates a listener bind address against the
// effective deployment profile (Spec 17 R17-01/R17-02). It fails closed on a
// mismatch instead of silently downgrading:
//
//   - local: loopback bind required. A wildcard bind is accepted only with an
//     explicit DEPLOYMENT_BIND_ATTESTATION that the wildcard listen address is
//     on a verified isolated local backend network (local Compose with
//     host-loopback-only publishing).
//   - tailnet: a wildcard bind is never accepted; bind the selected private
//     overlay interface. A wildcard bind is not private by name alone.
//   - public: explicit proxy configuration (TRUSTED_PROXIES) is mandatory —
//     the backend may listen only behind a controlled TLS-terminating proxy.
func (c *Config) ValidateDeploymentBinding(bindAddr string) error {
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		return err
	}
	host := BindHostFromAddr(bindAddr)

	switch profile {
	case DeploymentProfileLocal:
		if IsLoopbackBindHost(host) {
			return nil
		}
		if IsWildcardBindHost(host) {
			if strings.TrimSpace(c.DeploymentBindAttestation) == "" {
				return fmt.Errorf(
					"profile %q rejects wildcard bind %q: a container-wide bind is allowed only on a verified isolated local backend network; set DEPLOYMENT_BIND_ATTESTATION to attest the isolated local bridge with host-loopback-only publishing, or bind a loopback address",
					profile, bindAddr)
			}
			return nil
		}
		return fmt.Errorf("profile %q requires a loopback bind address, got %q", profile, bindAddr)

	case DeploymentProfileTailnet:
		if IsWildcardBindHost(host) {
			return fmt.Errorf(
				"profile %q rejects wildcard bind %q: bind the selected private overlay interface; a wildcard bind is not private by name alone",
				profile, bindAddr)
		}
		return nil

	case DeploymentProfilePublic:
		if len(c.TrustedProxies) == 0 {
			return fmt.Errorf(
				"profile %q requires explicit proxy configuration (TRUSTED_PROXIES): the backend may only listen behind a controlled TLS-terminating proxy with a dedicated proxy upstream",
				profile)
		}
		return nil

	default:
		// NormalizeDeploymentProfile already rejected unknown profiles.
		return fmt.Errorf("unknown deployment profile %q", profile)
	}
}

// MCPBindAddr returns the bind address for the MCP listener.
func (c *Config) MCPBindAddr() string {
	return fmt.Sprintf("%s:%d", c.MCPBindHost, c.MCPPort)
}

// WebBindAddr returns the bind address for the web listener.
func (c *Config) WebBindAddr() string {
	return fmt.Sprintf("%s:%d", c.WebBindHost, c.WebPort)
}
