package support_http

import (
	"strings"
	"unicode"

	"github.com/ambi/idmagic/backend/shared/logging"

	"github.com/labstack/echo/v5"
)

// ProblemContentType is the media type for RFC 9457 Problem Details responses
// (ADR-154).
const ProblemContentType = "application/problem+json"

// problemTypeURNPrefix identifies idmagic-internal problem types. There is no
// published documentation page behind these URIs yet (ADR-154 defers that);
// clients are expected to switch on `code`/the URN suffix, not dereference it.
const problemTypeURNPrefix = "urn:idmagic:error:"

// Problem is the RFC 9457 Problem Details response body.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// WriteProblem writes an RFC 9457 Problem Details response (ADR-154), the
// default envelope for generic API errors (see spec/SPECIFICATION.md HTTP error
// responses). code becomes both the `type` URN suffix and, humanized, the
// stable `title`; detail carries the occurrence-specific explanation.
// instance is the request's correlation id (spec/SPECIFICATION.md request
// correlation), read from context rather than passed in so call sites cannot
// drift from the id actually returned to the client via X-Request-ID.
func WriteProblem(c *echo.Context, status int, code, detail string) error {
	c.Response().Header().Set("Content-Type", ProblemContentType)
	return NoStoreJSON(c, status, Problem{
		Type:     problemTypeURNPrefix + code,
		Title:    humanizeErrorCode(code),
		Status:   status,
		Detail:   detail,
		Instance: logging.RequestIDFromContext(c.Request().Context()),
	})
}

// humanizeErrorCode turns a snake_case error code into a short human-readable
// title, e.g. "invalid_role" -> "Invalid role".
func humanizeErrorCode(code string) string {
	words := strings.ReplaceAll(code, "_", " ")
	if words == "" {
		return words
	}
	r := []rune(words)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
