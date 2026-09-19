package server_http_test

// 要求の種別 (family) の記録規則と、組み立てた router が実際に登録する経路との対応を
// 固定する。
//
// 種別は用途による分類であり、`route` ラベルの正規表現として運用資材の側に置く。実体を
// 資材へ置いたので、分類の抜けはコードのどの検査にも現れない。抜けた経路は集計から消える
// だけで、系列が空になるわけでもエラーが出るわけでもない。**失敗が欠落の形をしている**ため、
// 「管理系の到達率が低い」という読みが、実は分類漏れだったという取り違えを生む。ここが
// 唯一の防壁である。
//
// 資材と router を突き合わせる検査は ops_assets_test.go が同じ形で持っており、root への
// 相対パス、YAML の読み取り、空振りを落とす番人はそこから借りる。
//
// REQ-SYSTEM-001 の具体例は主張しない。EX-SYSTEM-001-01 が定めるのは OAuth2/OIDC を
// 母集団とする評価であり、種別別の観測はその母集団を変えない。被覆は同ファイルの
// TestMonitoringAssetsScrapeTheRegisteredMetricsEndpoint 側が持つ。

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"

	"github.com/goccy/go-yaml"
	"github.com/labstack/echo/v5"
)

// requestFamilies は `family` ラベルが取りうる値。順序は所見の並びを決めるだけで、意味は
// 持たない。
//
// 名前だけをここに置き、経路との対応は規則ファイルに残す。ここにあるのは語彙であって
// 2 つ目の対応表ではない。名前を持たないと、`portal` を `account` へ改名した変更が経路の
// 網羅性だけを見る検査を素通りし、ダッシュボードと文書だけが静かに空になる。
var requestFamilies = []string{
	"authentication",
	"portal",
	"management",
	"scim",
	"shared_signals",
	"operations",
}

// familyRuleAssets は種別の記録規則を置く資材。docs/design/observability/monitoring.md が
// 「判定するルールは、ローカル Docker Compose と Kubernetes の両方に同じ内容で置く」と
// 定めているので、両方を読み、両者が食い違っていることも所見にする。
var familyRuleAssets = []string{
	"infra/docker/prometheus-rules.yml",
	"infra/k8s/monitoring/prometheus-rule.yaml",
}

// familyRuleDocument は記録規則のファイル 2 種を 1 つの形で受ける。Docker Compose 側は
// `groups` が根に、Kubernetes の PrometheusRule は `spec.groups` にある。
type familyRuleDocument struct {
	Groups []familyRuleGroup `yaml:"groups"`
	Spec   struct {
		Groups []familyRuleGroup `yaml:"groups"`
	} `yaml:"spec"`
}

type familyRuleGroup struct {
	Rules []struct {
		Record string            `yaml:"record"`
		Expr   string            `yaml:"expr"`
		Labels map[string]string `yaml:"labels"`
	} `yaml:"rules"`
}

func (d familyRuleDocument) groups() []familyRuleGroup {
	return append(slices.Clone(d.Groups), d.Spec.Groups...)
}

// familyRouteMatcher は PromQL の式から `route=~"..."` の中身を取り出す。種別の正規表現に
// 脱出していない二重引用符は現れないので、閉じ引用符までを素直に読む。
var familyRouteMatcher = regexp.MustCompile(`route=~"((?:[^"\\]|\\.)*)"`)

// familyRoutePattern は取り出した字面を、Prometheus が正規表現として受け取る形へ直す。
//
// PromQL の二重引用符文字列は Go と同じ脱出を解釈するので、資材に書かれた `\\.` は
// 正規表現 `\.` になる。この 1 段を省くと、検査は資材より 1 つ多い逆斜線を持つ別の
// 正規表現を照合することになり、資材が正しいときに限って落ちる。
func familyRoutePattern(literal string) (string, error) {
	return strconv.Unquote(`"` + literal + `"`)
}

// familyPatterns は 1 つの資材が宣言する「種別 → 経路の正規表現」を返す。所見が空であれば
// その資材は自己整合している。
func familyPatterns(asset, source string) (map[string]string, []string) {
	findings := make([]string, 0)

	var document familyRuleDocument
	if err := yaml.Unmarshal([]byte(source), &document); err != nil {
		return nil, []string{fmt.Sprintf("%s: the recording rules do not parse: %v", asset, err)}
	}

	patterns := map[string]string{}
	for _, group := range document.groups() {
		for _, rule := range group.Rules {
			family, declared := rule.Labels["family"]
			if !declared {
				continue
			}
			if !slices.Contains(requestFamilies, family) {
				findings = append(findings, fmt.Sprintf(
					"%s: rule %q labels family %q, which is not one of %s",
					asset, rule.Record, family, strings.Join(requestFamilies, ", ")))
				continue
			}
			match := familyRouteMatcher.FindStringSubmatch(rule.Expr)
			if match == nil {
				findings = append(findings, fmt.Sprintf(
					"%s: rule %q labels family %q but selects no route=~ expression, so the label claims a population the query does not restrict",
					asset, rule.Record, family))
				continue
			}
			pattern, err := familyRoutePattern(match[1])
			if err != nil {
				findings = append(findings, fmt.Sprintf(
					"%s: rule %q selects route=~%s, which is not a valid PromQL string: %v",
					asset, rule.Record, match[1], err))
				continue
			}
			if existing, seen := patterns[family]; seen && existing != pattern {
				findings = append(findings, fmt.Sprintf(
					"%s: family %q is selected by two different route expressions (%q and %q); one signal would then cover a different set of routes than the others",
					asset, family, existing, pattern))
				continue
			}
			patterns[family] = pattern
		}
	}

	for _, family := range requestFamilies {
		if _, ok := patterns[family]; !ok {
			findings = append(findings, fmt.Sprintf(
				"%s: no recording rule labels family %q, so nothing observes its routes",
				asset, family))
		}
	}
	return patterns, findings
}

// familyCoverageFindings は、組み立て済みの経路がちょうど 1 つの種別に当たることを確かめる。
//
// 漏れと重なりは別の誤りなので別の所見にする。漏れた経路は集計から消え、重なった経路は
// 2 つの種別に数えられる。後者は種別の合計が全体を超えるので合計と比べれば気づけるが、
// 比べるのは人である。
func familyCoverageFindings(t *testing.T, asset string, patterns map[string]string) []string {
	t.Helper()
	findings := make([]string, 0)

	compiled := map[string]*regexp.Regexp{}
	for _, family := range requestFamilies {
		pattern, ok := patterns[family]
		if !ok {
			continue
		}
		// Prometheus の `=~` は両端が固定されるので、同じ条件で比較する。
		expression, err := regexp.Compile("^(?:" + pattern + ")$")
		if err != nil {
			findings = append(findings, fmt.Sprintf(
				"%s: the route expression for family %q does not compile: %v", asset, family, err))
			continue
		}
		compiled[family] = expression
	}

	matched := map[string]int{}
	seen := map[string]struct{}{}
	routes := 0
	for _, route := range assembledRoutePatterns(t) {
		if _, duplicate := seen[route]; duplicate {
			continue
		}
		seen[route] = struct{}{}
		routes++

		hits := make([]string, 0, 1)
		for _, family := range requestFamilies {
			if expression, ok := compiled[family]; ok && expression.MatchString(route) {
				hits = append(hits, family)
				matched[family]++
			}
		}
		switch len(hits) {
		case 0:
			findings = append(findings, fmt.Sprintf(
				"%s: route %q matches no family, so it vanishes from every by-family series without warning",
				asset, route))
		case 1:
		default:
			findings = append(findings, fmt.Sprintf(
				"%s: route %q matches %d families (%s), so it is counted more than once and the family totals exceed the whole",
				asset, route, len(hits), strings.Join(hits, ", ")))
		}
	}
	if routes == 0 {
		return []string{fmt.Sprintf("%s: no route was assembled; the comparison would pass vacuously", asset)}
	}

	for _, family := range requestFamilies {
		if _, ok := compiled[family]; ok && matched[family] == 0 {
			findings = append(findings, fmt.Sprintf(
				"%s: family %q matches no assembled route; its series stays empty and nothing says why",
				asset, family))
		}
	}
	return findings
}

// assembledRoutePatterns は組み立てた router が登録したルートパターンを返す。`route` ラベル
// には MetricsMiddleware が c.Path() をそのまま入れるので、/realms/:tenant_id 接頭辞を
// 落とさない。落とすと、接頭辞付きの形だけが分類から漏れた状態を見逃す。
func assembledRoutePatterns(t *testing.T) []string {
	t.Helper()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{})
	patterns := make([]string, 0, len(e.Router().Routes()))
	for _, route := range e.Router().Routes() {
		// echo v5.3+ がグループごとに自動登録する暗黙の not-found 経路は、API の操作では
		// なく router の内部事情なので数えない。
		if route.Method == echo.RouteNotFound {
			continue
		}
		patterns = append(patterns, route.Path)
	}
	return patterns
}

// TestRequestFamilyRulesCoverEveryAssembledRoute は、2 つの規則ファイルの種別が、組み立て
// 済みの経路をちょうど 1 つずつ覆い、かつ互いに一致することを確かめる。
func TestRequestFamilyRulesCoverEveryAssembledRoute(t *testing.T) {
	t.Parallel()

	declared := make(map[string]map[string]string, len(familyRuleAssets))
	findings := make([]string, 0)
	for _, asset := range familyRuleAssets {
		patterns, assetFindings := familyPatterns(asset, string(readAsset(t, asset)))
		findings = append(findings, assetFindings...)
		findings = append(findings, familyCoverageFindings(t, asset, patterns)...)
		declared[asset] = patterns
	}

	// 資材ごとに網羅していても、互いに違う分類であれば、どちらの配備で読んだかによって
	// 同じ名前の系列が別の母集団を指す。
	reference := familyRuleAssets[0]
	for _, asset := range familyRuleAssets[1:] {
		for _, family := range requestFamilies {
			want, declaredThere := declared[reference][family]
			got, declaredHere := declared[asset][family]
			if !declaredThere || !declaredHere || want == got {
				continue
			}
			findings = append(findings, fmt.Sprintf(
				"%s and %s select family %q differently (%q and %q); the same series name would cover a different population per deployment",
				reference, asset, family, want, got))
		}
	}

	for _, finding := range findings {
		t.Error(finding)
	}
}

// TestRequestFamilyFindingsNameEachKindOfDrift は、上の検査が区別すべき 5 つの崩れ方を
// それぞれ別の所見として報告することを確かめる。
//
// 所見を 1 種類にまとめた実装でも網羅性の検査は通る。通ったうえで、運用者は「どの経路が
// 漏れたのか」にも「どちらのファイルがずれたのか」にも答えられなくなる。ここが、検査が
// 検査として役に立つ条件を固定する側である。
func TestRequestFamilyFindingsNameEachKindOfDrift(t *testing.T) {
	t.Parallel()

	// 全経路を 1 つの種別へ入れる正規表現。個々の事例は、ここから 1 か所だけ崩す。
	const everything = `.*`

	rules := func(entries ...[2]string) string {
		var b strings.Builder
		b.WriteString("groups:\n  - name: idmagic-request-families\n    rules:\n")
		for i, entry := range entries {
			fmt.Fprintf(&b, "      - record: idmagic:test_%d\n", i)
			fmt.Fprintf(&b, "        labels: { family: %s }\n", entry[0])
			fmt.Fprintf(&b, "        expr: sum(rate(http_requests_total{route=~\"%s\"}[5m]))\n", entry[1])
		}
		return b.String()
	}
	// すべての種別を宣言した健全な土台。以降の事例はここから 1 か所だけ崩す。
	sound := make([][2]string, 0, len(requestFamilies))
	for _, family := range requestFamilies {
		pattern := "/never-registered-" + family
		if family == "authentication" {
			pattern = everything
		}
		sound = append(sound, [2]string{family, pattern})
	}

	t.Run("unknown family", func(t *testing.T) {
		t.Parallel()
		_, findings := familyPatterns("asset.yml", rules([2]string{"accounts", everything}))
		requireFinding(t, findings, `labels family "accounts", which is not one of`)
	})

	t.Run("family with no rule", func(t *testing.T) {
		t.Parallel()
		_, findings := familyPatterns("asset.yml", rules([2]string{"portal", everything}))
		requireFinding(t, findings, `no recording rule labels family "management"`)
	})

	t.Run("one family selected two ways", func(t *testing.T) {
		t.Parallel()
		_, findings := familyPatterns("asset.yml",
			rules([2]string{"portal", "/api/account/v1/.*"}, [2]string{"portal", "/api/account/.*"}))
		requireFinding(t, findings, `family "portal" is selected by two different route expressions`)
	})

	t.Run("route in no family", func(t *testing.T) {
		t.Parallel()
		patterns, _ := familyPatterns("asset.yml", rules([2]string{"authentication", "/token"}))
		requireFinding(t, familyCoverageFindings(t, "asset.yml", patterns), "matches no family")
	})

	t.Run("route in two families", func(t *testing.T) {
		t.Parallel()
		patterns, _ := familyPatterns("asset.yml",
			rules([2]string{"authentication", everything}, [2]string{"portal", everything}))
		requireFinding(t, familyCoverageFindings(t, "asset.yml", patterns), "matches 2 families")
	})

	t.Run("family matching nothing", func(t *testing.T) {
		t.Parallel()
		patterns, _ := familyPatterns("asset.yml", rules(sound...))
		requireFinding(t, familyCoverageFindings(t, "asset.yml", patterns), `family "portal" matches no assembled route`)
	})
}

func requireFinding(t *testing.T, findings []string, want string) {
	t.Helper()
	for _, finding := range findings {
		if strings.Contains(finding, want) {
			return
		}
	}
	t.Fatalf("no finding mentions %q; got %v", want, findings)
}
