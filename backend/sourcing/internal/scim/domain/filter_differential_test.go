package domain_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"unicode/utf16"

	scimdomain "github.com/ambi/idmagic/backend/sourcing/internal/scim/domain"
)

// filterGen は RFC 7644 §3.4.2.2 の文法から式を作り、文法から外す変形を加える。
type filterGen struct {
	r     *rand.Rand
	allow scimdomain.AttributeAllowlist
	attrs []string
}

func newFilterGen(seed uint64, allow scimdomain.AttributeAllowlist) *filterGen {
	attrs := make([]string, 0, len(allow))
	for a := range allow {
		attrs = append(attrs, a)
	}
	// map の反復順に依存しないよう整列する。
	for i := range attrs {
		for j := i + 1; j < len(attrs); j++ {
			if attrs[j] < attrs[i] {
				attrs[i], attrs[j] = attrs[j], attrs[i]
			}
		}
	}
	return &filterGen{r: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), allow: allow, attrs: attrs}
}

// 生成する値の文字。ASCII と、大文字小文字を持たない文字とサロゲートペアになる文字だけを使う。
var genRunes = []rune("aAbBcC01-_.@ 日😀")

func (g *filterGen) pick(n int) int { return g.r.IntN(n) }

func (g *filterGen) value() string {
	var sb strings.Builder
	for range g.pick(4) {
		sb.WriteRune(genRunes[g.pick(len(genRunes))])
	}
	return sb.String()
}

var genDateTimes = []string{"2020-01-01T00:00:00Z", "2020-01-01T09:00:00+09:00", "2021-06-15T12:30:00Z", "2019-12-31T23:59:59-05:00"}

// jsonLiteral は値を JSON の文字列として書く。文字を `\u` で逃がす書き方も混ぜる。
func (g *filterGen) jsonLiteral(s string) string {
	if g.pick(3) > 0 {
		data, err := json.Marshal(s)
		if err != nil {
			panic(err) // string の符号化は失敗しない
		}
		return string(data)
	}
	var sb strings.Builder
	sb.WriteByte('"')
	for _, c := range s {
		switch {
		case c == '"' || c == '\\':
			sb.WriteByte('\\')
			sb.WriteRune(c)
		case c > 0xffff:
			hi, lo := utf16.EncodeRune(c)
			fmt.Fprintf(&sb, `\u%04x\u%04x`, hi, lo)
		default:
			fmt.Fprintf(&sb, `\u%04X`, c)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func (g *filterGen) attrExp() string {
	attr := g.attrs[g.pick(len(g.attrs))]
	spec := g.allow[attr]
	name := attr
	if g.pick(2) == 0 {
		name = strings.ToUpper(attr[:1]) + attr[1:]
	}
	if g.pick(6) == 0 {
		name = "urn:ietf:params:scim:schemas:core:2.0:User:" + name
	}
	ops := []string{"pr"}
	for op := range spec.Ops {
		ops = append(ops, op)
	}
	for i := range ops {
		for j := i + 1; j < len(ops); j++ {
			if ops[j] < ops[i] {
				ops[i], ops[j] = ops[j], ops[i]
			}
		}
	}
	op := ops[g.pick(len(ops))]
	if op == "pr" {
		return name + " pr"
	}
	var value string
	switch spec.Kind {
	case scimdomain.AttrBoolean:
		value = []string{"true", "false"}[g.pick(2)]
	case scimdomain.AttrDateTime:
		value = g.jsonLiteral(genDateTimes[g.pick(len(genDateTimes))])
	default:
		value = g.jsonLiteral(g.value())
	}
	return name + " " + op + " " + value
}

func (g *filterGen) filter(depth int) string {
	if depth >= 2 {
		return g.attrExp()
	}
	switch g.pick(5) {
	case 0:
		return g.filter(depth+1) + " and " + g.filter(depth+1)
	case 1:
		return g.filter(depth+1) + " or " + g.filter(depth+1)
	case 2:
		return "not (" + g.filter(depth+1) + ")"
	case 3:
		return "(" + g.filter(depth+1) + ")"
	default:
		return g.attrExp()
	}
}

// mutate は文法から外れうる変形を一つ加える。
func (g *filterGen) mutate(f string) string {
	if f == "" {
		return f
	}
	i := g.pick(len(f))
	switch g.pick(8) {
	case 0: // SP を二つにする
		return strings.Replace(f, " ", "  ", 1)
	case 1: // SP をタブにする
		return strings.Replace(f, " ", "\t", 1)
	case 2: // 一文字消す
		return f[:i] + f[i+1:]
	case 3: // リテラルと演算子の大文字
		return strings.NewReplacer("true", "TRUE", " and ", " AND ", "false", "False").Replace(f)
	case 4: // 文字列に制御文字をそのまま入れる
		return strings.Replace(f, `"`, "\"\x01", 1)
	case 5: // 符号付きの \u
		return strings.Replace(f, `"`, `"\u+041`, 1)
	case 6: // 引用符の前の SP を消す
		return strings.Replace(f, ` "`, `"`, 1)
	default: // 文法にない文字を入れる
		return f[:i] + string("[]{}'!"[g.pick(6)]) + f[i:]
	}
}

func (g *filterGen) resource() map[string]any {
	attrs := map[string]any{}
	for _, a := range g.attrs {
		if g.pick(5) == 0 {
			continue
		}
		switch g.allow[a].Kind {
		case scimdomain.AttrBoolean:
			attrs[a] = g.pick(2) == 0
		case scimdomain.AttrDateTime:
			attrs[a] = genDateTimes[g.pick(len(genDateTimes))]
		default:
			attrs[a] = g.value()
		}
	}
	return attrs
}

// normalizeLenient は、製品が宣言した ABNF からの寛容さ (RFC7644-FILTERING) を参照実装の文法へ写す。
// 文字列の外を字句に分け、字句の間を SP 一つでつなぎ直す。`(` の後と `)` の前には空白を置かない。
// 語の `true` と `false` は小文字にする。文字列の中は変えない。
func normalizeLenient(f string) string {
	isWord := func(c byte) bool {
		return c == '_' || c == '-' || c == '.' || c == ':' || c == '+' || (c >= '0' && c <= '9') || (c|0x20 >= 'a' && c|0x20 <= 'z') || c >= 0x80
	}
	var tokens []string
	for i := 0; i < len(f); {
		c := f[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '"':
			j := i + 1
			for j < len(f) && f[j] != '"' {
				if f[j] == '\\' {
					j++
				}
				j++
			}
			j = min(j+1, len(f))
			tokens = append(tokens, f[i:j])
			i = j
		case isWord(c):
			j := i
			for j < len(f) && isWord(f[j]) {
				j++
			}
			word := f[i:j]
			if strings.EqualFold(word, "true") || strings.EqualFold(word, "false") {
				word = strings.ToLower(word)
			}
			tokens = append(tokens, word)
			i = j
		default:
			tokens = append(tokens, f[i:i+1])
			i++
		}
	}
	var sb strings.Builder
	for i, tok := range tokens {
		if i > 0 && tokens[i-1] != "(" && tok != ")" {
			sb.WriteByte(' ')
		}
		sb.WriteString(tok)
	}
	return sb.String()
}

// filterDivergence は二つの実装の結果が食い違った一つの入力である。
type filterDivergence struct {
	filter string
	detail string
}

// diffFilters は生成した入力について、受理と拒否、および受理した式の評価結果を比べる。
func diffFilters(seed uint64, n int, allow scimdomain.AttributeAllowlist) []filterDivergence {
	g := newFilterGen(seed, allow)
	var out []filterDivergence
	for range n {
		f := g.filter(0)
		if g.pick(2) == 0 {
			f = g.mutate(f)
		}
		got, gotErr := scimdomain.ParseFilter(f, allow)
		want, wantErr := refParse(normalizeLenient(f), allow)
		if (gotErr == nil) != (wantErr == nil) {
			out = append(out, filterDivergence{f, fmt.Sprintf("ParseFilter error = %v, reference error = %v", gotErr, wantErr)})
			continue
		}
		if gotErr != nil {
			continue
		}
		for range 20 {
			res := g.resource()
			if got.Matches(res) != want.match(res) {
				out = append(out, filterDivergence{f, fmt.Sprintf("on %v: ParseFilter matches = %v, reference = %v", res, got.Matches(res), want.match(res))})
				break
			}
		}
	}
	return out
}

// 製品の解析器は、仕様の文面だけから書いた参照実装と、生成した入力のすべてで同じ判定をする。
//
// 入力は固定のシードで生成するので、失敗は毎回同じ入力で再現する。
//
//spec:covers RFC7644-FILTERING: 比較値の文字列は RFC 7159 の JSON の文字列として解釈する
func TestParseFilter_AgreesWithRFC7644Reference(t *testing.T) {
	for _, tc := range []struct {
		name  string
		allow scimdomain.AttributeAllowlist
	}{{"User", scimdomain.UserFilterAttributes}, {"Group", scimdomain.GroupFilterAttributes}} {
		t.Run(tc.name, func(t *testing.T) {
			for _, d := range diffFilters(422, 5000, tc.allow) {
				t.Errorf("filter %q: %s", d.filter, d.detail)
			}
		})
	}
}
