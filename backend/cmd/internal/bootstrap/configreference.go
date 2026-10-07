package bootstrap

import (
	"fmt"
	"slices"
	"strings"
)

// configReferenceSection is one process-scoped group of the generated
// ConfigurationReference. Load calls exactly the Load*Config functions that
// process runs at startup, so the section lists what that process really
// reads rather than a hand-kept list beside it (REQ-SYSTEM-017).
type configReferenceSection struct {
	Title     string
	Processes string
	Summary   string
	Load      func(*ConfigLoader)
}

var configReferenceSections = []configReferenceSection{
	{
		Title:     "共通",
		Processes: "idmagic、idmagic-worker、idmagic-batch、idmagic-seed",
		Summary:   "すべてのプロセスがアダプターを組み立てるときに読む、永続化、通知、WebAuthn、認可、鍵の保管の設定。",
		Load:      func(l *ConfigLoader) { LoadSharedConfig(l) },
	},
	{
		Title:     "API",
		Processes: "idmagic",
		Summary:   "HTTP の待ち受け、堅牢化、セキュリティヘッダー、エンドポイントごとのレート制限。",
		Load:      func(l *ConfigLoader) { LoadAPIConfig(l) },
	},
	{
		Title:     "ワーカー",
		Processes: "idmagic-worker",
		Summary:   "永続ジョブキューのレーン、ランナーの実行間隔、常駐する掃除ループ。",
		Load:      func(l *ConfigLoader) { LoadWorkerConfig(l) },
	},
	{
		Title:     "シード",
		Processes: "idmagic（起動時のシード）、idmagic-seed",
		Summary:   "明示的なシードの要求。idmagic は SEED_PROFILE が設定されているとき起動時に一度だけ適用し、idmagic-seed は自身のフラグのデフォルト値として使う。",
		Load:      func(l *ConfigLoader) { LoadSeedConfig(l) },
	},
}

// configFieldDescriptions is the one hand-written part of the
// ConfigurationReference: what a key means. Every other column is recorded
// by the loader itself. A key with no entry here, and an entry naming a key
// no process reads, both fail RenderConfigReference and its test, so this
// table cannot drift into a stale parallel list of settings.
var configFieldDescriptions = map[string]string{
	// 共通
	"PERSISTENCE":                  "ストレージのバックエンド。`postgres` には DATABASE_URL が必要である。",
	"OBSERVABILITY":                "`otel` にすると OTLP でトレースとメトリクスを送る。この設定にかかわらず、取得型の /metrics は常に提供する。",
	"AUTHZEN":                      "認可の判定点。`remote` は判定を AuthZEN の PDP へ委ね、AUTHZEN_URL を必要とする。",
	"FEATURES_ENABLE":              "明示的に有効にする実行時の機能 ID をカンマ区切りで並べる。使える ID と成熟度は、製品の機能レジストリから生成する。",
	"FEATURES_DISABLE":             "明示的に無効にする実行時の機能 ID をカンマ区切りで並べる。必須の依存は無効にできない。",
	"AUTHZEN_URL":                  "AuthZEN のポリシー判定点のエンドポイント。AUTHZEN=remote のとき必須である。",
	"WEBAUTHN_RP_ID":               "WebAuthn の Relying Party ID。例：`localhost`。未設定の間は WebAuthn とパスキーを無効のままにする。",
	"WEBAUTHN_RP_ORIGINS":          "WebAuthn のセレモニーを実行してよいブラウザーのオリジン。例：`http://localhost:5173`。WEBAUTHN_RP_ID を設定したとき必須である。",
	"WEBAUTHN_RP_DISPLAY_NAME":     "認証器が表示する Relying Party の表示名。",
	"DATABASE_URL":                 "PostgreSQL の接続文字列。PERSISTENCE=postgres のとき必須である。",
	"DB_MAX_CONNS":                 "PostgreSQL のコネクションプールの上限。",
	"DB_MIN_CONNS":                 "プールが常に確保しておく接続の数。",
	"DB_MAX_CONN_IDLE_TIME":        "プール内の接続を閉じるまでのアイドル時間。",
	"DB_MAX_CONN_LIFETIME":         "プール内の接続の最長寿命。",
	"DB_CONNECT_TIMEOUT":           "PostgreSQL への接続を確立するまでの期限。",
	"DB_QUERY_TIMEOUT":             "一つのクエリの期限。結果を読み終えるまで有効である。",
	"DB_BREAKER_FAILURE_THRESHOLD": "PostgreSQL のサーキットブレーカーを開く失敗率。",
	"DB_BREAKER_COOLDOWN":          "PostgreSQL のサーキットブレーカーが開いてから、再び試行するまでの時間。",
	"DB_BREAKER_MIN_REQUESTS":      "失敗率でブレーカーを開いてよくなるまでに、一つのウィンドウで観測するリクエストの数。",
	"KEY_PROVIDER":                 "署名鍵の保管先。鍵を永続化するときは明示的に選ぶ。`local` は秘密 JWK を平文のままアプリケーションのデータベースへ保存するので、そのデータベースのバックアップにも含まれる。`vault` は Vault に保管し、VAULT_ADDR と VAULT_TOKEN を必要とする。PERSISTENCE=postgres のとき必須である。",
	"VAULT_ADDR":                   "Vault のベースアドレス。KEY_PROVIDER=vault のとき必須である。",
	"VAULT_TOKEN":                  "Vault のトークン。KEY_PROVIDER=vault のとき必須である。",
	"VAULT_TRANSIT_MOUNT":          "署名鍵に使う Vault Transit エンジンのマウントパス。",
	"VAULT_KEY_PREFIX":             "テナントごとの Vault Transit 署名鍵の名前に付ける接頭辞。",
	"DATA_KEY_PROVIDER":            "エンベロープ暗号化した可逆なシークレットのマスター鍵の保管先。`openbao` は OPENBAO_ADDR と OPENBAO_TOKEN を必要とする。未設定ではプロセス内の平文の鍵セットを使う（開発専用）。",
	"OPENBAO_ADDR":                 "OpenBao のベースアドレス。DATA_KEY_PROVIDER=openbao のとき必須である。",
	"OPENBAO_TOKEN":                "OpenBao のトークン。DATA_KEY_PROVIDER=openbao のとき必須である。",
	"OPENBAO_TRANSIT_MOUNT":        "OpenBao Transit エンジンのマウントパス。",
	"OPENBAO_DATA_KEY_PREFIX":      "テナントごとの OpenBao Transit 鍵の名前に付ける接頭辞（`{prefix}/{tenant_id}`）。",
	"EMAIL_SENDER":                 "パスワードリセットと通知のメールを送る経路。`smtp` には SMTP_HOST と SMTP_FROM が必要である。",
	"SMTP_HOST":                    "SMTP サーバーのホスト名。EMAIL_SENDER=smtp のとき必須である。",
	"SMTP_FROM":                    "送信するメールの From アドレス。EMAIL_SENDER=smtp のとき必須である。",
	"SMTP_TLS":                     "SMTP の通信の保護方式。",
	"SMTP_PORT":                    "SMTP のポート。`0` は SMTP_TLS の方式に応じたデフォルトのポートを選ぶ（starttls は 587、implicit は 465、none は 25）。",
	"SMTP_USERNAME":                "SMTP のユーザー名。認証のないリレーでは設定しない。",
	"SMTP_PASSWORD":                "SMTP のパスワード。",
	"SMTP_HELO":                    "SMTP サーバーへ名乗る HELO/EHLO の名前。",
	"SMTP_TIMEOUT_SECONDS":         "一回の SMTP 配送の試行の期限。",
	"DEFAULT_LOCALE":               "通知メールの最後の手段となる言語。対応しない値を設定すると起動に失敗する。",
	"BREACHED_PASSWORD_CHECKER":    "パスワードの変更時に行う漏えいパスワードの検査。`hibp` は Have I Been Pwned の range API を呼ぶ。",

	// API
	"ISSUER":                     "このデプロイメントがトークンと Security Event Token を発行する公開のベース URL。Relying Party と SSF の受信者が解決する値と一致させる。",
	"ADDR":                       "HTTP サーバーの待ち受けアドレス。",
	"OTEL_SERVICE_NAME":          "ログ、メトリクス、トレースに付ける service.name。",
	"LOG_LEVEL":                  "標準出力へ書く最低の重大度。",
	"REQUEST_ID_TRUST_INBOUND":   "受け取った X-Request-ID を生成せずに再利用する。このヘッダーを所有して無害化するプロキシの背後でだけ有効にする。",
	"TENANT_BASE_DOMAIN":         "`{realm}.<domain>` のサブドメイン形式のエンドポイントを有効にする親ホスト名。例：`id.example.com`。未設定ではすべてのテナントをパスで振り分ける。",
	"TRUSTED_FORWARDED_HOPS":     "このプロセスの前段にある、信頼する X-Forwarded-For のホップ数。ログインの抑制とエンドポイントのレート制限で、実際のクライアント IP を解決するのに使う。`0` はこのヘッダーをまったく信頼しない。",
	"DRAIN_GRACE_PERIOD_SECONDS": "SIGTERM を受けてから待ち受けを閉じるまで処理を続ける秒数（idmagic）、または実行中のジョブを終えさせる秒数（idmagic-worker）。",
	"PAGINATION_CURSOR_SECRET":   "キーセット方式のページングのカーソルに署名する HMAC のシークレット。複数のレプリカで動かすデプロイメントでは明示的に設定する。設定しないと各レプリカが起動時に自身の値を生成し、他のレプリカが発行したカーソルを拒否する。",

	"ADMISSION_CONTROL_ENABLED":                         "プロセスが飽和したとき、リクエストの優先度クラスに応じて負荷を落とす。無効にすると、対話的な認証を管理系のトラフィックより優先させる唯一の仕組みがなくなる。誤って設定した閾値を、変更をデプロイせずに運用者が止められるようにするための設定である。",
	"ADMISSION_MAX_CONCURRENT_REQUESTS":                 "対話的な認証まで拒否し始める前に、このプロセスが同時に実行してよいリクエストの数（ロードシェディングのステージ 5）。一つのリクエストが同時に保持する PostgreSQL の接続は一つまでなので、この値はプールの待ち行列の長さも抑える。",
	"ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS":      "管理 API、アカウントポータル、SCIM、Shared Signals の受信、動的クライアント登録の同時実行数の上限（ロードシェディングのステージ 4）。ADMISSION_MAX_CONCURRENT_REQUESTS を超えてはならない。",
	"ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS": "集計、エクスポート、インポート、完全な再同期の経路の同時実行数の上限。最初に負荷を落とす対象である（ロードシェディングのステージ 3）。ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS を超えてはならない。",

	"HTTP_READ_HEADER_TIMEOUT": "リクエストヘッダーを読み終えるまでの期限。",
	"HTTP_READ_TIMEOUT":        "リクエスト全体を読み終えるまでの期限。",
	"HTTP_WRITE_TIMEOUT":       "レスポンスを書き終えるまでの期限。",
	"HTTP_IDLE_TIMEOUT":        "アイドル状態の keep-alive 接続を開いたままにする時間。",
	"HTTP_MAX_BODY_BYTES":      "リクエストボディの上限。これより大きいボディは 413 で拒否する。",

	"CSP_REPORT_ONLY":         "段階的に導入するため、Content Security Policy を Content-Security-Policy-Report-Only として送る。",
	"CSP_REPORT_URI":          "Content Security Policy の違反を集める report-uri。",
	"HSTS_ENABLED":            "Strict-Transport-Security を送る。TLS の終端がこのヘッダーを所有するので、デフォルトでは送らない。",
	"HSTS_MAX_AGE_SECONDS":    "有効にしたときの Strict-Transport-Security ヘッダーの max-age。",
	"HSTS_INCLUDE_SUBDOMAINS": "Strict-Transport-Security に includeSubDomains を付ける。",

	"RATE_LIMIT_TOKEN_MAX_REQUESTS":                        "`/token` の固定ウィンドウの上限。client_id と IP ごとに数える。",
	"RATE_LIMIT_TOKEN_WINDOW_SECONDS":                      "`/token` の上限のウィンドウの長さ。",
	"RATE_LIMIT_AUTHORIZE_MAX_REQUESTS":                    "`/authorize` の固定ウィンドウの上限。IP と client_id ごとに数える。",
	"RATE_LIMIT_AUTHORIZE_WINDOW_SECONDS":                  "`/authorize` の上限のウィンドウの長さ。",
	"RATE_LIMIT_PAR_MAX_REQUESTS":                          "`/par` の固定ウィンドウの上限。IP と client_id ごとに数える。",
	"RATE_LIMIT_PAR_WINDOW_SECONDS":                        "`/par` の上限のウィンドウの長さ。",
	"RATE_LIMIT_DEVICE_AUTHORIZATION_MAX_REQUESTS":         "`/device_authorization` の固定ウィンドウの上限。client_id と IP ごとに数える。",
	"RATE_LIMIT_DEVICE_AUTHORIZATION_WINDOW_SECONDS":       "`/device_authorization` の上限のウィンドウの長さ。",
	"RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_MAX_REQUESTS":   "`/bc-authorize` の固定ウィンドウの上限。client_id と IP ごとに数える。",
	"RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_WINDOW_SECONDS": "`/bc-authorize` の上限のウィンドウの長さ。",
	"RATE_LIMIT_PASSWORD_RESET_MAX_REQUESTS":               "`/api/auth/forgot-password` の固定ウィンドウの上限。送信された識別子と IP ごとに数える。",
	"RATE_LIMIT_PASSWORD_RESET_WINDOW_SECONDS":             "パスワードリセットの上限のウィンドウの長さ。",
	"RATE_LIMIT_LOGIN_MAX_REQUESTS":                        "`/api/auth/login` の固定ウィンドウの上限。IP ごとに数える。アカウントごとのログインの抑制とは別に、それに加えて働く。",
	"RATE_LIMIT_LOGIN_WINDOW_SECONDS":                      "ログインの上限のウィンドウの長さ。",

	// ワーカー
	"WORKER_ID":              "ジョブのリースでこのワーカーを識別する値。デフォルトはホスト名で、取得できなければ生成した ID を使う。",
	"JOB_WORKER_LANES":       "このプロセスが引き受ける実行レーン。デフォルトでは一つのプロセスがすべてのレーンを担う。レーンごとに分けた本番のデプロイメントでは、ちょうど一つを設定する。",
	"JOB_WORKER_CONCURRENCY": "レーンごとに並行して実行するジョブの数。下のレーン別の設定があればそちらを使う。",

	"JOB_WORKER_CONCURRENCY_LATENCY_SENSITIVE": "latency_sensitive レーンの JOB_WORKER_CONCURRENCY を上書きする。",
	"JOB_WORKER_CONCURRENCY_DEFAULT":           "default レーンの JOB_WORKER_CONCURRENCY を上書きする。",
	"JOB_WORKER_CONCURRENCY_BULK":              "bulk レーンの JOB_WORKER_CONCURRENCY を上書きする。",
	"JOB_POLL_INTERVAL":                        "ランナーが引き受けられるジョブを探してレーンを問い合わせる間隔。",
	"JOB_LEASE_DURATION":                       "引き受けたジョブに保持するリースの期間。期限が切れると別のワーカーがジョブを引き取る。",
	"JOB_BACKOFF_BASE":                         "ジョブの試行が失敗した後の最初の再試行までの遅延。",
	"JOB_BACKOFF_CAP":                          "指数的に伸びる再試行の遅延の上限。",
	"EPHEMERAL_SWEEP_INTERVAL":                 "TTL の短い一時ストアから期限切れの行を回収する間隔。",
	"SHARED_SIGNALS_DELIVERY_INTERVAL":         "配送期限の来た送信用の Security Event Token を配送する間隔。",
	"PROVISIONING_RECONCILE_INTERVAL":          "ワーカーが有効なすべてのプロビジョニング接続を、記録している下流の状態と照合し、差分についてプロビジョニングタスクを作る間隔。",

	// シード
	"SEED_PROFILE":                   "起動時に適用する明示的なシードのプロファイル。未設定ではシードしない。",
	"SEED_ENVIRONMENT":               "シードを適用する環境。SEED_PROFILE を設定したとき必須である。",
	"SEED_MANIFEST":                  "ルートとなるシードのマニフェストのパス。デフォルトはプロファイル自身のマニフェストである。",
	"SEED_GENERATOR_SEED":            "performance プロファイルで使う、決定的な生成器のシード値。",
	"SEED_SECRET_ROOT":               "マニフェスト中の相対パスの `file` シークレットロケーターを解決するルートディレクトリ。",
	"SEED_FIRST_PARTY_REDIRECT_URIS": "シードするファーストパーティのクライアントのリダイレクト URI。本番の初期構築では必須である。",
}

// externallyOwnedConfigNote documents the variables idmagic responds to
// without reading them itself, so an operator reading the reference is not
// left thinking they do not exist.
const externallyOwnedConfigNote = `## 外部が所有する変数

次の変数は idmagic ではなくライブラリーやビルドが読む。上の検証済みの設定には含まれず、
不正な値でも起動は失敗しない。

| 変数 | 所有者 | 用途 |
| --- | --- | --- |
| ` + "`OTEL_EXPORTER_OTLP_ENDPOINT`" + ` | OpenTelemetry SDK | OBSERVABILITY=otel のときに使う OTLP/HTTP のコレクターのエンドポイント。 |
| ` + "`VITE_DEFAULT_LOCALE`" + ` | フロントエンドのビルド | React のビルドへ埋め込む、起動時のデフォルトの UI ロケール。 |
| ` + "`VITE_DEMO_LOGIN_ENABLED`" + ` | フロントエンドのビルド | Vite の開発サーバーの外でも、デモ用のログインの近道を表示する。 |
`

// RenderConfigReference renders CONFIGURATION.md from the fields each
// process's Load*Config records, so the reference is generated from the same
// code that parses and validates the values (REQ-SYSTEM-017). Secret fields
// render as `secret` with no default: their values never leave the process.
func RenderConfigReference() (string, error) {
	return renderConfigReference(configReferenceSections, configFieldDescriptions, ProductFeatureRegistry())
}

// renderConfigReference は描画の計算そのものである。節・説明・registry を引数で受けるのは、
// 乖離を報告する経路をテストから踏めるようにするためである。製品の表を書き換えて踏ませると、
// 並行して走る他のテストが同じ表を読むので競合になる。
func renderConfigReference(
	sections []configReferenceSection,
	descriptions map[string]string,
	registry FeatureRegistry,
) (string, error) {
	var out strings.Builder
	out.WriteString("<!-- `mise run generate-config-reference` が生成する。手で編集しない。 -->\n\n")
	out.WriteString("# 設定リファレンス\n\n")
	out.WriteString("この文書は、idmagic が起動時に読むすべての環境変数を、それを読むプロセスごとに示す。\n")
	out.WriteString("値は、どのプロセスも待ち受けを開く前、依存先へ接続する前、シードを適用する前に検証する。\n")
	out.WriteString("必須の値の欠落、形式または範囲の誤り、矛盾する組み合わせは、すべての問題をまとめて報告して起動を中止する。\n\n")
	out.WriteString("型が `secret` の値は、エラーメッセージ、ログ、この文書のどこにも書かない。\n")

	documented := map[string]bool{}
	var missing []string
	for _, section := range sections {
		l := NewConfigLoader(func(string) string { return "" })
		section.Load(l)

		fmt.Fprintf(&out, "\n## %s\n\n", section.Title)
		fmt.Fprintf(&out, "読むプロセス：%s\n\n", section.Processes)
		fmt.Fprintf(&out, "%s\n\n", section.Summary)
		out.WriteString("| 変数 | 型 | デフォルト値 | 必須 | 用途 |\n| --- | --- | --- | --- | --- |\n")

		for _, field := range l.Fields() {
			documented[field.Key] = true
			description, ok := descriptions[field.Key]
			if !ok {
				missing = append(missing, field.Key)
				continue
			}
			fmt.Fprintf(&out, "| `%s` | %s | %s | %s | %s |\n",
				field.Key, fieldTypeColumn(field), fieldDefaultColumn(field), requiredColumn(field), description)
		}
	}

	if len(missing) > 0 {
		slices.Sort(missing)
		return "", fmt.Errorf("configFieldDescriptions has no entry for %s", strings.Join(missing, ", "))
	}
	if orphans := undocumentedDescriptions(descriptions, documented); len(orphans) > 0 {
		return "", fmt.Errorf("configFieldDescriptions describes %s, which no process reads", strings.Join(orphans, ", "))
	}

	out.WriteString(RenderFeatureRegistryReference(registry))
	out.WriteString("\n")
	out.WriteString(externallyOwnedConfigNote)
	return out.String(), nil
}

// RenderFeatureRegistryReference は設定と同じ正本から機能の成熟度と更新影響を描画する。
func RenderFeatureRegistryReference(registry FeatureRegistry) string {
	var out strings.Builder
	out.WriteString("\n## 実行時の機能\n\n")
	if len(registry) == 0 {
		out.WriteString("このビルドには、実行時に選択できる機能はない。常に使える静的な機能は、意図してここに載せない。\n")
		return out.String()
	}
	out.WriteString("| 機能 ID | バージョン | 成熟度 | デフォルトの有効化 | 依存 | 更新の方針 | 仕様 |\n")
	out.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, feature := range registry {
		dependencies := "—"
		if len(feature.Dependencies) > 0 {
			values := make([]string, len(feature.Dependencies))
			for i, dependency := range feature.Dependencies {
				values[i] = "`" + string(dependency) + "`"
			}
			dependencies = strings.Join(values, ", ")
		}
		specification := "—"
		if feature.SpecificationRef != "" {
			specification = "`" + feature.SpecificationRef + "`"
		}
		fmt.Fprintf(&out, "| `%s` | `%s` | %s | %s | %s | %s | %s |\n",
			feature.ID, feature.Version, feature.Maturity, feature.DefaultEnablement, dependencies, feature.UpdatePolicy, specification)
	}
	return out.String()
}

func undocumentedDescriptions(descriptions map[string]string, read map[string]bool) []string {
	var orphans []string
	for key := range descriptions {
		if !read[key] {
			orphans = append(orphans, key)
		}
	}
	slices.Sort(orphans)
	return orphans
}

func fieldTypeColumn(field ConfigField) string {
	column := field.Type
	if len(field.Allowed) > 0 {
		column += ": " + "`" + strings.Join(field.Allowed, "`, `") + "`"
	} else if field.Constraint != "" {
		column += " (" + field.Constraint + ")"
	}
	return column
}

func fieldDefaultColumn(field ConfigField) string {
	if field.Secret || field.Default == "" {
		return "—"
	}
	return "`" + field.Default + "`"
}

func requiredColumn(field ConfigField) string {
	if field.Required {
		return "はい"
	}
	if field.RequiredWhen != "" {
		return "`" + field.RequiredWhen + "` のとき"
	}
	return "いいえ"
}
