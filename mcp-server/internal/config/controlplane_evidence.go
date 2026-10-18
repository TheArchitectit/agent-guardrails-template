package config

import (
	"fmt"
	"strings"
	"time"
)

// Control-plane evidence retention posture (Spec 08 R08-4.3.6).
//
// R08-4.3.6: "Evidence retention, signing, access control, and deletion policy
// MUST be configurable and auditable." Configurable is the half that is easy to
// fake. The three ways to fake it, and the rule that refuses each:
//
//  1. A store with no retention window. "Keep forever" is a policy, and one
//     nobody chose: it grows without bound on a box whose disk is a production
//     dependency. A configured directory therefore REQUIRES a positive window.
//  2. A store with no signing identity. An unsigned retained record cannot be
//     told apart from an edited one, which is the whole value of retaining it.
//     A configured store outside `local` therefore requires EVIDENCE_PRIVATE_KEY
//     (which ValidateEvidenceIdentity already gates); `local` may retain
//     unsigned, and the store reports that state rather than implying a
//     signature it does not have.
//  3. A count bound that cannot hold a day of evaluations. A max_records of 1
//     with a 30-day window silently keeps one record and looks configured; the
//     minimum keeps the setting from being a decoration.
//
// An UNCONFIGURED store is not an error: a deployment that never opted into a
// composition retains nothing, and ValidateCompositionEvidenceRetention is
// called only when a directory is set.

// minEvidenceMaxRecords is the smallest count bound this build will accept.
// It is not a tuning knob: a bound below this cannot hold a working day of
// evaluations for a single instance, so it is a misconfiguration rather than a
// policy.
const minEvidenceMaxRecords = 10

// maxEvidenceRetention bounds the window. A retention longer than two years is
// almost always a typo in a duration string ("730d" parsed as hours, say), and
// a store that silently keeps a record for a decade is a data-retention
// liability rather than evidence.
const maxEvidenceRetention = 2 * 365 * 24 * time.Hour

// ValidateCompositionEvidenceRetention fails startup when the configured
// retention posture cannot meet R08-4.3.6.
func (c *Config) ValidateCompositionEvidenceRetention() error {
	dir := strings.TrimSpace(c.ControlPlaneEvidenceDir)
	if dir == "" {
		// Not configured: nothing is retained, and a retention window or count
		// bound with no directory is a policy for a store that does not exist.
		// Refusing it prevents an operator from believing evidence is retained
		// because RETENTION was set.
		if c.ControlPlaneEvidenceRetention != 0 || c.ControlPlaneEvidenceMaxRecords != 0 {
			return fmt.Errorf(
				"CONTROL_PLANE_EVIDENCE_RETENTION/MAX_RECORDS are set but CONTROL_PLANE_EVIDENCE_DIR is empty (Spec 08 R08-4.3.6): a retention policy with no store retains nothing")
		}
		return nil
	}

	if c.ControlPlaneEvidenceRetention <= 0 {
		return fmt.Errorf(
			"CONTROL_PLANE_EVIDENCE_DIR is set but CONTROL_PLANE_EVIDENCE_RETENTION is not (Spec 08 R08-4.3.6): retention is required, because an implicit \"keep forever\" is a policy nobody chose and an unbounded evidence store is an unsized disk dependency")
	}
	if c.ControlPlaneEvidenceRetention > maxEvidenceRetention {
		return fmt.Errorf(
			"CONTROL_PLANE_EVIDENCE_RETENTION %s exceeds the %s maximum (Spec 08 R08-4.3.6): a longer window is almost always a duration typo and a decade-long retention is a liability, not evidence",
			c.ControlPlaneEvidenceRetention, maxEvidenceRetention)
	}
	if c.ControlPlaneEvidenceMaxRecords < 0 {
		return fmt.Errorf(
			"CONTROL_PLANE_EVIDENCE_MAX_RECORDS must be non-negative, got %d",
			c.ControlPlaneEvidenceMaxRecords)
	}
	if c.ControlPlaneEvidenceMaxRecords > 0 && c.ControlPlaneEvidenceMaxRecords < minEvidenceMaxRecords {
		return fmt.Errorf(
			"CONTROL_PLANE_EVIDENCE_MAX_RECORDS %d is below the %d minimum (Spec 08 R08-4.3.6): a bound this small cannot hold a working day of evaluations and is a decoration, not a policy",
			c.ControlPlaneEvidenceMaxRecords, minEvidenceMaxRecords)
	}

	// A retained record outside `local` must carry a signature, so the store
	// needs the instance identity. This duplicates ValidateEvidenceIdentity on
	// purpose: that check is about outbound artifacts and can be satisfied by a
	// producer ID alone in future, while this one is specifically about the
	// retained record being verifiable later.
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		return err
	}
	if profile != DeploymentProfileLocal && strings.TrimSpace(c.EvidencePrivateKey) == "" {
		return fmt.Errorf(
			"profile %q retains control-plane evidence but has no EVIDENCE_PRIVATE_KEY (Spec 08 R08-4.3.6): an unsigned retained record cannot be distinguished from an edited one",
			profile)
	}
	return nil
}
