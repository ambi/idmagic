package domain_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	scimdomain "github.com/ambi/idmagic/backend/sourcing/internal/scim/domain"
)

// このファイルは、RFC 7644 §3.4.2.2 の文面だけから書いた SCIM のフィルターの参照実装と、
// 製品の解析器 ParseFilter との差分テストを持つ。参照実装はテストの中だけで使う。
//
// 参照実装が文面から取る規則は次である。
//   - 文法は Figure 1 の ABNF。演算子の前後は SP (0x20) 一つである。
//     ただし `not` と `(` の間は、ABNF が空白を置かず、同じ節の例が空白を置くので、どちらも受理する。
//   - ABNF の引用符付きの文字列は大文字と小文字を区別しない (RFC 5234 §2.3)。比較演算子と論理演算子が該当する。
//   - compValue の false、null、true、数値、文字列は RFC 7159 の規則である。リテラルは小文字だけであり、
//     文字列は制御文字をそのまま含めず、`\u` の後は 16 進の 4 桁で、サロゲートペアは一つの文字になる。
//   - ATTRNAME は ASCII の英字で始まり、英数字、`-`、`_` が続く。副属性は一段だけである。
//   - 演算子の意味は表 3 から表 5。`and` は `or` より強く結び付く。
//   - 文字列の比較は属性の caseExact に従う (RFC 7643 §2.2)。`id` は caseExact が真 (RFC 7643 §3.1)、
//     そのほかの許可属性は偽である。
//
// 文面が決めないので製品と同じにした点は次である。
//   - どの属性と演算子を許すか。製品の方針なので許可表を共有する。
//   - 大文字と小文字を無視する比較の畳み込み。RFC は方式を定めないので、生成する値を ASCII と
//     大文字小文字を持たない文字に限り、どの方式でも結果が同じになるようにする。
//   - 属性が無いときの比較の結果。RFC は定めないので、一致しないとする。
//   - dateTime の文法。RFC 7643 §2.3.5 は xsd:dateTime を指すが、生成する値はオフセット付きの
//     RFC 3339 の形と、日時でない文字列だけにする。
//   - 長さと入れ子の深さの上限。生成する入力は上限に届かない大きさにする。

type refExpr interface {
	match(attrs map[string]any) bool
}

type (
	refAnd struct{ l, r refExpr }
	refOr  struct{ l, r refExpr }
	refNot struct{ e refExpr }
)

func (e refAnd) match(a map[string]any) bool { return e.l.match(a) && e.r.match(a) }
func (e refOr) match(a map[string]any) bool  { return e.l.match(a) || e.r.match(a) }
func (e refNot) match(a map[string]any) bool { return !e.e.match(a) }

type refCompare struct {
	attr      string
	op        string
	kind      scimdomain.AttributeKind
	caseExact bool
	str       string
	b         bool
	t         time.Time
}

func (e refCompare) match(attrs map[string]any) bool {
	v, ok := attrs[e.attr]
	if !ok || v == nil {
		return false
	}
	if e.op == "pr" {
		if s, isStr := v.(string); isStr {
			return s != ""
		}
		return true
	}
	switch e.kind {
	case scimdomain.AttrBoolean:
		b, isBool := v.(bool)
		if !isBool {
			return false
		}
		return (e.op == "eq") == (b == e.b)
	case scimdomain.AttrDateTime:
		s, isStr := v.(string)
		if !isStr {
			return false
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return false
		}
		c := t.Compare(e.t)
		switch e.op {
		case "eq":
			return c == 0
		case "ne":
			return c != 0
		case "gt":
			return c > 0
		case "ge":
			return c >= 0
		case "lt":
			return c < 0
		case "le":
			return c <= 0
		}
		return false
	default:
		s, isStr := v.(string)
		if !isStr {
			return false
		}
		ref := e.str
		if !e.caseExact {
			s, ref = strings.ToLower(s), strings.ToLower(ref)
		}
		switch e.op {
		case "eq":
			return s == ref
		case "ne":
			return s != ref
		case "co":
			return strings.Contains(s, ref)
		case "sw":
			return strings.HasPrefix(s, ref)
		case "ew":
			return strings.HasSuffix(s, ref)
		}
		return false
	}
}

// refCaseExact は RFC 7643 が caseExact を真とする許可属性である。
var refCaseExact = map[string]bool{"id": true}

// refSchemaURIs は RFC 7643 §8.7.1 の core スキーマの URI である。
var refSchemaURIs = []string{
	"urn:ietf:params:scim:schemas:core:2.0:User",
	"urn:ietf:params:scim:schemas:core:2.0:Group",
}

var (
	refAttrName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*(\.[A-Za-z][A-Za-z0-9_-]*)?$`)
	refNumber   = regexp.MustCompile(`^-?(0|[1-9]\d*)(\.\d+)?([eE][+-]?\d+)?`)
)

var errRef = errors.New("invalidFilter")

type refParser struct {
	in    string
	pos   int
	allow scimdomain.AttributeAllowlist
}

func refParse(filter string, allow scimdomain.AttributeAllowlist) (refExpr, error) {
	p := &refParser{in: filter, allow: allow}
	e, err := p.or()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.in) {
		return nil, fmt.Errorf("%w: trailing input at %d", errRef, p.pos)
	}
	return e, nil
}

// keyword は SP <kw> SP を大文字と小文字を区別せずに読む。
func (p *refParser) keyword(kw string) bool {
	n := len(kw) + 2
	if p.pos+n > len(p.in) {
		return false
	}
	if p.in[p.pos] == ' ' && strings.EqualFold(p.in[p.pos+1:p.pos+1+len(kw)], kw) && p.in[p.pos+n-1] == ' ' {
		p.pos += n
		return true
	}
	return false
}

func (p *refParser) or() (refExpr, error) {
	l, err := p.and()
	if err != nil {
		return nil, err
	}
	for p.keyword("or") {
		r, err := p.and()
		if err != nil {
			return nil, err
		}
		l = refOr{l, r}
	}
	return l, nil
}

func (p *refParser) and() (refExpr, error) {
	l, err := p.unary()
	if err != nil {
		return nil, err
	}
	for p.keyword("and") {
		r, err := p.unary()
		if err != nil {
			return nil, err
		}
		l = refAnd{l, r}
	}
	return l, nil
}

func (p *refParser) unary() (refExpr, error) {
	negate := false
	if len(p.in)-p.pos >= 3 && strings.EqualFold(p.in[p.pos:p.pos+3], "not") {
		rest := p.in[p.pos+3:]
		switch {
		case strings.HasPrefix(rest, "("):
			negate, p.pos = true, p.pos+3
		case strings.HasPrefix(rest, " ("):
			negate, p.pos = true, p.pos+4
		}
	}
	if p.pos < len(p.in) && p.in[p.pos] == '(' {
		p.pos++
		e, err := p.or()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.in) || p.in[p.pos] != ')' {
			return nil, fmt.Errorf("%w: missing )", errRef)
		}
		p.pos++
		if negate {
			return refNot{e}, nil
		}
		return e, nil
	}
	if negate {
		return nil, fmt.Errorf("%w: not without (", errRef)
	}
	return p.attrExp()
}

// token は次の SP、`)`、または入力の終わりまでを読む。
func (p *refParser) token() string {
	start := p.pos
	for p.pos < len(p.in) && p.in[p.pos] != ' ' && p.in[p.pos] != ')' {
		p.pos++
	}
	return p.in[start:p.pos]
}

func (p *refParser) sp() bool {
	if p.pos < len(p.in) && p.in[p.pos] == ' ' {
		p.pos++
		return true
	}
	return false
}

func (p *refParser) attrExp() (refExpr, error) {
	raw := p.token()
	name := raw
	for _, uri := range refSchemaURIs {
		if len(name) > len(uri) && strings.EqualFold(name[:len(uri)+1], uri+":") {
			name = name[len(uri)+1:]
			break
		}
	}
	if !refAttrName.MatchString(name) {
		return nil, fmt.Errorf("%w: attrPath %q", errRef, raw)
	}
	attr := strings.ToLower(name)
	spec, ok := p.allow[attr]
	if !ok {
		return nil, fmt.Errorf("%w: attribute %q not filterable", errRef, raw)
	}
	if !p.sp() {
		return nil, fmt.Errorf("%w: SP after attrPath", errRef)
	}
	op := strings.ToLower(p.token())
	if op == "pr" {
		return refCompare{attr: attr, op: "pr"}, nil
	}
	if !spec.Ops[op] {
		return nil, fmt.Errorf("%w: operator %q", errRef, op)
	}
	if !p.sp() {
		return nil, fmt.Errorf("%w: SP after compareOp", errRef)
	}
	value, err := p.compValue()
	if err != nil {
		return nil, err
	}
	e := refCompare{attr: attr, op: op, kind: spec.Kind, caseExact: refCaseExact[attr]}
	switch spec.Kind {
	case scimdomain.AttrBoolean:
		b, isBool := value.(bool)
		if !isBool {
			return nil, fmt.Errorf("%w: boolean expected", errRef)
		}
		e.b = b
	case scimdomain.AttrDateTime:
		s, isStr := value.(string)
		if !isStr {
			return nil, fmt.Errorf("%w: dateTime expected", errRef)
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("%w: dateTime %q", errRef, s)
		}
		e.t = t
	default:
		s, isStr := value.(string)
		if !isStr {
			return nil, fmt.Errorf("%w: string expected", errRef)
		}
		e.str = s
	}
	return e, nil
}

// compValue は RFC 7159 の false、null、true、数値、文字列を一つ読む。
func (p *refParser) compValue() (any, error) {
	rest := p.in[p.pos:]
	for _, lit := range []struct {
		text  string
		value any
	}{{"false", false}, {"true", true}, {"null", nil}} {
		if strings.HasPrefix(rest, lit.text) {
			p.pos += len(lit.text)
			return lit.value, p.valueEnd()
		}
	}
	if m := refNumber.FindString(rest); m != "" {
		p.pos += len(m)
		return json.Number(m), p.valueEnd()
	}
	if !strings.HasPrefix(rest, `"`) {
		return nil, fmt.Errorf("%w: compValue", errRef)
	}
	end := 1
	for end < len(rest) && rest[end] != '"' {
		if rest[end] == '\\' {
			end++
		}
		end++
	}
	if end >= len(rest) {
		return nil, fmt.Errorf("%w: unterminated string", errRef)
	}
	var s string
	if err := json.Unmarshal([]byte(rest[:end+1]), &s); err != nil {
		return nil, fmt.Errorf("%w: string: %w", errRef, err)
	}
	p.pos += end + 1
	return s, p.valueEnd()
}

// valueEnd は値の後が入力の終わり、SP、`)` のいずれかであることを確かめる。
func (p *refParser) valueEnd() error {
	if p.pos == len(p.in) || p.in[p.pos] == ' ' || p.in[p.pos] == ')' {
		return nil
	}
	return fmt.Errorf("%w: value followed by %q", errRef, p.in[p.pos])
}

// 参照実装の判定が文面どおりであることを、RFC 7644 §3.4.2.2 の例と文法の境界で確かめる。
// 差分テストの判定基準そのものが誤っていないかを、製品の解析器を通さずに見る。
func TestReferenceFilter_FollowsRFC7644Text(t *testing.T) {
	accept := []string{
		`userName eq "bjensen"`,
		`userName Eq "john"`,
		`Username eq "john"`,
		`name.familyName co "O'Malley"`,
		`userName sw "J"`,
		`urn:ietf:params:scim:schemas:core:2.0:User:userName sw "J"`,
		`meta.lastModified gt "2011-05-13T04:42:34Z"`,
		`meta.lastModified ge "2011-05-13T04:42:34Z"`,
		`emails.value pr`,
		`userName eq "a" and not (emails.value co "example.com" or emails.value co "example.org")`,
		`userName eq "a" and not(emails.value co "example.com")`,
		`(userName eq "a")`,
		`active eq true`,
		`userName eq "😀"`,
	}
	reject := []string{
		`userName  eq "x"`,
		"userName\teq \"x\"",
		`userName eq"x"`,
		`active eq TRUE`,
		"userName eq \"a\x01\"",
		`userName eq "\u+041"`,
		`userName eq "\x41"`,
		`userName eq null`,
		`userName eq 1`,
		`emails[type eq "work"]`,
		`( userName eq "x" )`,
		`userName eq "x" AND`,
		`not userName eq "x"`,
	}
	for _, f := range accept {
		if _, err := refParse(f, scimdomain.UserFilterAttributes); err != nil {
			t.Errorf("refParse(%q) error = %v, want accepted", f, err)
		}
	}
	for _, f := range reject {
		if _, err := refParse(f, scimdomain.UserFilterAttributes); err == nil {
			t.Errorf("refParse(%q) accepted, want invalidFilter", f)
		}
	}
	expr, _ := refParse(`userName eq "a" or userName eq "b" and active eq true`, scimdomain.UserFilterAttributes)
	if !expr.match(map[string]any{"username": "a", "active": false}) {
		t.Error("and must bind tighter than or")
	}
	idExpr, _ := refParse(`id eq "ABC"`, scimdomain.UserFilterAttributes)
	if idExpr.match(map[string]any{"id": "abc"}) {
		t.Error("id is caseExact (RFC 7643 §3.1)")
	}
	emoji, _ := refParse(`userName eq "😀"`, scimdomain.UserFilterAttributes)
	if !emoji.match(map[string]any{"username": "😀"}) {
		t.Error("a surrogate pair escape is one character (RFC 7159 §7)")
	}
}
