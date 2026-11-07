package spec

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
	"uuid"
)

// Issues は検証で見つかった違反の並びである。All が一つの error へまとめる。
type Issues []issue

type issue struct {
	path    string
	message string
	length  bool
}

// Rule は値が満たすべき条件である。一つの Rule が複数の条件を持つことがあり、
// 満たさない条件ごとに違反を一つ報告する。
type Rule[T any] struct{ checks []check[T] }

type check[T any] struct {
	holds   func(T) bool
	message string
	length  bool
}

func newRule[T any](holds func(T) bool, message string) Rule[T] {
	return Rule[T]{checks: []check[T]{{holds: holds, message: message}}}
}

// WithMessage は違反の文言を message に置き換えた Rule を返す。
func (r Rule[T]) WithMessage(message string) Rule[T] {
	checks := make([]check[T], len(r.checks))
	for i, c := range r.checks {
		c.message = message
		checks[i] = c
	}
	return Rule[T]{checks: checks}
}

// Must は holds が偽を返す値を message で拒否する。
func Must[T any](holds func(T) bool, message string) Rule[T] {
	return newRule(holds, message)
}

// OneOf は options のいずれかと等しい値だけを受け付ける。
func OneOf[T comparable](options ...T) Rule[T] {
	return newRule(func(value T) bool { return slices.Contains(options, value) },
		fmt.Sprintf("string must be one of %v", options))
}

// GreaterThan は minimum より大きい数だけを受け付ける。
func GreaterThan(minimum int) Rule[int] {
	return newRule(func(value int) bool { return value > minimum },
		fmt.Sprintf("number must be greater than %d", minimum))
}

// AtLeast は minimum 以上の数だけを受け付ける。
func AtLeast(minimum int) Rule[int] {
	return newRule(func(value int) bool { return value >= minimum },
		fmt.Sprintf("number must be greater than or equal to %d", minimum))
}

// Chars は下限と上限をコードポイント数で課す。len(string) は UTF-8 のバイト数を
// 数えるので、上限 100 を len で比べると日本語では 33 文字で拒否してしまう。
// minimum が 0 のときは下限を課さない。
func Chars(minimum, maximum int) Rule[string] {
	var rule Rule[string]
	if minimum > 0 {
		rule.checks = append(rule.checks, check[string]{
			holds:   func(value string) bool { return utf8.RuneCountInString(value) >= minimum },
			message: fmt.Sprintf("must be at least %d characters", minimum),
			length:  true,
		})
	}
	rule.checks = append(rule.checks, check[string]{
		holds:   func(value string) bool { return utf8.RuneCountInString(value) <= maximum },
		message: fmt.Sprintf("must be at most %d characters", maximum),
		length:  true,
	})
	return rule
}

// CharsAtMost は上限だけをコードポイント数で課す。
func CharsAtMost(maximum int) Rule[string] { return Chars(0, maximum) }

// KeyString は索引の鍵の成分になる文字列に、契約の上限 (コードポイント) と資源の
// 上限 (バイト) を重ねて課す。コードポイントの上限だけでは、標準が許す長さのまま
// UTF-8 で btree の索引行上限を超える値を通してしまう。
func KeyString(maxChars, maxBytes int) Rule[string] {
	rule := Chars(0, maxChars)
	rule.checks = append(rule.checks, check[string]{
		holds:   func(value string) bool { return len(value) <= maxBytes },
		message: fmt.Sprintf("must be at most %d bytes", maxBytes),
		length:  true,
	})
	return rule
}

// UUID は IdMagic が採番する識別子の形、すなわち UUID v4 の正規形 (小文字、ハイフン
// 区切り) だけを受け付ける。uuid.Parse は大文字、波括弧、urn:uuid: 接頭辞、ハイフンの
// ない形も受けるので、正規形へ戻した文字列が入力と一致することを求める。
func UUID() Rule[string] {
	return newRule(func(value string) bool {
		parsed, err := uuid.Parse(value)
		const version4, rfc9562Variant = 4, 0b10
		return err == nil && parsed.String() == value &&
			parsed[6]>>4 == version4 && parsed[8]>>6 == rfc9562Variant
	}, "must be a valid UUID")
}

// IsURL は値がスキームとホストを持つ URL かを返す。
func IsURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

// URL はスキームとホストを持つ URL だけを受け付ける。
func URL() Rule[string] { return newRule(IsURL, "must be a valid URL") }

// IsEmailAddress は値が表示名を持たない addr-spec かを返す。RFC 6532 の国際化された
// アドレスも受け付ける。
func IsEmailAddress(value string) bool {
	// ParseAddress は「Name <a@example.com>」の形も受けるので、読み取った addr-spec が
	// 入力そのものであることを求める。
	address, err := mail.ParseAddress(value)
	return err == nil && address.Name == "" && address.Address == value
}

// Email はメールアドレスの書式だけを受け付ける。
func Email() Rule[string] { return newRule(IsEmailAddress, "must be a valid email") }

// Required はゼロ値を「is required」で拒否し、ゼロ値でなければ rules を課す。
// ゼロ値のときは rules を評価しない。
func Required[T comparable](field string, value T, rules ...Rule[T]) Issues {
	var zero T
	if value == zero {
		return Issues{{path: field, message: "is required"}}
	}
	return apply(field, value, rules)
}

// Optional はゼロ値を未設定として受け付け、ゼロ値でなければ rules を課す。
func Optional[T comparable](field string, value T, rules ...Rule[T]) Issues {
	var zero T
	if value == zero {
		return nil
	}
	return apply(field, value, rules)
}

// OptionalPtr は nil を未設定として受け付け、指す値には Optional と同じ規則を課す。
func OptionalPtr[T comparable](field string, value *T, rules ...Rule[T]) Issues {
	if value == nil {
		return nil
	}
	return Optional(field, *value, rules...)
}

// NonEmpty は要素のないスライスを「is required」で拒否する。
func NonEmpty[T any](field string, values []T) Issues {
	if len(values) == 0 {
		return Issues{{path: field, message: "is required"}}
	}
	return nil
}

// Each は各要素に Optional と同じ規則を課し、違反を field[i] として報告する。
func Each[T comparable](field string, values []T, rules ...Rule[T]) Issues {
	var issues Issues
	for i, value := range values {
		issues = append(issues, Optional(fmt.Sprintf("%s[%d]", field, i), value, rules...)...)
	}
	return issues
}

// EachWith は各要素を validate で検証し、違反を field[i] の下に置く。
func EachWith[T any](field string, values []T, validate func(T) Issues) Issues {
	var issues Issues
	for i, value := range values {
		issues = append(issues, Nested(fmt.Sprintf("%s[%d]", field, i), validate(value))...)
	}
	return issues
}

// Nested は入れ子の値の違反を field の下に置く。
func Nested(field string, issues Issues) Issues {
	nested := make(Issues, len(issues))
	for i, found := range issues {
		if found.path != "" {
			found.path = field + "." + found.path
		} else {
			found.path = field
		}
		nested[i] = found
	}
	return nested
}

// Condition は特定のフィールドに属さない条件を課す。holds が偽なら message を報告する。
func Condition(holds bool, message string) Issues {
	if holds {
		return nil
	}
	return Issues{{message: message}}
}

func apply[T any](field string, value T, rules []Rule[T]) Issues {
	var issues Issues
	for _, rule := range rules {
		for _, c := range rule.checks {
			if !c.holds(value) {
				issues = append(issues, issue{path: field, message: c.message, length: c.length})
			}
		}
	}
	return issues
}

// All は違反をまとめた error を返し、違反がなければ nil を返す。各違反は
// 「wire 名: 文言」の形で、辞書順に並べて「; 」で結ぶ。長さの違反を含むときは
// *LengthError を返す。
func All(groups ...Issues) error {
	var messages []string
	length := false
	for _, group := range groups {
		for _, found := range group {
			message := found.message
			if found.path != "" {
				message = fmt.Sprintf("%s: %s", snakeCase(found.path), message)
			}
			messages = append(messages, message)
			length = length || found.length
		}
	}
	if len(messages) == 0 {
		return nil
	}
	sort.Strings(messages)
	joined := strings.Join(messages, "; ")
	if length {
		return &LengthError{message: joined}
	}
	return errors.New(joined)
}
