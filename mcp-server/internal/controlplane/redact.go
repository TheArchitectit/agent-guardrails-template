package controlplane

// Redaction for the evidence bundle (Spec 08 R08-4.3.3).
//
// An evidence bundle is serialized and shipped — to an operator, a peer
// product, a ticket, a log sink. Every byte in it leaves the trust boundary of
// the process. So the requirement "secret values MUST not be stored" cannot be
// satisfied by asking observers to be careful: the observer supplies free-form
// detail text and one careless string puts a credential in the artifact.
//
// Two layers do the work here:
//
//  1. Structural: the bundle's typed fields cannot hold a secret. Component IDs,
//     versions, digests and reason codes are constrained or derived, never
//     copied from a caller's free text.
//  2. Textual: every free-form `Detail` is passed through RedactDetail, which
//     removes credential-shaped material. This is defence in depth against the
//     one field that IS caller-supplied prose.
//
// A redaction that only recognizes the exact fixture string it was tested with
// is worthless, so the patterns below cover credential SHAPES (bearer tokens,
// key prefixes with high-entropy tails, userinfo in a URL, known env
// assignments) rather than enumerated values.

import (
	"regexp"
	"strings"
)

// redacted marks substituted material. It is a fixed token so a bundle reader
// can tell that something was removed rather than that the field was empty.
const redacted = "[REDACTED]"

type redactionRule struct {
	name string
	re   *regexp.Regexp
}

// redactionRules are applied in order. Order matters where a narrower rule
// would be eaten by a broader one.
var redactionRules = []redactionRule{
	// Authorization headers and bearer tokens. Covers "Bearer <token>",
	// "Authorization: <scheme> <token>".
	{
		name: "authorization-header",
		re:   regexp.MustCompile(`(?i)\b(bearer|basic|token)\s+[A-Za-z0-9\-._~+/]{8,}=*`),
	},
	{
		name: "authorization-assignment",
		re:   regexp.MustCompile(`(?i)\bauthorization\s*[:=]\s*\S+`),
	},
	// Userinfo embedded in a URL: scheme://user:password@host. The password is
	// the whole point of the rule, so keep the scheme and host, drop the rest.
	{
		name: "url-userinfo",
		re:   regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9+.-]*://)[^/@\s:]+:[^/@\s]+@`),
	},
	// Common key prefixes with a high-entropy tail. These are shapes emitted by
	// real providers; a fixture that merely looks similar is caught too.
	{
		name: "prefixed-api-key",
		re:   regexp.MustCompile(`\b(sk|rk|pk|ghp|gho|ghs|ghu|glpat|xox[abp]|AKIA)[-_][A-Za-z0-9\-_]{12,}`),
	},
	// A long base64/hex run adjacent to a secret-ish name. This is the shape of
	// an inline credential in prose ("key=abc123...").
	//
	// The name set is broad on purpose: the first version of this rule listed
	// only SECRET/PASSWORD/TOKEN/API_KEY and a real fixture (`WEBHOOK_ENCRYPTION_KEY`)
	// walked straight through it. A redaction rule whose name list is
	// hand-enumerated fails on the next credential — the same vacuity class this
	// program keeps finding in gates. Bare `KEY` is included for that reason.
	{
		name: "secret-assignment",
		re:   regexp.MustCompile(`(?i)\b([A-Z0-9_]*(?:SECRET|PASSWORD|PASSWD|TOKEN|APIKEY|API_KEY|PRIVATE_KEY|CREDENTIAL|ENCRYPTION_KEY|KEY|HMAC|SIGNING|SALT|DSN|PASSPHRASE)[A-Z0-9_]*)\s*[:=]\s*("[^"]*"|'[^']*'|\S+)`),
	},
	// PEM private key blocks, which are multi-line and unmistakable.
	{
		name: "pem-private-key",
		re:   regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
	},
	// JWT: three base64url segments separated by dots, starting with eyJ.
	{
		name: "jwt",
		re:   regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
	},
}

// RedactDetail removes credential-shaped material from free-form text and
// bounds the result. The bound exists because an evidence bundle is meant to
// carry a bounded diagnostic, and an unbounded string is how a payload ends up
// copied into a record wholesale.
func RedactDetail(detail string) string {
	// A private key block is removed whole: keeping its header would leave a
	// marker saying "a key was here" next to stray base64 lines.
	if i := strings.Index(detail, "-----BEGIN"); i >= 0 {
		if j := strings.Index(detail[i:], "-----END"); j >= 0 {
			end := i + j
			if k := strings.Index(detail[end:], "-----"); k >= 0 {
				detail = detail[:i] + redacted + detail[end+k+len("-----"):]
			}
		}
	}

	for _, rule := range redactionRules {
		switch rule.name {
		case "url-userinfo":
			// Keep the scheme so the diagnostic still says where it pointed.
			detail = rule.re.ReplaceAllString(detail, "${1}"+redacted+"@")
		case "secret-assignment":
			// Keep the variable NAME: "which secret leaked" is the useful half.
			detail = rule.re.ReplaceAllString(detail, "${1}="+redacted)
		case "authorization-assignment":
			detail = rule.re.ReplaceAllString(detail, "authorization="+redacted)
		case "authorization-header":
			detail = rule.re.ReplaceAllString(detail, redacted)
		default:
			detail = rule.re.ReplaceAllString(detail, redacted)
		}
	}

	return BoundDetail(detail)
}

// MaxDetailLength bounds a single diagnostic string. Evidence is not a log
// stream; a very long detail is either a leaked payload or a stack trace, and
// neither belongs in a signed artifact.
const MaxDetailLength = 512

// BoundDetail truncates an over-long detail with an explicit marker so a reader
// can tell truncation from a short value.
func BoundDetail(detail string) string {
	if len(detail) <= MaxDetailLength {
		return detail
	}
	const marker = "…[truncated]"
	// Trim on a rune boundary rather than mid-byte, so the JSON stays valid
	// UTF-8 even when the cut lands inside a multi-byte character.
	cut := MaxDetailLength - len(marker)
	if cut < 0 {
		cut = 0
	}
	for cut > 0 && !utf8Start(detail[cut]) {
		cut--
	}
	return detail[:cut] + marker
}

// utf8Start reports whether b begins a UTF-8 sequence (i.e. is not a
// continuation byte).
func utf8Start(b byte) bool { return b&0xC0 != 0x80 }
