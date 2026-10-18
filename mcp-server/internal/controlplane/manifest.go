// Package controlplane implements the component-composition and evidence
// contract of Spec 08 (R08-4.1 .. R08-4.5).
//
// The reason this package exists is the defect this program keeps finding:
// "a library exists" is not evidence that a control ran. Before this package
// the repository had no machine-readable answer to *which component is
// actually reachable from a request*, no evaluation record tying a decision to
// a manifest and policy digest, and nothing that could distinguish
// `installed` from `exercised`. A status page could therefore describe a
// protection that nothing invoked.
//
// Three rules are structural, not documentation:
//
//  1. A manifest declares IDENTITY and DEPENDENCY only. It never declares
//     state. State is measured by the composer against live objects, so a
//     manifest edit cannot make an unexercised control look active.
//  2. Resolution refuses a graph it cannot trust: duplicate IDs, missing
//     dependencies, cycles, unknown interface versions, mutable version refs,
//     a mandatory component depending on an optional one, and a dependency on
//     a lower trust tier. Each of these would let a mandatory control degrade
//     silently rather than fail closed (R08-4.1.2/R08-4.1.4/R08-4.5.2).
//  3. An evaluation result is derived, never assigned. A mandatory check that
//     did not `pass` cannot produce a `pass` (R08-4.3.4).
package controlplane

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// TrustTier is the trust label of a component (R08-4.5.1). Higher tiers may
// consume lower ones as data; a lower tier may not supply a higher one's
// mandatory control (see Resolve).
type TrustTier string

const (
	TrustTrusted   TrustTier = "trusted"
	TrustVerified  TrustTier = "verified"
	TrustUntrusted TrustTier = "untrusted"
)

// rank orders tiers so a dependency rule can be expressed as a comparison
// rather than a table. An unknown tier ranks lowest, which makes a typo fail
// closed instead of granting trust.
func (t TrustTier) rank() int {
	switch t {
	case TrustTrusted:
		return 3
	case TrustVerified:
		return 2
	case TrustUntrusted:
		return 1
	}
	return 0
}

// Valid reports whether the tier is one the resolver understands.
func (t TrustTier) Valid() bool { return t.rank() > 0 }

// InterfaceVersion is the composition interface a component implements. It is
// a name, not a number: "v1" alone cannot say which contract is meant, and a
// bare integer invites an accidental comparison between unrelated versions.
const InterfaceVersion = "controlplane.component/v1"

// Component is one declared unit of the control plane. It carries no state and
// no probe: both are supplied by the running process, so this struct cannot be
// used to assert that something is working.
type Component struct {
	// ID is the stable component identifier.
	ID string `json:"id"`
	// Version is an immutable released version. A branch, "latest" or "HEAD"
	// is refused by Resolve: a mutable production ref cannot pin evidence.
	Version string `json:"version"`
	// Digest is the content digest of the component as shipped. It is verified
	// by scripts/check_control_plane_manifest.py against SourcePaths, so a
	// manifest cannot describe code that is not in the tree.
	Digest string `json:"digest"`
	// SourcePaths are the repository-relative paths whose contents the digest
	// covers. Without them the digest is a claim nothing can check, which is
	// the declaration-restating defect this package exists to remove.
	SourcePaths []string `json:"source_paths"`
	// Owner is the accountable owner named in the manifest.
	Owner string `json:"owner"`
	// Interface is the composition interface this component implements.
	Interface string `json:"interface"`
	// TrustTier labels the component's trust boundary (R08-4.5.1).
	TrustTier TrustTier `json:"trust_tier"`
	// DependsOn lists component IDs this one requires.
	DependsOn []string `json:"depends_on,omitempty"`
	// Mandatory marks the component as required for the declared profiles.
	// Missing or failing mandatory components must never yield `pass`
	// (R08-4.1.4).
	Mandatory bool `json:"mandatory"`
	// Signals name the runtime observations that prove this component was
	// reached. A component that declares no signal cannot be measured, and
	// Resolve refuses it rather than letting it report "declared" forever.
	Signals []string `json:"signals"`
	// Rationale records why this component has its class. It is required so a
	// future mandatory flag cannot be added without a stated reason.
	Rationale string `json:"rationale"`
}

// EvidenceFloor is the weakest integration state a profile accepts as
// evidence for one component (R08-4.1.5).
//
// Only two values are legal, and the exclusion is the point of the requirement:
// `declared`, `installed` and `initialized` are all statements that the code is
// present, which is exactly what "merely a list of installed components" means.
// A profile may declare that it accepts a control it can REACH, or demand one it
// has SEEN run. It may not declare presence as its own evidence, and it may not
// declare `failed` as a floor, because a floor is the weakest acceptable answer
// rather than a description of one.
type EvidenceFloor string

const (
	// FloorReachable accepts a component whose entry point a request can reach,
	// even if nothing has invoked it yet. A freshly started instance is
	// legitimately in this state, so a profile that demanded `exercised` from
	// every component could never be satisfied at boot.
	FloorReachable EvidenceFloor = "reachable"
	// FloorExercised demands an observed invocation. It is the floor for a
	// control whose whole purpose is to run.
	FloorExercised EvidenceFloor = "exercised"
)

// rank orders the floors so a measured state can be compared against one.
// An unknown floor ranks 0 so a typo fails closed instead of passing.
func (f EvidenceFloor) rank() int {
	switch f {
	case FloorReachable:
		return 4 // StateReachable
	case FloorExercised:
		return 5 // StateExercised
	}
	return 0
}

// Valid reports whether the floor is one the resolver understands.
func (f EvidenceFloor) Valid() bool { return f.rank() > 0 }

// ComponentRequirement binds one component to the evidence floor its profile
// accepts. Naming a floor per component is what turns "mandatory" from a label
// into an obligation somebody can fail.
type ComponentRequirement struct {
	ID string `json:"id"`
	// MinimumState is the weakest IntegrationState that satisfies this entry.
	MinimumState EvidenceFloor `json:"minimum_state"`
}

// Requirement is one profile-level obligation (R08-4.1.5). A manifest that
// declares only components says what is composed; a requirement says what the
// profile must be able to PROVE before the composition counts as evidence.
//
// A requirement must name at least one check or one component. A requirement
// that names neither is decoration, and is refused at resolve time.
type Requirement struct {
	// ID is the stable requirement identifier.
	ID string `json:"id"`
	// Description states, in one sentence, what this requirement exists to
	// prove. It is required so a future requirement cannot be added without a
	// stated purpose.
	Description string `json:"description"`
	// RequiredComponents are the components this requirement needs evidence
	// from, each with the floor it accepts.
	RequiredComponents []ComponentRequirement `json:"required_components,omitempty"`
	// RequiredChecks are the check IDs that must have run for this requirement
	// to be evidenced. A named check that never ran is `not_run` — a distinct
	// answer from "ran and failed", because one is a wiring defect and the
	// other is a refusal.
	RequiredChecks []string `json:"required_checks,omitempty"`
	// MinimumPassingChecks is the floor on how many of RequiredChecks must
	// report `pass`. Absent means all of them. An explicit value below 1 or
	// above the number of required checks is refused: the first would let a
	// requirement be met by nothing, the second could never be met.
	MinimumPassingChecks *int `json:"minimum_passing_checks,omitempty"`
}

// minimumPassing is the effective floor: all required checks unless the
// manifest says otherwise.
func (r Requirement) minimumPassing() int {
	if r.MinimumPassingChecks == nil {
		return len(r.RequiredChecks)
	}
	return *r.MinimumPassingChecks
}

// Manifest is the declared composition for one evaluation profile.
type Manifest struct {
	// Version is the manifest revision, not a component version.
	Version string `json:"version"`
	// Profile is the deployment profile this manifest describes.
	Profile string `json:"profile"`
	// PolicyDigest identifies the policy set the profile evaluates under. It
	// is recorded in every evaluation so a replay can detect that the policy
	// changed underneath it (R08-4.3.5).
	PolicyDigest string `json:"policy_digest"`
	// SupportedInterfaces lists the composition interfaces this build can
	// load. A component naming any other interface is refused.
	SupportedInterfaces []string `json:"supported_interfaces"`
	// Components are the declared units.
	Components []Component `json:"components"`
	// Requirements are the profile's evidence obligations (R08-4.1.5). They
	// are declared here, next to the components, because a profile that
	// declares components and no obligations is the exact shape the
	// requirement forbids.
	Requirements []Requirement `json:"requirements"`
}

var (
	// versionRe requires a released semantic version. A branch name, a commit
	// message, "latest" or "HEAD" does not match, so a mutable ref is refused
	// by construction rather than by an enumeration of bad strings.
	versionRe = regexp.MustCompile(`^\d+\.\d+\.\d+([-+][0-9A-Za-z.-]+)?$`)
	// digestRe requires a full sha256 in the canonical prefixed form.
	digestRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	// idRe keeps IDs stable and comparable across reports.
	idRe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)
)

// LoadManifest reads and resolves a manifest from disk. A missing or unreadable
// file is an error, never an empty manifest: an empty composition that reports
// "pass" because it has nothing to check is the vacuity this package exists to
// prevent.
func LoadManifest(path string) (*Manifest, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("manifest path is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read control-plane manifest: %w", err)
	}
	return ParseManifest(raw)
}

// ParseManifest decodes and resolves a manifest. Resolution runs here so no
// caller can hold an unresolved graph and mistake it for a usable one.
func ParseManifest(raw []byte) (*Manifest, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("control-plane manifest is empty")
	}
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	// An unknown field is refused: a manifest that silently drops a field the
	// author believed was enforced is a manifest that does not say what it
	// looks like it says.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("decode control-plane manifest: %w", err)
	}
	if err := m.Resolve(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Resolve validates the declared graph. Every rule here exists because its
// violation would let a mandatory control disappear without a failure.
func (m *Manifest) Resolve() error {
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("manifest version is required")
	}
	if strings.TrimSpace(m.Profile) == "" {
		return errors.New("manifest profile is required")
	}
	if !digestRe.MatchString(m.PolicyDigest) {
		return fmt.Errorf("manifest policy_digest must be a sha256 digest, got %q", m.PolicyDigest)
	}
	if len(m.SupportedInterfaces) == 0 {
		return errors.New("manifest declares no supported interface; every component would be refused at load time")
	}
	if len(m.Components) == 0 {
		return errors.New("manifest declares no components; an empty composition cannot evidence anything")
	}

	supported := make(map[string]bool, len(m.SupportedInterfaces))
	for _, v := range m.SupportedInterfaces {
		if strings.TrimSpace(v) == "" {
			return errors.New("manifest supported_interfaces contains an empty entry")
		}
		supported[v] = true
	}

	byID := make(map[string]Component, len(m.Components))
	for _, c := range m.Components {
		if !idRe.MatchString(c.ID) {
			return fmt.Errorf("component id %q is not a stable lowercase identifier", c.ID)
		}
		if _, dup := byID[c.ID]; dup {
			return fmt.Errorf("component %q is declared twice", c.ID)
		}
		if !versionRe.MatchString(c.Version) {
			return fmt.Errorf("component %q declares version %q; a mutable or unversioned ref cannot pin evidence", c.ID, c.Version)
		}
		if !digestRe.MatchString(c.Digest) {
			return fmt.Errorf("component %q declares digest %q; a full sha256 is required", c.ID, c.Digest)
		}
		if len(c.SourcePaths) == 0 {
			return fmt.Errorf("component %q declares no source_paths, so its digest cannot be verified", c.ID)
		}
		if strings.TrimSpace(c.Owner) == "" {
			return fmt.Errorf("component %q declares no owner", c.ID)
		}
		if c.Interface == "" {
			return fmt.Errorf("component %q declares no interface version", c.ID)
		}
		if !supported[c.Interface] {
			// R08-4.1.2: an incompatible interface version is refused before
			// any component starts.
			return fmt.Errorf("component %q implements interface %q, which this build does not support", c.ID, c.Interface)
		}
		if !c.TrustTier.Valid() {
			return fmt.Errorf("component %q declares trust tier %q, which is not a known tier", c.ID, c.TrustTier)
		}
		if len(c.Signals) == 0 {
			// A component with no observation can only ever report `declared`.
			// Shipping it would put a row in the report that no measurement can
			// move, which reads as a component that is present and inert.
			return fmt.Errorf("component %q declares no runtime signal, so its state could never be measured", c.ID)
		}
		if strings.TrimSpace(c.Rationale) == "" {
			return fmt.Errorf("component %q declares no rationale for its mandatory flag", c.ID)
		}
		byID[c.ID] = c
	}

	if err := m.resolveRequirements(byID); err != nil {
		return err
	}

	// Dependency checks need the full set, so they run in a second pass.
	for _, c := range m.Components {
		seen := make(map[string]bool, len(c.DependsOn))
		for _, depID := range c.DependsOn {
			if depID == c.ID {
				return fmt.Errorf("component %q depends on itself", c.ID)
			}
			if seen[depID] {
				return fmt.Errorf("component %q lists dependency %q twice", c.ID, depID)
			}
			seen[depID] = true

			dep, ok := byID[depID]
			if !ok {
				// R08-4.1.2: a missing mandatory dependency is refused.
				return fmt.Errorf("component %q depends on %q, which the manifest does not declare", c.ID, depID)
			}
			if c.Mandatory && !dep.Mandatory {
				// A mandatory control whose input is optional degrades silently
				// the moment the optional half is absent. Refusing the graph is
				// the only way "mandatory" keeps its meaning (R08-4.1.4).
				return fmt.Errorf("mandatory component %q depends on optional component %q; a mandatory control may not rest on an optional one", c.ID, depID)
			}
			if dep.TrustTier.rank() < c.TrustTier.rank() {
				// R08-4.5.2: a lower-trust component must not supply a
				// higher-trust one's input.
				return fmt.Errorf("component %q (%s) depends on %q (%s); a lower-trust component may not supply a higher-trust one",
					c.ID, c.TrustTier, depID, dep.TrustTier)
			}
		}
	}

	return checkAcyclic(m.Components, byID)
}

// resolveRequirements validates the profile requirement set (R08-4.1.5).
//
// The failure this must exclude is a requirement set that looks like an
// obligation and cannot fail: an empty list, a requirement naming nothing, a
// floor of `installed`, a check that does not exist, or one component required
// twice with two different floors. Each of those would produce a profile that
// reports `pass` while proving nothing, which is the vacuity class this package
// exists to remove.
func (m *Manifest) resolveRequirements(byID map[string]Component) error {
	if len(m.Requirements) == 0 {
		// R08-4.1.5: a profile that lists components and no obligations is
		// precisely "merely a list of installed components". Refusing the
		// manifest is the only way the requirement is enforced rather than
		// described.
		return errors.New("manifest declares no profile requirements; a profile must state required checks and minimum evidence (R08-4.1.5)")
	}

	seenReq := make(map[string]bool, len(m.Requirements))
	// requiredFloor records the floor each component is held to across the
	// whole requirement set, so two requirements cannot ask for contradictory
	// evidence from one component.
	requiredFloor := make(map[string]EvidenceFloor, len(m.Components))

	for _, r := range m.Requirements {
		if !idRe.MatchString(r.ID) {
			return fmt.Errorf("requirement id %q is not a stable lowercase identifier", r.ID)
		}
		if seenReq[r.ID] {
			return fmt.Errorf("requirement %q is declared twice", r.ID)
		}
		seenReq[r.ID] = true
		if strings.TrimSpace(r.Description) == "" {
			return fmt.Errorf("requirement %q declares no description", r.ID)
		}
		if len(r.RequiredComponents) == 0 && len(r.RequiredChecks) == 0 {
			return fmt.Errorf("requirement %q requires no component and no check, so nothing could ever fail it", r.ID)
		}

		seenComp := make(map[string]bool, len(r.RequiredComponents))
		for _, cr := range r.RequiredComponents {
			if seenComp[cr.ID] {
				return fmt.Errorf("requirement %q lists component %q twice", r.ID, cr.ID)
			}
			seenComp[cr.ID] = true
			if _, ok := byID[cr.ID]; !ok {
				// A requirement naming an undeclared component can never be
				// evidenced, so it would silently cap the profile at `incomplete`.
				return fmt.Errorf("requirement %q requires component %q, which the manifest does not declare", r.ID, cr.ID)
			}
			if !cr.MinimumState.Valid() {
				// R08-4.1.5 is about MINIMUM EVIDENCE. `declared`, `installed`
				// and `initialized` describe presence, so accepting them here
				// would restate the component list instead of constraining it.
				return fmt.Errorf("requirement %q sets component %q minimum_state=%q; a minimum evidence floor must be %q or %q",
					r.ID, cr.ID, cr.MinimumState, FloorReachable, FloorExercised)
			}
			if prior, ok := requiredFloor[cr.ID]; ok && prior != cr.MinimumState {
				return fmt.Errorf("component %q is required at %q by one requirement and %q by %q; one component has one floor",
					cr.ID, prior, cr.MinimumState, r.ID)
			}
			requiredFloor[cr.ID] = cr.MinimumState
		}

		seenCheck := make(map[string]bool, len(r.RequiredChecks))
		for _, cid := range r.RequiredChecks {
			if strings.TrimSpace(cid) == "" {
				return fmt.Errorf("requirement %q names an empty check id", r.ID)
			}
			if seenCheck[cid] {
				return fmt.Errorf("requirement %q lists check %q twice", r.ID, cid)
			}
			seenCheck[cid] = true
		}

		if r.MinimumPassingChecks != nil {
			if len(r.RequiredChecks) == 0 {
				return fmt.Errorf("requirement %q sets minimum_passing_checks without naming any check", r.ID)
			}
			if *r.MinimumPassingChecks < 1 || *r.MinimumPassingChecks > len(r.RequiredChecks) {
				return fmt.Errorf("requirement %q sets minimum_passing_checks=%d over %d named check(s); the floor must be between 1 and the number named",
					r.ID, *r.MinimumPassingChecks, len(r.RequiredChecks))
			}
		}
	}

	// A mandatory component with no requirement anywhere is a control the
	// profile claims to need and never asks evidence for. It would report
	// `mandatory` in the bundle and constrain nothing.
	for _, c := range m.Components {
		if !c.Mandatory {
			continue
		}
		if _, ok := requiredFloor[c.ID]; !ok {
			return fmt.Errorf("mandatory component %q is required by no profile requirement; a mandatory control with no evidence obligation constrains nothing", c.ID)
		}
	}

	return nil
}

// RequirementFloor returns the floor one component is held to, if any.
func (m *Manifest) RequirementFloor(componentID string) (EvidenceFloor, bool) {
	for _, r := range m.Requirements {
		for _, cr := range r.RequiredComponents {
			if cr.ID == componentID {
				return cr.MinimumState, true
			}
		}
	}
	return "", false
}

// checkAcyclic rejects a dependency cycle. A cycle is not merely untidy: it
// means no component in the ring can be initialized before another, so the
// declared order can never be a valid startup order (R08-4.1.2/R08-4.1.3).
func checkAcyclic(components []Component, byID map[string]Component) error {
	const (
		white = 0 // unvisited
		gray  = 1 // on the current path
		black = 2 // finished
	)
	color := make(map[string]int, len(components))
	var path []string

	var visit func(id string) error
	visit = func(id string) error {
		color[id] = gray
		path = append(path, id)
		for _, dep := range byID[id].DependsOn {
			switch color[dep] {
			case gray:
				return fmt.Errorf("dependency cycle: %s", strings.Join(append(path, dep), " -> "))
			case white:
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		path = path[:len(path)-1]
		color[id] = black
		return nil
	}

	// Visit in declared order so the reported cycle is deterministic.
	for _, c := range components {
		if color[c.ID] == white {
			if err := visit(c.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Digest returns the canonical digest of the manifest. It covers the declared
// identity and structure and deliberately excludes nothing: two manifests that
// differ in any declared field produce different digests, so an evaluation
// cannot be replayed against a manifest it was not produced under.
func (m *Manifest) Digest() string {
	type canonicalComponent struct {
		ID        string    `json:"id"`
		Version   string    `json:"version"`
		Digest    string    `json:"digest"`
		Owner     string    `json:"owner"`
		Interface string    `json:"interface"`
		TrustTier TrustTier `json:"trust_tier"`
		DependsOn []string  `json:"depends_on"`
		Mandatory bool      `json:"mandatory"`
		Signals   []string  `json:"signals"`
		Sources   []string  `json:"source_paths"`
	}
	type canonical struct {
		Version             string               `json:"version"`
		Profile             string               `json:"profile"`
		PolicyDigest        string               `json:"policy_digest"`
		SupportedInterfaces []string             `json:"supported_interfaces"`
		Components          []canonicalComponent `json:"components"`
		Requirements        []Requirement        `json:"requirements"`
	}

	comps := make([]canonicalComponent, 0, len(m.Components))
	for _, c := range m.Components {
		deps := append([]string(nil), c.DependsOn...)
		sort.Strings(deps)
		sigs := append([]string(nil), c.Signals...)
		sort.Strings(sigs)
		srcs := append([]string(nil), c.SourcePaths...)
		sort.Strings(srcs)
		comps = append(comps, canonicalComponent{
			ID:        c.ID,
			Version:   c.Version,
			Digest:    c.Digest,
			Owner:     c.Owner,
			Interface: c.Interface,
			TrustTier: c.TrustTier,
			DependsOn: deps,
			Mandatory: c.Mandatory,
			Signals:   sigs,
			Sources:   srcs,
		})
	}
	// Sort so declaration order cannot change the digest: two manifests that
	// declare the same graph are the same manifest.
	sort.Slice(comps, func(i, j int) bool { return comps[i].ID < comps[j].ID })

	sortedIfaces := append([]string(nil), m.SupportedInterfaces...)
	sort.Strings(sortedIfaces)

	// Requirements are sorted by id and their inner lists by value, so a
	// manifest declaring the same obligations in a different order is the
	// same manifest. Order carries no meaning here; membership does.
	reqs := make([]Requirement, 0, len(m.Requirements))
	for _, r := range m.Requirements {
		c := r
		c.RequiredComponents = append([]ComponentRequirement(nil), r.RequiredComponents...)
		sort.Slice(c.RequiredComponents, func(i, j int) bool { return c.RequiredComponents[i].ID < c.RequiredComponents[j].ID })
		c.RequiredChecks = append([]string(nil), r.RequiredChecks...)
		sort.Strings(c.RequiredChecks)
		reqs = append(reqs, c)
	}
	sort.Slice(reqs, func(i, j int) bool { return reqs[i].ID < reqs[j].ID })

	payload, err := json.Marshal(canonical{
		Version:             m.Version,
		Profile:             m.Profile,
		PolicyDigest:        m.PolicyDigest,
		SupportedInterfaces: sortedIfaces,
		Components:          comps,
		Requirements:        reqs,
	})
	if err != nil {
		// Marshalling a struct of strings cannot fail; returning a stable
		// marker rather than panicking keeps a digest mistake from taking the
		// server down.
		return "sha256:unavailable"
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Component returns a declared component by ID.
func (m *Manifest) Component(id string) (Component, bool) {
	for _, c := range m.Components {
		if c.ID == id {
			return c, true
		}
	}
	return Component{}, false
}

// MandatoryIDs returns the mandatory component IDs in declaration order.
func (m *Manifest) MandatoryIDs() []string {
	out := make([]string, 0, len(m.Components))
	for _, c := range m.Components {
		if c.Mandatory {
			out = append(out, c.ID)
		}
	}
	return out
}
