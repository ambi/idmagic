package support_http

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/labstack/echo/v5"
)

var ErrInvalidIfMatch = errors.New("If-Match must contain entity tags or a single wildcard")

// EntityTag は派生した値を含めた返送表現を識別する。
func EntityTag(body any) (string, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(encoded)
	return `"` + hex.EncodeToString(hash[:]) + `"`, nil
}

// MatchCondition は、HTTP の入力を読み終えた後に現在のタグへ適用する純粋な条件である。
type MatchCondition struct {
	tags     []string
	wildcard bool
	present  bool
}

func (condition MatchCondition) Present() bool { return condition.present }

func ReadIfMatch(c *echo.Context) (MatchCondition, error) {
	values := c.Request().Header.Values("If-Match")
	if len(values) == 0 {
		return MatchCondition{}, nil
	}
	tags, wildcard, err := parseIfMatch(strings.Join(values, ","))
	return MatchCondition{tags: tags, wildcard: wildcard, present: true}, err
}

// Match は、保存先が排他した最新の状態に対して呼ぶ。HTTP の読み取りと書き出しは行わない。
func (condition MatchCondition) Match(tag string, exists bool) error {
	if !condition.present {
		return nil
	}
	if condition.wildcard && exists {
		return nil
	}
	for _, expected := range condition.tags {
		if exists && expected == tag && !strings.HasPrefix(expected, "W/") {
			return nil
		}
	}
	return kernel.ErrPreconditionFailed
}

func (condition MatchCondition) Check(body any) error {
	if !condition.present {
		return nil
	}
	tag, err := EntityTag(body)
	if err != nil {
		return err
	}
	return condition.Match(tag, true)
}

// ETagJSON はタグを計算したものと同じ JSON バイト列を返す。
// PUT の変換後の表現には RFC 9110 §9.3.4 に従ってタグを付けない。
func ETagJSON(c *echo.Context, status int, body any, explicitTag ...string) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if c.Request().Method != http.MethodPut {
		tag, err := EntityTag(body)
		if err != nil {
			return err
		}
		if len(explicitTag) > 0 {
			tag = explicitTag[0]
		}
		c.Response().Header().Set("ETag", tag)
	}
	if c.Response().Header().Get("Cache-Control") == "" {
		c.Response().Header().Set("Cache-Control", "no-store")
	}
	return c.Blob(status, echo.MIMEApplicationJSON, encoded)
}

// タグ内のカンマは区切りではないため、文字列の分割ではなく entity-tag の構文を読む。
func parseIfMatch(raw string) ([]string, bool, error) {
	raw = strings.Trim(raw, " \t")
	if raw == "*" {
		return nil, true, nil
	}
	var tags []string
	for raw != "" {
		if raw[0] == ',' {
			raw = strings.TrimLeft(raw[1:], " \t")
			continue
		}
		start := raw
		raw = strings.TrimPrefix(raw, "W/")
		if raw == "" || raw[0] != '"' {
			return nil, false, ErrInvalidIfMatch
		}
		i := 1
		for i < len(raw) && raw[i] != '"' {
			if raw[i] < 0x21 || raw[i] == 0x7f {
				return nil, false, ErrInvalidIfMatch
			}
			i++
		}
		if i == len(raw) {
			return nil, false, ErrInvalidIfMatch
		}
		consumed := len(start) - len(raw) + i + 1
		tags = append(tags, start[:consumed])
		raw = strings.TrimLeft(raw[i+1:], " \t")
		if raw != "" && raw[0] != ',' {
			return nil, false, ErrInvalidIfMatch
		}
	}
	if len(tags) == 0 {
		return nil, false, ErrInvalidIfMatch
	}
	return tags, false, nil
}
