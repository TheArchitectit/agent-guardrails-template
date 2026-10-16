package middleware

import "strings"

// redactedQuery is the marker left in place of a query string in log output.
// It is deliberately visible: an operator reading a log line must be able to
// tell that a query string was present and was withheld, rather than believing
// the request carried none.
const redactedQuery = "?<redacted>"

// RedactRequestURI returns a request URI safe to write to logs: the path is
// preserved (needed for correlation and rate analysis) and the query string is
// replaced by a marker.
//
// Spec 17 R17-08 requires that credential material never reach logs. A query
// string routinely carries exactly that material — `?api_key=…`, `?token=…`,
// `?access_token=…`, a signed URL — and the raw `RequestURI` the request logger
// wrote put every one of those values into the log in cleartext. There is no
// way to tell a secret-bearing query from an ordinary one by inspection, so the
// whole query is withheld rather than filtered by parameter name (a name-based
// allowlist would fail open on the first unlisted parameter).
//
// Named limit (documented, not hidden): a secret carried in the *path* is not
// redacted here. Paths are needed for correlation, and this spec's own text
// keeps "paths to secret files" out of ordinary-caller *responses* rather than
// log correlation fields; a path-embedded credential remains the caller's
// contract to avoid.
func RedactRequestURI(rawURI string) string {
	if rawURI == "" {
		return ""
	}
	if i := strings.IndexByte(rawURI, '?'); i >= 0 {
		return rawURI[:i] + redactedQuery
	}
	return rawURI
}
