// フロントエンドゲートウェイの経路許可リストと、組み立て済みの経路表の照合
// (REQ-SYSTEM-021)。
//
// ゲートウェイはブラウザーから見える境界を同一オリジンに揃えるため、どのパスを API へ
// 中継するかを設定上の許可リストで選ぶ。許可リストは参照設定に 3 つあり、どれも手書きの
// 列挙である。列挙が経路表から外れても、ゲートウェイの最後の `handle` が SPA へ落とすので
// 応答は 404 ではなく `index.html` の 200 になる。**失敗が成功の形をしているため、
// 手で足す検査では見つからない。** ここが唯一の防壁である。
//
// 分類を経路の登録と同じパッケージへ置くのは ClassifyRoute と同じ理由による。起動時設定へ
// 置けば、どの経路を公開してよいかがデプロイごとに変わる状態ができる。
package server_http

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// GatewayExposure は、ある経路をフロントエンドゲートウェイが中継してよいかを表す。
type GatewayExposure string

const (
	// GatewayRequired は、利用者または外部の当事者がゲートウェイ越しに呼ぶ経路。
	// どの許可リストにも一致しなければ、その機能は届かない。
	GatewayRequired GatewayExposure = "required"
	// GatewayOptional は、通しても通さなくてもよい経路。
	GatewayOptional GatewayExposure = "optional"
	// GatewayForbidden は、ゲートウェイを通してはならない経路。
	GatewayForbidden GatewayExposure = "forbidden"
)

// gatewayExposureRules は明示の分類。ここに現れない経路は優先度クラスから導く。
//
// 運用経路だけが判断を要する。`/health` は現行の参照設定が既に通しており、外形監視が
// 当てる先でもある。残りの 3 つはオーケストレーターが Pod へ直接当てるので、通す通さないの
// どちらでも正しい。`/metrics` をここへ書かないのは意図である。書かなければ下のデフォルトが
// forbidden を返し、docs/domain/system/decisions.md が定める「認証の無い指標を公開入口へ
// 出さない」が、誰かが名指しし続けなくても保たれる。
var gatewayExposureRules = []struct {
	path     string
	exposure GatewayExposure
}{
	{path: "/health", exposure: GatewayRequired},
	{path: "/livez", exposure: GatewayOptional},
	{path: "/readyz", exposure: GatewayOptional},
	{path: "/startupz", exposure: GatewayOptional},
}

// ClassifyGatewayExposure は登録済みのルートパターンを公開可否へ写す。
//
// デフォルトの向きは 2 つある。優先度クラスが infrastructure または unclassified の経路は
// forbidden、それ以外は required になる。前者は「知らない運用経路を公開しない」、後者は
// 「知らない製品経路が届かないままにならない」を意味する。どちらも、名指しを忘れた側が
// 安全に倒れる向きである。
func ClassifyGatewayExposure(pattern string) GatewayExposure {
	path := normalizeRoutePattern(pattern)
	for _, rule := range gatewayExposureRules {
		if rule.path == path {
			return rule.exposure
		}
	}
	switch ClassifyRoute(pattern) {
	case support.ClassInfrastructure, support.ClassUnclassified:
		return GatewayForbidden
	default:
		return GatewayRequired
	}
}

// GatewayRouteExpectation は 1 つのルートパターンに対する期待。SamplePath は許可リストの
// 照合器へ与えるために合成した具体パスであり、パターンそのものではない。
type GatewayRouteExpectation struct {
	Pattern    string
	SamplePath string
	Exposure   GatewayExposure
}

// GatewayRouteExpectations は組み立て済みの router が登録した経路すべての期待を返す。
//
// ホスト形式と /realms/{tenant_id} 配下のパス形式を 1 行へ潰さない。潰すと
// `@realmBackend` の列挙が何とも照合されなくなる。ROUTE_PRIORITY.md が潰しているのは
// 読み手が知りたいのが分類だけだからであり、ここでは形の違いそのものが検査対象である。
func GatewayRouteExpectations() []GatewayRouteExpectation {
	seen := map[string]struct{}{}
	expectations := make([]GatewayRouteExpectation, 0)
	for _, route := range echoRoutes() {
		if _, ok := seen[route.Path]; ok {
			continue
		}
		seen[route.Path] = struct{}{}
		expectations = append(expectations, GatewayRouteExpectation{
			Pattern:    route.Path,
			SamplePath: gatewaySamplePath(route.Path),
			Exposure:   ClassifyGatewayExposure(route.Path),
		})
	}
	sort.Slice(expectations, func(i, j int) bool { return expectations[i].Pattern < expectations[j].Pattern })
	return expectations
}

// gatewaySamplePath はルートパターンから具体パスを 1 本作る。パラメータの値は許可リストの
// 照合に影響しないので、どのセグメントにも同じ占位子を入れる。
func gatewaySamplePath(pattern string) string {
	segments := strings.Split(pattern, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*") {
			segments[i] = "sample"
		}
	}
	return strings.Join(segments, "/")
}

// gatewayAllowlist は 1 つの設定ファイルが表す許可リスト。match は「この設定はこのパスを
// API へ中継するか」に答える。
type gatewayAllowlist struct {
	name  string
	match func(path string) bool
}

// CheckGatewayAllowlists は参照設定の許可リストを組み立て済みの経路表と突き合わせ、乖離を
// 返す。空の戻り値が合格である。
//
// 設定の中身を引数で受け取り、ファイルを読まない。読み取りは
// backend/cmd/idmagic-gateway-routes だけが行う。
func CheckGatewayAllowlists(caddyfile, viteConfig string) []string {
	findings := make([]string, 0)

	caddy, caddyFindings := caddyAllowlist(caddyfile)
	findings = append(findings, caddyFindings...)
	vite, viteFindings := viteProxyAllowlist(viteConfig)
	findings = append(findings, viteFindings...)

	allowlists := make([]gatewayAllowlist, 0, 2)
	if caddy != nil {
		allowlists = append(allowlists, *caddy)
	}
	if vite != nil {
		allowlists = append(allowlists, *vite)
	}
	// 読めた許可リストが無ければ、経路の照合は何も確かめない。上の findings が既に
	// 理由を持っているので、突き合わせずに返す。
	if len(allowlists) == 0 {
		return findings
	}

	for _, expectation := range GatewayRouteExpectations() {
		for _, allowlist := range allowlists {
			matched := allowlist.match(expectation.SamplePath)
			switch expectation.Exposure {
			case GatewayRequired:
				if !matched {
					findings = append(findings, fmt.Sprintf(
						"%s does not proxy %s (%s), so the route is unreachable behind the gateway and the SPA fallback answers it instead",
						allowlist.name, expectation.SamplePath, expectation.Pattern))
				}
			case GatewayForbidden:
				if matched {
					findings = append(findings, fmt.Sprintf(
						"%s proxies %s (%s), which must not be reachable through the gateway",
						allowlist.name, expectation.SamplePath, expectation.Pattern))
				}
			case GatewayOptional:
			}
		}
	}
	return findings
}

// caddyAllowlist は Caddyfile の 2 つのマッチャーを 1 つの許可リストにまとめる。どちらの
// `handle` も同じ API へ reverse_proxy するので、設定としての意味は「どちらかに一致すれば
// 中継する」である。まとめても、ホスト形式とパス形式それぞれの具体パスを問うので、片方の
// 列挙が欠けたことは依然として検出できる。
func caddyAllowlist(source string) (*gatewayAllowlist, []string) {
	findings := make([]string, 0)

	globs, ok := caddyPathMatcher(source, "@backend")
	if !ok || len(globs) == 0 {
		findings = append(findings, "frontend/Caddyfile: the @backend path matcher could not be read; the comparison would otherwise pass vacuously")
	}
	pattern, ok := caddyPathRegexpMatcher(source, "@realmBackend")
	if !ok {
		findings = append(findings, "frontend/Caddyfile: the @realmBackend path_regexp matcher could not be read; the comparison would otherwise pass vacuously")
	}
	var realm *regexp.Regexp
	if ok {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			findings = append(findings, fmt.Sprintf("frontend/Caddyfile: the @realmBackend expression does not compile: %v", err))
		} else {
			realm = compiled
		}
	}
	if len(findings) > 0 {
		return nil, findings
	}

	matchers := make([]*regexp.Regexp, 0, len(globs))
	for _, glob := range globs {
		compiled, err := compileCaddyPathGlob(glob)
		if err != nil {
			return nil, []string{fmt.Sprintf("frontend/Caddyfile: the @backend entry %q does not compile: %v", glob, err)}
		}
		matchers = append(matchers, compiled)
	}

	return &gatewayAllowlist{
		name: "frontend/Caddyfile",
		match: func(path string) bool {
			if realm.MatchString(path) {
				return true
			}
			for _, matcher := range matchers {
				if matcher.MatchString(path) {
					return true
				}
			}
			return false
		},
	}, nil
}

// caddyPathMatcher は `<name> path <glob> ...` のトークン列を返す。
func caddyPathMatcher(source, name string) ([]string, bool) {
	for _, line := range caddyLogicalLines(source) {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != name || fields[1] != "path" {
			continue
		}
		return fields[2:], true
	}
	return nil, false
}

// caddyPathRegexpMatcher は `<name> path_regexp [識別子] <式>` の式を返す。識別子は省略でき
// るので、末尾のトークンを式とみなす。
func caddyPathRegexpMatcher(source, name string) (string, bool) {
	for _, line := range caddyLogicalLines(source) {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != name || fields[1] != "path_regexp" {
			continue
		}
		return fields[len(fields)-1], true
	}
	return "", false
}

// caddyLogicalLines は行末の `\` による継続を畳んだ論理行を返す。`@backend` のパス列挙は
// 1 行 1 パスの継続行で書かれているので、畳まずにトークンは読めない。
func caddyLogicalLines(source string) []string {
	lines := make([]string, 0)
	var current strings.Builder
	for raw := range strings.SplitSeq(source, "\n") {
		line := strings.TrimSpace(raw)
		if continued, ok := strings.CutSuffix(line, `\`); ok {
			current.WriteString(continued)
			current.WriteString(" ")
			continue
		}
		current.WriteString(line)
		lines = append(lines, current.String())
		current.Reset()
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

// compileCaddyPathGlob は Caddy の `path` マッチャー 1 件を正規表現へ写す。`*` は `/` を
// またいで一致し、比較は大文字小文字を区別しない。
func compileCaddyPathGlob(glob string) (*regexp.Regexp, error) {
	parts := strings.Split(glob, "*")
	for i, part := range parts {
		parts[i] = regexp.QuoteMeta(part)
	}
	return regexp.Compile("(?i)^" + strings.Join(parts, ".*") + "$")
}

// viteProxyKey は `server.proxy` のキー 1 件。行頭の文字列リテラルとそれに続く `:` を読む。
var viteProxyKey = regexp.MustCompile(`^\s*(['"])((?:[^'"\\]|\\.)*)['"]\s*:`)

// viteProxyAllowlist は vite.config.ts の `server.proxy` を許可リストへ写す。
//
// ここだけは設定の値ではなくソースの字面を読む。字面の解釈がずれると照合は空振りし、しかも
// 成功に見えるので、キーを 1 件も取れなければ失敗させる。解釈が Vite の読む設定と一致して
// いることは frontend/src/devProxy.test.ts が反対側から固定する。
func viteProxyAllowlist(source string) (*gatewayAllowlist, []string) {
	keys, ok := viteProxyKeys(source)
	if !ok || len(keys) == 0 {
		return nil, []string{"frontend/vite.config.ts: the server.proxy block could not be read; the comparison would otherwise pass vacuously"}
	}

	prefixes := make([]string, 0, len(keys))
	patterns := make([]*regexp.Regexp, 0, len(keys))
	for _, key := range keys {
		// Vite は `^` で始まるキーを正規表現、それ以外を前方一致として扱う。
		if !strings.HasPrefix(key, "^") {
			prefixes = append(prefixes, key)
			continue
		}
		compiled, err := regexp.Compile(key)
		if err != nil {
			return nil, []string{fmt.Sprintf("frontend/vite.config.ts: the server.proxy key %q does not compile: %v", key, err)}
		}
		patterns = append(patterns, compiled)
	}

	return &gatewayAllowlist{
		name: "frontend/vite.config.ts",
		match: func(path string) bool {
			for _, prefix := range prefixes {
				if strings.HasPrefix(path, prefix) {
					return true
				}
			}
			for _, pattern := range patterns {
				if pattern.MatchString(path) {
					return true
				}
			}
			return false
		},
	}, nil
}

// viteProxyKeys は `proxy: {` から対応する `}` までのキーを返す。
func viteProxyKeys(source string) ([]string, bool) {
	start := strings.Index(source, "proxy: {")
	if start < 0 {
		return nil, false
	}
	depth := 0
	keys := make([]string, 0)
	for line := range strings.SplitSeq(source[start:], "\n") {
		if match := viteProxyKey.FindStringSubmatch(line); depth == 1 && match != nil {
			keys = append(keys, unescapeJSString(match[2]))
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			break
		}
	}
	return keys, true
}

// unescapeJSString は JavaScript の文字列リテラルから逆斜線の脱出を外す。正規表現のキーは
// `\\?` や `\\.` のように、逆斜線自身を脱出した形で書かれている。
func unescapeJSString(literal string) string {
	var out strings.Builder
	for i := 0; i < len(literal); i++ {
		if literal[i] == '\\' && i+1 < len(literal) {
			i++
		}
		out.WriteByte(literal[i])
	}
	return out.String()
}
