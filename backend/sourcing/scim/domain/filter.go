package domain

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Resource limits guard the RFC 7644 §3.4.2.2 filter parser against
// pathological input (see wi-238 Risk Notes: unbounded filters invite
// excessive computation).
const (
	MaxFilterLength = 1000
	MaxFilterDepth  = 10
	// MaxResults is the page size ceiling ServiceProviderConfig advertises
	// via filter.maxResults and that ListScimUsers/ListScimGroups enforce.
	MaxResults = 100
)

// FilterError signals a filter that is syntactically invalid, refers to an
// attribute/operator outside the resource's allowlist, or exceeds a resource
// limit. Callers map it to HTTP 400 with scimType "invalidFilter"
// (RFC 7644 §3.12).
type FilterError struct {
	msg string
}

func (e *FilterError) Error() string { return e.msg }

func newFilterError(format string, args ...any) *FilterError {
	return &FilterError{msg: fmt.Sprintf(format, args...)}
}

// AttributeKind is the comparison type of an allowlisted filter attribute.
type AttributeKind int

const (
	AttrString AttributeKind = iota
	AttrBoolean
	// AttrDateTime attributes compare as RFC3339 real time instants (parsed
	// via time.Parse), not string lexical order, so differing offset
	// notations for the same instant compare equal (wi-244).
	AttrDateTime
)

// AttributeSpec allowlists a single filter attribute and the comparison
// operators it supports. "pr" (presence) is implicitly supported for every
// allowlisted attribute and is not part of Ops.
type AttributeSpec struct {
	Kind AttributeKind
	Ops  map[string]bool
}

// AttributeAllowlist maps a lower-cased attribute path (e.g. "username",
// "name.givenname") to the comparisons it supports. Attributes absent from
// the map are rejected with a *FilterError.
type AttributeAllowlist map[string]AttributeSpec

func stringAttr(ops ...string) AttributeSpec {
	set := make(map[string]bool, len(ops))
	for _, op := range ops {
		set[op] = true
	}
	return AttributeSpec{Kind: AttrString, Ops: set}
}

func boolAttr(ops ...string) AttributeSpec {
	set := make(map[string]bool, len(ops))
	for _, op := range ops {
		set[op] = true
	}
	return AttributeSpec{Kind: AttrBoolean, Ops: set}
}

func dateTimeAttr(ops ...string) AttributeSpec {
	set := make(map[string]bool, len(ops))
	for _, op := range ops {
		set[op] = true
	}
	return AttributeSpec{Kind: AttrDateTime, Ops: set}
}

// UserFilterAttributes is the allowlist for ListScimUsers filters
// (SCL interfaces.ListScimUsers).
var UserFilterAttributes = AttributeAllowlist{
	"username":          stringAttr("eq", "ne", "co", "sw", "ew"),
	"active":            boolAttr("eq", "ne"),
	"name.formatted":    stringAttr("eq", "ne", "co", "sw", "ew"),
	"name.givenname":    stringAttr("eq", "ne", "co", "sw", "ew"),
	"name.familyname":   stringAttr("eq", "ne", "co", "sw", "ew"),
	"emails.value":      stringAttr("eq", "ne", "co", "sw", "ew"),
	"id":                stringAttr("eq", "ne"),
	"meta.created":      dateTimeAttr("eq", "ne", "gt", "ge", "lt", "le"),
	"meta.lastmodified": dateTimeAttr("eq", "ne", "gt", "ge", "lt", "le"),
}

// GroupFilterAttributes is the allowlist for ListScimGroups filters
// (SCL interfaces.ListScimGroups).
var GroupFilterAttributes = AttributeAllowlist{
	"displayname":       stringAttr("eq", "ne", "co", "sw", "ew"),
	"id":                stringAttr("eq", "ne"),
	"meta.created":      dateTimeAttr("eq", "ne", "gt", "ge", "lt", "le"),
	"meta.lastmodified": dateTimeAttr("eq", "ne", "gt", "ge", "lt", "le"),
}

// FilterExpr is a parsed, semantically-validated SCIM filter expression. It
// evaluates against a flattened, lower-cased attribute map built by the
// caller (e.g. usecases.userFilterAttrs).
type FilterExpr interface {
	Matches(attrs map[string]any) bool
}

type andExpr struct{ left, right FilterExpr }

func (e *andExpr) Matches(attrs map[string]any) bool {
	return e.left.Matches(attrs) && e.right.Matches(attrs)
}

type orExpr struct{ left, right FilterExpr }

func (e *orExpr) Matches(attrs map[string]any) bool {
	return e.left.Matches(attrs) || e.right.Matches(attrs)
}

type notExpr struct{ inner FilterExpr }

func (e *notExpr) Matches(attrs map[string]any) bool {
	return !e.inner.Matches(attrs)
}

type presenceExpr struct{ attr string }

func (e *presenceExpr) Matches(attrs map[string]any) bool {
	v, ok := attrs[e.attr]
	if !ok || v == nil {
		return false
	}
	if s, isStr := v.(string); isStr {
		return s != ""
	}
	return true
}

type compareExpr struct {
	attr string
	op   string
	kind AttributeKind
	str  string
	b    bool
	t    time.Time
}

func (e *compareExpr) Matches(attrs map[string]any) bool {
	v, ok := attrs[e.attr]
	if !ok || v == nil {
		return false
	}
	if e.kind == AttrBoolean {
		bv, isBool := v.(bool)
		if !isBool {
			return false
		}
		switch e.op {
		case "eq":
			return bv == e.b
		case "ne":
			return bv != e.b
		}
		return false
	}
	if e.kind == AttrDateTime {
		sv, isStr := v.(string)
		if !isStr {
			return false
		}
		tv, err := time.Parse(time.RFC3339, sv)
		if err != nil {
			return false
		}
		switch e.op {
		case "eq":
			return tv.Equal(e.t)
		case "ne":
			return !tv.Equal(e.t)
		case "gt":
			return tv.After(e.t)
		case "ge":
			return !tv.Before(e.t)
		case "lt":
			return tv.Before(e.t)
		case "le":
			return !tv.After(e.t)
		}
		return false
	}

	sv, isStr := v.(string)
	if !isStr {
		return false
	}
	lsv := strings.ToLower(sv)
	lref := strings.ToLower(e.str)
	switch e.op {
	case "eq":
		return lsv == lref
	case "ne":
		return lsv != lref
	case "co":
		return strings.Contains(lsv, lref)
	case "sw":
		return strings.HasPrefix(lsv, lref)
	case "ew":
		return strings.HasSuffix(lsv, lref)
	}
	return false
}

// ParseFilter parses and semantically validates a SCIM filter expression per
// RFC 7644 §3.4.2.2, closing over the given attribute allowlist. Attributes,
// operators, and syntax outside that allowlist, and filters exceeding the
// resource limits (MaxFilterLength, MaxFilterDepth), fail with *FilterError.
func ParseFilter(filter string, allow AttributeAllowlist) (FilterExpr, error) {
	if filter == "" {
		return nil, newFilterError("filter must not be empty")
	}
	if len(filter) > MaxFilterLength {
		return nil, newFilterError("filter exceeds maximum length of %d", MaxFilterLength)
	}

	p := &parser{lex: newLexer(filter), allow: allow}
	if err := p.advance(); err != nil {
		return nil, err
	}
	expr, err := p.parseOr(0)
	if err != nil {
		return nil, err
	}
	if p.cur.kind != tokEOF {
		return nil, newFilterError("unexpected trailing input in filter")
	}
	return expr, nil
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokLParen
	tokRParen
	tokIdent
	tokString
	tokNumber
)

type token struct {
	kind tokenKind
	text string
}

type lexer struct {
	input []rune
	pos   int
}

func newLexer(s string) *lexer {
	return &lexer{input: []rune(s)}
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
		l.pos++
	}
}

func (l *lexer) next() (token, error) {
	l.skipSpace()
	if l.pos >= len(l.input) {
		return token{kind: tokEOF}, nil
	}

	c := l.input[l.pos]
	switch {
	case c == '(':
		l.pos++
		return token{kind: tokLParen}, nil
	case c == ')':
		l.pos++
		return token{kind: tokRParen}, nil
	case c == '"':
		return l.lexString()
	case isIdentStart(c):
		return l.lexIdent(), nil
	case c == '-' || unicode.IsDigit(c):
		return l.lexNumber(), nil
	default:
		return token{}, newFilterError("unexpected character %q in filter", c)
	}
}

func (l *lexer) lexIdent() token {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.pos++
	}
	return token{kind: tokIdent, text: string(l.input[start:l.pos])}
}

func (l *lexer) lexNumber() token {
	start := l.pos
	if l.input[l.pos] == '-' {
		l.pos++
	}
	for l.pos < len(l.input) && (unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.pos++
	}
	return token{kind: tokNumber, text: string(l.input[start:l.pos])}
}

func (l *lexer) lexString() (token, error) {
	l.pos++ // opening quote
	var sb strings.Builder
	for {
		if l.pos >= len(l.input) {
			return token{}, newFilterError("unterminated string literal in filter")
		}
		c := l.input[l.pos]
		if c == '"' {
			l.pos++
			return token{kind: tokString, text: sb.String()}, nil
		}
		if c == '\\' {
			l.pos++
			if l.pos >= len(l.input) {
				return token{}, newFilterError("unterminated escape sequence in filter")
			}
			esc := l.input[l.pos]
			switch esc {
			case '"':
				sb.WriteRune('"')
			case '\\':
				sb.WriteRune('\\')
			case '/':
				sb.WriteRune('/')
			case 'b':
				sb.WriteRune('\b')
			case 'f':
				sb.WriteRune('\f')
			case 'n':
				sb.WriteRune('\n')
			case 'r':
				sb.WriteRune('\r')
			case 't':
				sb.WriteRune('\t')
			case 'u':
				if l.pos+4 >= len(l.input) {
					return token{}, newFilterError("invalid unicode escape in filter")
				}
				hex := string(l.input[l.pos+1 : l.pos+5])
				n, err := strconv.ParseInt(hex, 16, 32)
				if err != nil {
					return token{}, newFilterError("invalid unicode escape in filter")
				}
				sb.WriteRune(rune(n))
				l.pos += 4
			default:
				return token{}, newFilterError("invalid escape sequence \\%c in filter", esc)
			}
			l.pos++
			continue
		}
		sb.WriteRune(c)
		l.pos++
	}
}

func isIdentStart(c rune) bool {
	return unicode.IsLetter(c) || c == '_'
}

func isIdentPart(c rune) bool {
	// ':' is included so schema URN-prefixed attribute paths (RFC 7644
	// §3.4.2.2, e.g. "urn:ietf:params:scim:schemas:core:2.0:User:userName")
	// lex as a single identifier; parseAttrExpr strips the prefix (wi-244).
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '.' || c == ':'
}

type parser struct {
	lex   *lexer
	cur   token
	allow AttributeAllowlist
}

func (p *parser) advance() error {
	t, err := p.lex.next()
	if err != nil {
		return err
	}
	p.cur = t
	return nil
}

func (p *parser) isKeyword(kw string) bool {
	return p.cur.kind == tokIdent && strings.EqualFold(p.cur.text, kw)
}

func (p *parser) parseOr(depth int) (FilterExpr, error) {
	if depth > MaxFilterDepth {
		return nil, newFilterError("filter exceeds maximum nesting depth of %d", MaxFilterDepth)
	}
	left, err := p.parseAnd(depth + 1)
	if err != nil {
		return nil, err
	}
	for p.isKeyword("or") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseAnd(depth + 1)
		if err != nil {
			return nil, err
		}
		left = &orExpr{left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd(depth int) (FilterExpr, error) {
	if depth > MaxFilterDepth {
		return nil, newFilterError("filter exceeds maximum nesting depth of %d", MaxFilterDepth)
	}
	left, err := p.parseNot(depth + 1)
	if err != nil {
		return nil, err
	}
	for p.isKeyword("and") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseNot(depth + 1)
		if err != nil {
			return nil, err
		}
		left = &andExpr{left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseNot(depth int) (FilterExpr, error) {
	if depth > MaxFilterDepth {
		return nil, newFilterError("filter exceeds maximum nesting depth of %d", MaxFilterDepth)
	}
	if p.isKeyword("not") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.cur.kind != tokLParen {
			return nil, newFilterError(`"not" must be followed by "(" per RFC 7644 §3.4.2.2`)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		inner, err := p.parseOr(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.cur.kind != tokRParen {
			return nil, newFilterError(`expected closing ")"`)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &notExpr{inner: inner}, nil
	}
	return p.parsePrimary(depth)
}

func (p *parser) parsePrimary(depth int) (FilterExpr, error) {
	if p.cur.kind == tokLParen {
		if err := p.advance(); err != nil {
			return nil, err
		}
		expr, err := p.parseOr(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.cur.kind != tokRParen {
			return nil, newFilterError(`expected closing ")"`)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		return expr, nil
	}
	return p.parseAttrExpr()
}

func (p *parser) parseAttrExpr() (FilterExpr, error) {
	if p.cur.kind != tokIdent {
		return nil, newFilterError("expected attribute name in filter")
	}
	attrRaw := p.cur.text
	attr, ok := stripSchemaURNPrefix(strings.ToLower(attrRaw))
	if !ok {
		return nil, newFilterError("unknown schema URN prefix in attribute path %q", attrRaw)
	}
	if !isValidAttrPath(attr) {
		return nil, newFilterError("invalid attribute path %q", attrRaw)
	}
	spec, ok := p.allow[attr]
	if !ok {
		return nil, newFilterError("attribute %q is not filterable", attrRaw)
	}
	if err := p.advance(); err != nil {
		return nil, err
	}

	if p.isKeyword("pr") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &presenceExpr{attr: attr}, nil
	}

	if p.cur.kind != tokIdent {
		return nil, newFilterError("expected comparison operator after attribute %q", attrRaw)
	}
	op := strings.ToLower(p.cur.text)
	if !isComparisonOp(op) {
		return nil, newFilterError("unknown operator %q", p.cur.text)
	}
	if !spec.Ops[op] {
		return nil, newFilterError("operator %q is not supported for attribute %q", op, attrRaw)
	}
	if err := p.advance(); err != nil {
		return nil, err
	}

	switch spec.Kind {
	case AttrBoolean:
		if !p.isKeyword("true") && !p.isKeyword("false") {
			return nil, newFilterError("expected boolean value for attribute %q", attrRaw)
		}
		b := p.isKeyword("true")
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &compareExpr{attr: attr, op: op, kind: AttrBoolean, b: b}, nil
	case AttrDateTime:
		if p.cur.kind != tokString {
			return nil, newFilterError("expected quoted dateTime value for attribute %q", attrRaw)
		}
		val := p.cur.text
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return nil, newFilterError("invalid dateTime literal %q for attribute %q", val, attrRaw)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &compareExpr{attr: attr, op: op, kind: AttrDateTime, t: t}, nil
	default:
		if p.cur.kind != tokString {
			return nil, newFilterError("expected quoted string value for attribute %q", attrRaw)
		}
		val := p.cur.text
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &compareExpr{attr: attr, op: op, kind: AttrString, str: val}, nil
	}
}

func isComparisonOp(op string) bool {
	return slices.Contains([]string{"eq", "ne", "co", "sw", "ew", "gt", "ge", "lt", "le"}, op)
}

// scimURNPrefixes are the RFC 7644 §3.4.2.2 schema URN prefixes accepted
// before an attribute name. Recognized generically regardless of which
// allowlist a given ParseFilter call closes over (wi-244): the stripped
// attribute name is still resolved through that allowlist, so a User-schema
// prefix on a Group-only attribute (or vice versa) is rejected downstream by
// the normal "not filterable" check, not by prefix matching.
var scimURNPrefixes = []string{
	"urn:ietf:params:scim:schemas:core:2.0:user:",
	"urn:ietf:params:scim:schemas:core:2.0:group:",
}

// stripSchemaURNPrefix removes a recognized schema URN prefix from a
// lower-cased attribute path. ok is false if attr contains a ':' that
// doesn't match any known prefix (invalidFilter).
func stripSchemaURNPrefix(attr string) (string, bool) {
	if !strings.Contains(attr, ":") {
		return attr, true
	}
	for _, prefix := range scimURNPrefixes {
		if stripped, found := strings.CutPrefix(attr, prefix); found {
			return stripped, true
		}
	}
	return "", false
}

func isValidAttrPath(attr string) bool {
	parts := strings.Split(attr, ".")
	if len(parts) > 2 {
		return false
	}
	return !slices.Contains(parts, "")
}
