package support_http

import (
	"strings"
	"unicode"

	"github.com/ambi/idmagic/backend/shared/logging"

	"github.com/labstack/echo/v5"
)

// ProblemContentType is the media type for RFC 9457 Problem Details responses.
const ProblemContentType = "application/problem+json"

// problemTypeURNPrefix identifies idmagic-internal problem types. There is no
// published documentation page behind these URIs yet (defers that);
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

// WriteProblem writes an RFC 9457 Problem Details response, the
// default envelope for generic API errors (see docs/design/application/api-guidelines.md
// 「エラーレスポンスの形式」). code becomes both the `type` URN suffix and, humanized, the
// stable `title`; detail carries the occurrence-specific explanation.
// instance is the request's correlation id (docs/design/observability/README.md request
// correlation), read from context rather than passed in so call sites cannot
// drift from the id actually returned to the client via X-Request-ID.
func WriteProblem(c *echo.Context, status int, code, detail string) error {
	return WriteProblemBody(c, status, NewProblem(c, status, code, detail))
}

// NewProblem は WriteProblem と同じ Problem Details を作る。拡張の欄を持つ問題は、これを埋め込んだ
// 型を WriteProblemBody で書く。
func NewProblem(c *echo.Context, status int, code, detail string) Problem {
	return Problem{
		Type:     problemTypeURNPrefix + code,
		Title:    humanizeErrorCode(code),
		Status:   status,
		Detail:   detail,
		Instance: logging.RequestIDFromContext(c.Request().Context()),
	}
}

// WriteProblemBody は、Problem を埋め込んだ body を Problem Details として書く。
func WriteProblemBody(c *echo.Context, status int, body any) error {
	c.Response().Header().Set("Content-Type", ProblemContentType)
	return NoStoreJSON(c, status, body)
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
