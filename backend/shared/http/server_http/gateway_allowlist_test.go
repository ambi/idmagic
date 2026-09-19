package server_http_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"

	"github.com/labstack/echo/v5"
)

// repositoryFile はリポジトリ直下からの相対パスで参照設定を読む。テストの作業
// ディレクトリはパッケージのディレクトリなので、そこから遡る。
func repositoryFile(t *testing.T, relative string) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "..", relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(source)
}

//spec:covers EX-SYSTEM-021-03: 通してはならない経路の既定が forbidden であること、および運用経路それぞれの分類
func TestClassifyGatewayExposureKeepsMetricsOffTheGateway(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		pattern string
		want    httpadapter.GatewayExposure
	}{
		{"/metrics", httpadapter.GatewayForbidden},
		{"/livez", httpadapter.GatewayOptional},
		{"/readyz", httpadapter.GatewayOptional},
		{"/startupz", httpadapter.GatewayOptional},
		{"/health", httpadapter.GatewayRequired},
		{"/ssf/streams/:stream_id/events", httpadapter.GatewayRequired},
		{"/realms/:tenant_id/ssf/streams/:stream_id/events", httpadapter.GatewayRequired},
		{"/session/check", httpadapter.GatewayRequired},
		{"/application-icons/:application_id/:id", httpadapter.GatewayRequired},
		// 分類の無い経路は既定で forbidden になる。既定を required にすると、
		// 認証を持たない運用経路が足されたときに検査が公開を要求してしまう。
		{"/api/future/v1/not-yet-classified", httpadapter.GatewayForbidden},
	} {
		if got := httpadapter.ClassifyGatewayExposure(tc.pattern); got != tc.want {
			t.Errorf("ClassifyGatewayExposure(%q) = %q, want %q", tc.pattern, got, tc.want)
		}
	}
}

//spec:covers EX-SYSTEM-021-01: 期待の一覧が組み立て済みの経路を両方の形で覆い、具体パスを持つこと
func TestGatewayRouteExpectationsCoverEveryAssembledRoute(t *testing.T) {
	t.Parallel()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{})
	assembled := map[string]struct{}{}
	for _, route := range e.Router().Routes() {
		if route.Method == echo.RouteNotFound {
			continue
		}
		assembled[route.Path] = struct{}{}
	}

	expectations := httpadapter.GatewayRouteExpectations()
	if len(expectations) == 0 {
		t.Fatal("no expectation was produced; the check would pass vacuously")
	}

	covered := map[string]struct{}{}
	realmForms := 0
	for _, expectation := range expectations {
		if _, ok := assembled[expectation.Pattern]; !ok {
			t.Errorf("expectation names %q, which the assembled router does not register", expectation.Pattern)
		}
		covered[expectation.Pattern] = struct{}{}
		// 具体パスでなければ許可リストの照合器に与えられない。
		if strings.Contains(expectation.SamplePath, ":") || strings.Contains(expectation.SamplePath, "*") {
			t.Errorf("sample path for %q is not concrete: %q", expectation.Pattern, expectation.SamplePath)
		}
		if strings.HasPrefix(expectation.Pattern, "/realms/:tenant_id/") {
			realmForms++
		}
	}
	for pattern := range assembled {
		if _, ok := covered[pattern]; !ok {
			t.Errorf("assembled route %q has no expectation", pattern)
		}
	}
	// ホスト形式だけを覆って終わると、`@realmBackend` の列挙は何とも照合されない。
	if realmForms == 0 {
		t.Fatal("no /realms/{tenant_id} form was enumerated; the realm allowlist would never be checked")
	}
}

//spec:covers EX-SYSTEM-021-02: 必須経路を落とした許可リストを、どの設定が欠いているかとともに拒否すること
func TestCheckGatewayAllowlistsReportsAMissingRequiredRoute(t *testing.T) {
	t.Parallel()

	caddyfile := repositoryFile(t, "frontend/Caddyfile")
	viteConfig := repositoryFile(t, "frontend/vite.config.ts")

	// `/ssf/*` を両方の設定から外す。Shared Signals の受信経路はこれ 1 本しかない。
	brokenCaddyfile := strings.ReplaceAll(caddyfile, " /ssf/* \\\n", "")
	brokenCaddyfile = strings.ReplaceAll(brokenCaddyfile, "ssf|", "")
	brokenViteConfig := strings.ReplaceAll(viteConfig, "      '/ssf': apiTarget,\n", "")
	brokenViteConfig = strings.ReplaceAll(brokenViteConfig, "ssf|", "")
	if brokenCaddyfile == caddyfile || brokenViteConfig == viteConfig {
		t.Fatal("the fixture did not remove anything; the check would pass vacuously")
	}

	findings := httpadapter.CheckGatewayAllowlists(brokenCaddyfile, brokenViteConfig)
	if len(findings) == 0 {
		t.Fatal("removing the Shared Signals route from both allowlists produced no finding")
	}
	report := strings.Join(findings, "\n")
	for _, want := range []string{"/ssf/streams/", "frontend/Caddyfile", "frontend/vite.config.ts"} {
		if !strings.Contains(report, want) {
			t.Errorf("findings do not name %q:\n%s", want, report)
		}
	}
}

//spec:covers EX-SYSTEM-021-03: 通してはならない経路を通している許可リストを拒否すること
func TestCheckGatewayAllowlistsRejectsAnExposedMetricsRoute(t *testing.T) {
	t.Parallel()

	caddyfile := repositoryFile(t, "frontend/Caddyfile")
	viteConfig := repositoryFile(t, "frontend/vite.config.ts")

	exposed := strings.ReplaceAll(caddyfile, " /health\n", " /health \\\n /metrics\n")
	if exposed == caddyfile {
		t.Fatal("the fixture did not add anything; the check would pass vacuously")
	}

	findings := httpadapter.CheckGatewayAllowlists(exposed, viteConfig)
	report := strings.Join(findings, "\n")
	if !strings.Contains(report, "/metrics") {
		t.Fatalf("exposing /metrics through the gateway produced no finding naming it:\n%s", report)
	}
}

//spec:covers EX-SYSTEM-021-04: 許可リストを読み取れないときに成功させず失敗させること
func TestCheckGatewayAllowlistsFailsWhenAnAllowlistCannotBeRead(t *testing.T) {
	t.Parallel()

	caddyfile := repositoryFile(t, "frontend/Caddyfile")
	viteConfig := repositoryFile(t, "frontend/vite.config.ts")

	for _, tc := range []struct {
		name       string
		caddyfile  string
		viteConfig string
		want       string
	}{
		{
			name:       "Caddy のパスマッチャーの名前が変わった",
			caddyfile:  strings.ReplaceAll(caddyfile, "@backend path", "@upstream path"),
			viteConfig: viteConfig,
			want:       "@backend",
		},
		{
			name:       "Caddy の正規表現マッチャーの名前が変わった",
			caddyfile:  strings.ReplaceAll(caddyfile, "@realmBackend path_regexp", "@realm path_regexp"),
			viteConfig: viteConfig,
			want:       "@realmBackend",
		},
		{
			name:       "Vite の設定が proxy ブロックを持たない",
			caddyfile:  caddyfile,
			viteConfig: strings.ReplaceAll(viteConfig, "proxy: {", "upstreams: {"),
			want:       "server.proxy",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			findings := httpadapter.CheckGatewayAllowlists(tc.caddyfile, tc.viteConfig)
			report := strings.Join(findings, "\n")
			if !strings.Contains(report, tc.want) {
				t.Fatalf("an unreadable allowlist produced no finding naming %q:\n%s", tc.want, report)
			}
		})
	}
}

//spec:covers EX-SYSTEM-021-01: Caddy が許す 2 つの path_regexp の書き方のどちらからも realm 形の許可リストを読むこと
func TestCheckGatewayAllowlistsReadsAnUnnamedPathRegexp(t *testing.T) {
	t.Parallel()

	// `path_regexp` はマッチャー名を省略できる。省略した書き方でも読めなければ、名前を
	// 外した瞬間に「読み取れない」側へ倒れ、乖離ではない失敗を報告することになる。
	caddyfile := strings.Replace(
		repositoryFile(t, "frontend/Caddyfile"),
		"@realmBackend path_regexp realmBackend ",
		"@realmBackend path_regexp ",
		1,
	)
	if strings.Contains(caddyfile, "path_regexp realmBackend") {
		t.Fatal("the fixture did not drop the matcher name; the check would pass vacuously")
	}

	findings := httpadapter.CheckGatewayAllowlists(caddyfile, repositoryFile(t, "frontend/vite.config.ts"))
	if len(findings) > 0 {
		t.Fatalf("dropping the optional matcher name changed the outcome:\n%s", strings.Join(findings, "\n"))
	}
}

//spec:covers REQ-SYSTEM-021, EX-SYSTEM-021-01: リポジトリが同梱する 3 つの許可リストが、必須経路を両方の形で通すこと
func TestGatewayAllowlistsCarryEveryRequiredRoute(t *testing.T) {
	t.Parallel()

	findings := httpadapter.CheckGatewayAllowlists(
		repositoryFile(t, "frontend/Caddyfile"),
		repositoryFile(t, "frontend/vite.config.ts"),
	)
	if len(findings) > 0 {
		t.Fatalf("the reference gateway configuration drifted from the assembled route table:\n%s", strings.Join(findings, "\n"))
	}
}
