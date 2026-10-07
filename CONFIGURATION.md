<!-- `mise run generate-config-reference` が生成する。手で編集しない。 -->

# 設定リファレンス

この文書は、idmagic が起動時に読むすべての環境変数を、それを読むプロセスごとに示す。
値は、どのプロセスも待ち受けを開く前、依存先へ接続する前、シードを適用する前に検証する。
必須の値の欠落、形式または範囲の誤り、矛盾する組み合わせは、すべての問題をまとめて報告して起動を中止する。

型が `secret` の値は、エラーメッセージ、ログ、この文書のどこにも書かない。

## 共通

読むプロセス：idmagic、idmagic-worker、idmagic-batch、idmagic-seed

すべてのプロセスがアダプターを組み立てるときに読む、永続化、通知、WebAuthn、認可、鍵の保管の設定。

| 変数 | 型 | デフォルト値 | 必須 | 用途 |
| --- | --- | --- | --- | --- |
| `ISSUER` | url | `http://localhost:8080` | いいえ | このデプロイメントがトークンと Security Event Token を発行する公開のベース URL。Relying Party と SSF の受信者が解決する値と一致させる。 |
| `PERSISTENCE` | enum: `memory`, `postgres` | `memory` | いいえ | ストレージのバックエンド。`postgres` には DATABASE_URL が必要である。 |
| `OBSERVABILITY` | enum: `noop`, `otel` | `noop` | いいえ | `otel` にすると OTLP でトレースとメトリクスを送る。この設定にかかわらず、取得型の /metrics は常に提供する。 |
| `FEATURES_ENABLE` | enum list | — | いいえ | 明示的に有効にする実行時の機能 ID をカンマ区切りで並べる。使える ID と成熟度は、製品の機能レジストリから生成する。 |
| `FEATURES_DISABLE` | enum list | — | いいえ | 明示的に無効にする実行時の機能 ID をカンマ区切りで並べる。必須の依存は無効にできない。 |
| `AUTHZEN` | enum: `local`, `remote` | `local` | いいえ | 認可の判定点。`remote` は判定を AuthZEN の PDP へ委ね、AUTHZEN_URL を必要とする。 |
| `AUTHZEN_URL` | url | — | `AUTHZEN=remote` のとき | AuthZEN のポリシー判定点のエンドポイント。AUTHZEN=remote のとき必須である。 |
| `WEBAUTHN_RP_ID` | string | — | いいえ | WebAuthn の Relying Party ID。例：`localhost`。未設定の間は WebAuthn とパスキーを無効のままにする。 |
| `WEBAUTHN_RP_ORIGINS` | list | — | `WEBAUTHN_RP_ID != ""` のとき | WebAuthn のセレモニーを実行してよいブラウザーのオリジン。例：`http://localhost:5173`。WEBAUTHN_RP_ID を設定したとき必須である。 |
| `WEBAUTHN_RP_DISPLAY_NAME` | string | `idmagic` | いいえ | 認証器が表示する Relying Party の表示名。 |
| `DATABASE_URL` | secret | — | `PERSISTENCE=postgres` のとき | PostgreSQL の接続文字列。PERSISTENCE=postgres のとき必須である。 |
| `DB_MAX_CONNS` | integer (>= 0) | `20` | いいえ | PostgreSQL のコネクションプールの上限。 |
| `DB_MIN_CONNS` | integer (>= 0) | `2` | いいえ | プールが常に確保しておく接続の数。 |
| `DB_MAX_CONN_IDLE_TIME` | duration (> 0) | `30s` | いいえ | プール内の接続を閉じるまでのアイドル時間。 |
| `DB_MAX_CONN_LIFETIME` | duration (> 0) | `1h0m0s` | いいえ | プール内の接続の最長寿命。 |
| `DB_CONNECT_TIMEOUT` | duration (> 0) | `5s` | いいえ | PostgreSQL への接続を確立するまでの期限。 |
| `DB_QUERY_TIMEOUT` | duration (> 0) | `5s` | いいえ | 一つのクエリの期限。結果を読み終えるまで有効である。 |
| `DB_BREAKER_FAILURE_THRESHOLD` | number (0..1) | `0.5` | いいえ | PostgreSQL のサーキットブレーカーを開く失敗率。 |
| `DB_BREAKER_COOLDOWN` | duration (> 0) | `30s` | いいえ | PostgreSQL のサーキットブレーカーが開いてから、再び試行するまでの時間。 |
| `DB_BREAKER_MIN_REQUESTS` | integer (>= 0) | `10` | いいえ | 失敗率でブレーカーを開いてよくなるまでに、一つのウィンドウで観測するリクエストの数。 |
| `KEY_PROVIDER` | enum: `local`, `vault` | — | `PERSISTENCE=postgres` のとき | 署名鍵の保管先。鍵を永続化するときは明示的に選ぶ。`local` は秘密 JWK を平文のままアプリケーションのデータベースへ保存するので、そのデータベースのバックアップにも含まれる。`vault` は Vault に保管し、VAULT_ADDR と VAULT_TOKEN を必要とする。PERSISTENCE=postgres のとき必須である。 |
| `VAULT_ADDR` | string | — | `KEY_PROVIDER=vault` のとき | Vault のベースアドレス。KEY_PROVIDER=vault のとき必須である。 |
| `VAULT_TOKEN` | secret | — | `KEY_PROVIDER=vault` のとき | Vault のトークン。KEY_PROVIDER=vault のとき必須である。 |
| `VAULT_TRANSIT_MOUNT` | string | — | いいえ | 署名鍵に使う Vault Transit エンジンのマウントパス。 |
| `VAULT_KEY_PREFIX` | string | — | いいえ | テナントごとの Vault Transit 署名鍵の名前に付ける接頭辞。 |
| `DATA_KEY_PROVIDER` | enum: `openbao` | — | いいえ | エンベロープ暗号化した可逆なシークレットのマスター鍵の保管先。`openbao` は OPENBAO_ADDR と OPENBAO_TOKEN を必要とする。未設定ではプロセス内の平文の鍵セットを使う（開発専用）。 |
| `OPENBAO_ADDR` | string | — | `DATA_KEY_PROVIDER=openbao` のとき | OpenBao のベースアドレス。DATA_KEY_PROVIDER=openbao のとき必須である。 |
| `OPENBAO_TOKEN` | secret | — | `DATA_KEY_PROVIDER=openbao` のとき | OpenBao のトークン。DATA_KEY_PROVIDER=openbao のとき必須である。 |
| `OPENBAO_TRANSIT_MOUNT` | string | — | いいえ | OpenBao Transit エンジンのマウントパス。 |
| `OPENBAO_DATA_KEY_PREFIX` | string | `idmagic/datakeys` | いいえ | テナントごとの OpenBao Transit 鍵の名前に付ける接頭辞（`{prefix}/{tenant_id}`）。 |
| `EMAIL_SENDER` | enum: `console`, `smtp` | `console` | いいえ | パスワードリセットと通知のメールを送る経路。`smtp` には SMTP_HOST と SMTP_FROM が必要である。 |
| `SMTP_HOST` | string | — | `EMAIL_SENDER=smtp` のとき | SMTP サーバーのホスト名。EMAIL_SENDER=smtp のとき必須である。 |
| `SMTP_FROM` | string | — | `EMAIL_SENDER=smtp` のとき | 送信するメールの From アドレス。EMAIL_SENDER=smtp のとき必須である。 |
| `SMTP_TLS` | enum: `starttls`, `implicit`, `none` | `starttls` | いいえ | SMTP の通信の保護方式。 |
| `SMTP_PORT` | integer (>= 0) | `0` | いいえ | SMTP のポート。`0` は SMTP_TLS の方式に応じたデフォルトのポートを選ぶ（starttls は 587、implicit は 465、none は 25）。 |
| `SMTP_USERNAME` | string | — | いいえ | SMTP のユーザー名。認証のないリレーでは設定しない。 |
| `SMTP_PASSWORD` | secret | — | いいえ | SMTP のパスワード。 |
| `SMTP_HELO` | string | — | いいえ | SMTP サーバーへ名乗る HELO/EHLO の名前。 |
| `SMTP_TIMEOUT_SECONDS` | integer (> 0) | `10` | いいえ | 一回の SMTP 配送の試行の期限。 |
| `DEFAULT_LOCALE` | string | — | いいえ | 通知メールの最後の手段となる言語。対応しない値を設定すると起動に失敗する。 |
| `BREACHED_PASSWORD_CHECKER` | enum: `noop`, `hibp` | `noop` | いいえ | パスワードの変更時に行う漏えいパスワードの検査。`hibp` は Have I Been Pwned の range API を呼ぶ。 |

## API

読むプロセス：idmagic

HTTP の待ち受け、堅牢化、セキュリティヘッダー、エンドポイントごとのレート制限。

| 変数 | 型 | デフォルト値 | 必須 | 用途 |
| --- | --- | --- | --- | --- |
| `ADDR` | string | `:8080` | いいえ | HTTP サーバーの待ち受けアドレス。 |
| `OTEL_SERVICE_NAME` | string | `idmagic` | いいえ | ログ、メトリクス、トレースに付ける service.name。 |
| `LOG_LEVEL` | enum: `debug`, `info`, `warn`, `warning`, `error` | `info` | いいえ | 標準出力へ書く最低の重大度。 |
| `REQUEST_ID_TRUST_INBOUND` | boolean: `true`, `false` | `false` | いいえ | 受け取った X-Request-ID を生成せずに再利用する。このヘッダーを所有して無害化するプロキシの背後でだけ有効にする。 |
| `TENANT_BASE_DOMAIN` | string | — | いいえ | `{realm}.<domain>` のサブドメイン形式のエンドポイントを有効にする親ホスト名。例：`id.example.com`。未設定ではすべてのテナントをパスで振り分ける。 |
| `TRUSTED_FORWARDED_HOPS` | integer (>= 0) | `0` | いいえ | このプロセスの前段にある、信頼する X-Forwarded-For のホップ数。ログインの抑制とエンドポイントのレート制限で、実際のクライアント IP を解決するのに使う。`0` はこのヘッダーをまったく信頼しない。 |
| `DRAIN_GRACE_PERIOD_SECONDS` | integer (>= 0) | `5` | いいえ | SIGTERM を受けてから待ち受けを閉じるまで処理を続ける秒数（idmagic）、または実行中のジョブを終えさせる秒数（idmagic-worker）。 |
| `PAGINATION_CURSOR_SECRET` | secret | — | いいえ | キーセット方式のページングのカーソルに署名する HMAC のシークレット。複数のレプリカで動かすデプロイメントでは明示的に設定する。設定しないと各レプリカが起動時に自身の値を生成し、他のレプリカが発行したカーソルを拒否する。 |
| `RATE_LIMIT_TOKEN_MAX_REQUESTS` | integer (> 0) | `60` | いいえ | `/token` の固定ウィンドウの上限。client_id と IP ごとに数える。 |
| `RATE_LIMIT_TOKEN_WINDOW_SECONDS` | integer (> 0) | `60` | いいえ | `/token` の上限のウィンドウの長さ。 |
| `RATE_LIMIT_AUTHORIZE_MAX_REQUESTS` | integer (> 0) | `30` | いいえ | `/authorize` の固定ウィンドウの上限。IP と client_id ごとに数える。 |
| `RATE_LIMIT_AUTHORIZE_WINDOW_SECONDS` | integer (> 0) | `60` | いいえ | `/authorize` の上限のウィンドウの長さ。 |
| `RATE_LIMIT_PAR_MAX_REQUESTS` | integer (> 0) | `30` | いいえ | `/par` の固定ウィンドウの上限。IP と client_id ごとに数える。 |
| `RATE_LIMIT_PAR_WINDOW_SECONDS` | integer (> 0) | `60` | いいえ | `/par` の上限のウィンドウの長さ。 |
| `RATE_LIMIT_DEVICE_AUTHORIZATION_MAX_REQUESTS` | integer (> 0) | `20` | いいえ | `/device_authorization` の固定ウィンドウの上限。client_id と IP ごとに数える。 |
| `RATE_LIMIT_DEVICE_AUTHORIZATION_WINDOW_SECONDS` | integer (> 0) | `60` | いいえ | `/device_authorization` の上限のウィンドウの長さ。 |
| `RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_MAX_REQUESTS` | integer (> 0) | `20` | いいえ | `/bc-authorize` の固定ウィンドウの上限。client_id と IP ごとに数える。 |
| `RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_WINDOW_SECONDS` | integer (> 0) | `60` | いいえ | `/bc-authorize` の上限のウィンドウの長さ。 |
| `RATE_LIMIT_PASSWORD_RESET_MAX_REQUESTS` | integer (> 0) | `5` | いいえ | `/api/auth/forgot-password` の固定ウィンドウの上限。送信された識別子と IP ごとに数える。 |
| `RATE_LIMIT_PASSWORD_RESET_WINDOW_SECONDS` | integer (> 0) | `900` | いいえ | パスワードリセットの上限のウィンドウの長さ。 |
| `RATE_LIMIT_LOGIN_MAX_REQUESTS` | integer (> 0) | `20` | いいえ | `/api/auth/login` の固定ウィンドウの上限。IP ごとに数える。アカウントごとのログインの抑制とは別に、それに加えて働く。 |
| `RATE_LIMIT_LOGIN_WINDOW_SECONDS` | integer (> 0) | `60` | いいえ | ログインの上限のウィンドウの長さ。 |
| `ADMISSION_CONTROL_ENABLED` | boolean: `true`, `false` | `true` | いいえ | プロセスが飽和したとき、リクエストの優先度クラスに応じて負荷を落とす。無効にすると、対話的な認証を管理系のトラフィックより優先させる唯一の仕組みがなくなる。誤って設定した閾値を、変更をデプロイせずに運用者が止められるようにするための設定である。 |
| `ADMISSION_MAX_CONCURRENT_REQUESTS` | integer (> 0) | `256` | いいえ | 対話的な認証まで拒否し始める前に、このプロセスが同時に実行してよいリクエストの数（ロードシェディングのステージ 5）。一つのリクエストが同時に保持する PostgreSQL の接続は一つまでなので、この値はプールの待ち行列の長さも抑える。 |
| `ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS` | integer (> 0) | `192` | いいえ | 管理 API、アカウントポータル、SCIM、Shared Signals の受信、動的クライアント登録の同時実行数の上限（ロードシェディングのステージ 4）。ADMISSION_MAX_CONCURRENT_REQUESTS を超えてはならない。 |
| `ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS` | integer (> 0) | `128` | いいえ | 集計、エクスポート、インポート、完全な再同期の経路の同時実行数の上限。最初に負荷を落とす対象である（ロードシェディングのステージ 3）。ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS を超えてはならない。 |
| `HTTP_READ_HEADER_TIMEOUT` | duration (> 0) | `10s` | いいえ | リクエストヘッダーを読み終えるまでの期限。 |
| `HTTP_READ_TIMEOUT` | duration (> 0) | `30s` | いいえ | リクエスト全体を読み終えるまでの期限。 |
| `HTTP_WRITE_TIMEOUT` | duration (> 0) | `1m0s` | いいえ | レスポンスを書き終えるまでの期限。 |
| `HTTP_IDLE_TIMEOUT` | duration (> 0) | `2m0s` | いいえ | アイドル状態の keep-alive 接続を開いたままにする時間。 |
| `HTTP_MAX_BODY_BYTES` | integer (> 0) | `1048576` | いいえ | リクエストボディの上限。これより大きいボディは 413 で拒否する。 |
| `CSP_REPORT_ONLY` | boolean: `true`, `false` | `false` | いいえ | 段階的に導入するため、Content Security Policy を Content-Security-Policy-Report-Only として送る。 |
| `CSP_REPORT_URI` | string | — | いいえ | Content Security Policy の違反を集める report-uri。 |
| `HSTS_ENABLED` | boolean: `true`, `false` | `false` | いいえ | Strict-Transport-Security を送る。TLS の終端がこのヘッダーを所有するので、デフォルトでは送らない。 |
| `HSTS_MAX_AGE_SECONDS` | integer (>= 0) | `31536000` | いいえ | 有効にしたときの Strict-Transport-Security ヘッダーの max-age。 |
| `HSTS_INCLUDE_SUBDOMAINS` | boolean: `true`, `false` | `true` | いいえ | Strict-Transport-Security に includeSubDomains を付ける。 |

## ワーカー

読むプロセス：idmagic-worker

永続ジョブキューのレーン、ランナーの実行間隔、常駐する掃除ループ。

| 変数 | 型 | デフォルト値 | 必須 | 用途 |
| --- | --- | --- | --- | --- |
| `OTEL_SERVICE_NAME` | string | `idmagic-worker` | いいえ | ログ、メトリクス、トレースに付ける service.name。 |
| `LOG_LEVEL` | enum: `debug`, `info`, `warn`, `warning`, `error` | `info` | いいえ | 標準出力へ書く最低の重大度。 |
| `ADDR` | string | `:8080` | いいえ | HTTP サーバーの待ち受けアドレス。 |
| `WORKER_ID` | string | — | いいえ | ジョブのリースでこのワーカーを識別する値。デフォルトはホスト名で、取得できなければ生成した ID を使う。 |
| `JOB_WORKER_LANES` | enum list: `latency_sensitive`, `default`, `bulk` | `latency_sensitive,default,bulk` | いいえ | このプロセスが引き受ける実行レーン。デフォルトでは一つのプロセスがすべてのレーンを担う。レーンごとに分けた本番のデプロイメントでは、ちょうど一つを設定する。 |
| `JOB_WORKER_CONCURRENCY` | integer (> 0) | `4` | いいえ | レーンごとに並行して実行するジョブの数。下のレーン別の設定があればそちらを使う。 |
| `JOB_WORKER_CONCURRENCY_LATENCY_SENSITIVE` | integer (> 0) | `4` | いいえ | latency_sensitive レーンの JOB_WORKER_CONCURRENCY を上書きする。 |
| `JOB_WORKER_CONCURRENCY_DEFAULT` | integer (> 0) | `4` | いいえ | default レーンの JOB_WORKER_CONCURRENCY を上書きする。 |
| `JOB_WORKER_CONCURRENCY_BULK` | integer (> 0) | `4` | いいえ | bulk レーンの JOB_WORKER_CONCURRENCY を上書きする。 |
| `JOB_POLL_INTERVAL` | duration (> 0) | `2s` | いいえ | ランナーが引き受けられるジョブを探してレーンを問い合わせる間隔。 |
| `JOB_LEASE_DURATION` | duration (> 0) | `5m0s` | いいえ | 引き受けたジョブに保持するリースの期間。期限が切れると別のワーカーがジョブを引き取る。 |
| `JOB_BACKOFF_BASE` | duration (> 0) | `30s` | いいえ | ジョブの試行が失敗した後の最初の再試行までの遅延。 |
| `JOB_BACKOFF_CAP` | duration (> 0) | `30m0s` | いいえ | 指数的に伸びる再試行の遅延の上限。 |
| `EPHEMERAL_SWEEP_INTERVAL` | duration (> 0) | `1m0s` | いいえ | TTL の短い一時ストアから期限切れの行を回収する間隔。 |
| `SHARED_SIGNALS_DELIVERY_INTERVAL` | duration (> 0) | `5s` | いいえ | 配送期限の来た送信用の Security Event Token を配送する間隔。 |
| `PROVISIONING_RECONCILE_INTERVAL` | duration (> 0) | `5m0s` | いいえ | ワーカーが有効なすべてのプロビジョニング接続を、記録している下流の状態と照合し、差分についてプロビジョニングタスクを作る間隔。 |
| `DRAIN_GRACE_PERIOD_SECONDS` | integer (>= 0) | `5` | いいえ | SIGTERM を受けてから待ち受けを閉じるまで処理を続ける秒数（idmagic）、または実行中のジョブを終えさせる秒数（idmagic-worker）。 |

## シード

読むプロセス：idmagic（起動時のシード）、idmagic-seed

明示的なシードの要求。idmagic は SEED_PROFILE が設定されているとき起動時に一度だけ適用し、idmagic-seed は自身のフラグのデフォルト値として使う。

| 変数 | 型 | デフォルト値 | 必須 | 用途 |
| --- | --- | --- | --- | --- |
| `SEED_PROFILE` | enum: `bootstrap`, `development`, `test`, `performance` | — | いいえ | 起動時に適用する明示的なシードのプロファイル。未設定ではシードしない。 |
| `SEED_ENVIRONMENT` | enum: `development`, `test`, `staging`, `production` | — | `SEED_PROFILE != ""` のとき | シードを適用する環境。SEED_PROFILE を設定したとき必須である。 |
| `SEED_MANIFEST` | string | — | いいえ | ルートとなるシードのマニフェストのパス。デフォルトはプロファイル自身のマニフェストである。 |
| `SEED_GENERATOR_SEED` | string | — | いいえ | performance プロファイルで使う、決定的な生成器のシード値。 |
| `SEED_SECRET_ROOT` | string | — | いいえ | マニフェスト中の相対パスの `file` シークレットロケーターを解決するルートディレクトリ。 |
| `SEED_FIRST_PARTY_REDIRECT_URIS` | list | — | `SEED_ENVIRONMENT=production && SEED_PROFILE=bootstrap` のとき | シードするファーストパーティのクライアントのリダイレクト URI。本番の初期構築では必須である。 |

## 実行時の機能

このビルドには、実行時に選択できる機能はない。常に使える静的な機能は、意図してここに載せない。

## 外部が所有する変数

次の変数は idmagic ではなくライブラリーやビルドが読む。上の検証済みの設定には含まれず、
不正な値でも起動は失敗しない。

| 変数 | 所有者 | 用途 |
| --- | --- | --- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry SDK | OBSERVABILITY=otel のときに使う OTLP/HTTP のコレクターのエンドポイント。 |
| `VITE_DEFAULT_LOCALE` | フロントエンドのビルド | React のビルドへ埋め込む、起動時のデフォルトの UI ロケール。 |
| `VITE_DEMO_LOGIN_ENABLED` | フロントエンドのビルド | Vite の開発サーバーの外でも、デモ用のログインの近道を表示する。 |
