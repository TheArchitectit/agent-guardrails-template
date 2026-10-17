package web

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// declaredPreflightOnly enforces Spec 12 Sec. 3.3.4: a CORS preflight (OPTIONS) is
// answered by the CORS middleware only for a DECLARED route.
//
// The Web UI mounts a catch-all static handler (echo.Static("/", …) plus the
// SPA fallback GET /web/*). Echo's router therefore reports every path as a
// match, and its default OPTIONS handling answers any such path with
// 204 + `Allow` — so an OPTIONS request to an undeclared path would be served
// as if it were a preflight for a real route, i.e. OPTIONS would "declare"
// arbitrary paths and bypass the normal 404/405 outcome (the request never
// reaches authentication, which is the correct behaviour for a genuine
// preflight but not for a path that does not exist).
//
// This guard runs before the CORS middleware and acts only on OPTIONS. A
// request whose matched route pattern is a catch-all (contains '*') matched
// only the file-serving / SPA-shell fallback and is never a real preflight
// target, so it is refused with 404 and never reaches CORS. Every other
// OPTIONS request is left untouched for the CORS middleware to answer.
func declaredPreflightOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodOptions {
				return next(c)
			}
			// c.Path() is the matched route pattern, set by the router before
			// the middleware chain runs. A catch-all pattern means the path
			// matched only the static/SPA file fallback, not a declared route.
			if pattern := c.Path(); pattern != "" && strings.Contains(pattern, "*") {
				return echo.NewHTTPError(http.StatusNotFound, "no such route")
			}
			return next(c)
		}
	}
}
