package web

import (
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

// defaultWebRoot is the Web UI asset root shipped in the container image
// (mcp-server/deploy/Dockerfile: `COPY --from=builder /app/web /app/web`).
const defaultWebRoot = "/app/web"

// newSPAHandler serves the Web UI shell and its real asset files (Spec 12 §3.1
// web-shell rules; Spec 19 R19-09 path/method-bypass controls).
//
// Confinement (Spec 19 R19-09). The request path is re-rooted under the web
// root and opened through an fs.FS subtree, so no request path can name a file
// outside that root. The previous implementation concatenated the RAW request
// path onto the root and called `c.File`, so `/web/../<file>` and
// `/web/..%2f<file>` served any file the process could read. Verified live
// before this fix: an unauthenticated `GET /web/../SECRET.txt` returned the
// file. The route is declared public in the exposure registry (Spec 12 §3.1
// web-spa-fallback), so the bypass needed no credential at all.
//
// Behaviour:
//   - A request whose cleaned name is a real, non-directory file under root is
//     served.
//   - Every other request (a client-side route, a missing file, or a name that
//     normalises away) falls back to the SPA shell `index.html`.
//
// `os.DirFS` is used rather than the request path: its `Open` rejects any name
// that is not an `fs.ValidPath` (no `..`, no leading slash), so a traversal
// cannot escape the root even if the cleaning below were removed — the two
// layers are independent.
func newSPAHandler(root string) echo.HandlerFunc {
	return func(c echo.Context) error {
		rel := strings.TrimPrefix(c.Request().URL.Path, "/web")

		// Re-root: path.Clean removes "." and ".." segments and the leading
		// slash re-anchors the result at the root, so a traversal cannot climb
		// above it (`/web/../SECRET.txt` -> `/SECRET.txt` -> `SECRET.txt`,
		// which resolves under root, never at root's parent).
		name := strings.TrimPrefix(path.Clean("/"+rel), "/")
		if name == "" || name == "." {
			name = "index.html"
		}
		if !fs.ValidPath(name) {
			return echo.ErrNotFound
		}

		sub := os.DirFS(root)
		if fi, err := fs.Stat(sub, name); err == nil && !fi.IsDir() {
			return echo.StaticFileHandler(name, sub)(c)
		}
		return echo.StaticFileHandler("index.html", sub)(c)
	}
}
