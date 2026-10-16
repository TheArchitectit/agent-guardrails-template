package notifications

import "strings"

// redactedQuery is the marker left in place of a URL query string in log and
// error output. It is deliberately visible: an operator reading a line must be
// able to tell that a query string was present and was withheld, rather than
// believing the URL carried none.
const redactedQuery = "?<redacted>"

// URL redaction for logs and error surfaces (Spec 17 R17-08).
//
// R17-08 requires that credential material never reach logs, errors, audit,
// traces, metrics or MCP/REST resources. A webhook target is operator-supplied
// and routinely carries credential material in its query string — a
// `?token=***`, `?api_key=***`, a signed URL, a shared-secret query parameter.
// The delivery path previously wrote the raw target in two places: the
// `deliver` failure log recorded `wh.URL` verbatim, and the error returned by
// `http.Client.Do` is a `*url.Error` whose text embeds the FULL URL (so the
// same secret reached the `error` attribute, the stored delivery row and the
// `get_webhook_deliveries` tool response). Verified live before the fix: a
// loopback target carrying a canary in its query wrote the canary verbatim to
// the delivery-failure log.
//
// The whole query is withheld rather than filtered by parameter name: a
// name-based allowlist fails open on the first unlisted parameter, which is the
// exact class R17-08 forbids. Scheme, host and path are preserved so the log
// line stays useful for correlation — the same rule the request logger applies
// to `RequestURI` (Spec 17 R17-08, middleware.RedactRequestURI).
//
// Named limit (documented, not hidden): a credential carried in the *path* is
// not redacted. Paths are the correlation key for a target, and this mirrors
// the request-log rule; a path-embedded credential remains the operator's
// contract to avoid.

// RedactURLsInText withholds the query string of every absolute http(s) URL in
// text. It is the single decision used by both the delivery-failure log and the
// stored/returned error message, so a future log site cannot re-introduce the
// leak by formatting a URL or an error differently.
//
// Text with no URL is returned unchanged, so an ordinary error message is not
// altered.
func RedactURLsInText(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	rest := text
	for {
		i := indexScheme(rest)
		if i < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:i])

		// A URL token runs to the first character that cannot be part of one.
		j := i
		for j < len(rest) && !isURLTokenEnd(rest[j]) {
			j++
		}
		b.WriteString(redactTokenQuery(rest[i:j]))
		rest = rest[j:]
	}
	return b.String()
}

// RedactURLForLog withholds the query string of a URL-valued log attribute.
// It is RedactURLsInText for a value that is expected to be a single URL; it is
// safe for a bare URL and for a URL embedded in surrounding text.
func RedactURLForLog(raw string) string {
	return RedactURLsInText(raw)
}

// indexScheme returns the index of the earliest http:// or https:// occurrence
// in s, or -1.
func indexScheme(s string) int {
	http := strings.Index(s, "http://")
	https := strings.Index(s, "https://")
	switch {
	case http < 0:
		return https
	case https < 0:
		return http
	case http < https:
		return http
	default:
		return https
	}
}

// isURLTokenEnd reports whether b ends a URL token. Whitespace and the quote
// characters a formatted log line or a wrapped error uses all terminate it, so
// the redactor never swallows the text around a URL.
func isURLTokenEnd(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '"', '\'', '`', ')', '>', ']', '}':
		return true
	}
	return false
}

// redactTokenQuery replaces the query string of a URL token with the marker,
// leaving scheme, host and path intact. A token with no query is unchanged.
func redactTokenQuery(token string) string {
	if i := strings.IndexByte(token, '?'); i >= 0 {
		return token[:i] + redactedQuery
	}
	return token
}
