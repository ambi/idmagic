# IdMagic

**プロダクションレディなエンタープライズ向け ID プロバイダー。**

IdMagic は Go で実装したIDプロバイダーであり、OAuth 2.0、OpenID Connect、SAML、WS-Federation、テナント分離、アプリケーションポータル、ID管理を堅牢に提供する。

## 主な機能

- **包括的な ID プロトコル**: 完全な OAuth 2.0 / OpenID Connect 認可サーバー。PKCE (RFC 7636)、Pushed Authorization Requests (RFC 9126)、デバイスフロー (RFC 8628)、DPoP (RFC 9449) と mTLS (RFC 8705) による送信者制約、動的クライアント登録 (RFC 7591)、Client ID Metadata Document、リソース指示子 (RFC 8707)、RFC 9068 JWT アクセストークン、トークン交換 (RFC 8693)、Rich Authorization Requests (RFC 9396)、CIBA、RP-Initiated / Front-Channel / Back-Channel Logout に対応し、FAPI 2.0 Security Profile と RFC 9700 (BCP 240) に沿って運用する。
- **エンタープライズフェデレーション**: SAML 2.0 IdP、WS-Federation Passive Profile、WS-Trust STS、Microsoft Entra ドメインフェデレーションのプリセットを標準で提供する。上流の OIDC / SAML IdP からのログインフェデレーションにも対応する。
- **マルチテナントアーキテクチャ**: テナントごとのサブドメイン、自動ローテーションする署名鍵、カスタマイズ可能なブランドポータルによって、テナントを厳密に分離する。
- **エージェントを第一級の主体として扱う**: 自律型・監督型の AI エージェントを `Agent` として登録し、所有者、目的、ライフサイクルを管理する。資格情報は既存の OAuth クライアントに束ね、Kubernetes や SPIFFE の JWT-SVID を長期シークレットなしでトークンへ交換できる。代行はなりすましではなく委譲を既定とし、権限は `authorization_details` で上限付きに宣言する。人間の承認が要る操作は CIBA を通し、キルスイッチは即時に効く。
- **ID ガバナンス**: 入社・異動・退職をライフサイクルワークフローとして自動化する。ユーザーの作成、属性変更、ステータス遷移をトリガーに、グループ、アプリケーション割り当て、必須操作、通知を順に適用し、実行前に結果を試算できる。
- **双方向のデータ連携**: 外部 IdP から SCIM 2.0 サーバーとしてユーザーとグループを受け取り、下流の SaaS へは SCIM 2.0 クライアントとして反映する。管理者向けには CSV の一括インポートとエクスポートを提供し、事前検証、行単位の判定、往復可能なエンコードを保証する。
- **カスタム属性とクレームマッピング**: ユーザーとグループにテナント定義の属性スキーマを持たせ、属性ごとに可視性と本人編集の可否を宣言する。公開するクレームは宣言的な対応付けだけで決まり、OIDC、SAML、WS-Federation が同じフェイルクローズの発行経路を共有する。
- **アプリケーションごとのサインインポリシー**: MFA の要否、再認証を求めるまでの時間、接続元ネットワークをアプリケーション単位で設定できる。テナント全体の既定を置き、アプリケーションごとに上書きでき、実際に適用される内容を管理画面で確認できる。
- **継続的アクセス評価 (SSF / CAEP)**: OpenID Shared Signals Framework の送信側と受信側の両方として動く。失効はまず自身の中で即時に確定させ、そのうえで署名済みの Security Event Token として外部へ伝える。
- **設定可能な資格情報ポリシー**: デフォルトでは NIST SP 800-63B-4 に沿ったパスワード規則を採用する。長さ、履歴、漏洩パスワード検査を適用し、文字種の構成規則や強制ローテーションは設けない。TOTP、パスキー (WebAuthn)、復旧コードによる多要素認証に対応する。
- **高可用性とスケーラビリティ**: 状態は PostgreSQL に集約し、第 2 のデータストアを運用しない。永続ジョブはレーンごとの `worker` が処理し、API とは独立して台数を増やせる。OpenTelemetry のトレースとメトリクスを標準で出力し、Kubernetes には生存・受付可否・起動完了の 3 つのプローブを分けて提供する。
- **モダンなユーザーエクスペリエンス**: Tailwind CSS を使った高性能な React 製のログイン画面・管理コンソール・アカウントポータル。

## 開発の始め方

### ローカルでのクイックスタート

組み込み PostgreSQL、バックエンド API、非同期処理の `worker`、フロントエンドを含む Docker 不要のローカルスタックを起動する:

```bash
just dev
```

初回実行時に、約 190 MB の組み込み PostgreSQL バイナリをダウンロードしてキャッシュする。開発データは一時的なもので、スタックの停止時に削除される。PostgreSQL のローカルエンドポイントは `127.0.0.1:55432` である。

永続ジョブと `worker` プロセスが不要なら、最小の API と UI の構成も起動できる:

```bash
just dev-memory
```

<http://localhost:5173/> を開き、ローカルデモ認証を選択する。使用できるアカウント:

- **`alice`** (パスワード: `demo-password-1234`): テナント管理者のデモユーザー
- **`root`** (パスワード: `demo-password-1234`): テナント管理者とシステム管理者を兼ねるユーザー

CIBA のポーリング方式による承認フローを試すには、`alice` でサインインし、別のターミナルで次を実行する:

```bash
just demo-ciba
```

### Docker 開発環境

Docker Compose スタックは PostgreSQL、OpenTelemetry Collector、Prometheus、Go API、UI ゲートウェイを起動する。Caddy が統合アプリケーションを <http://localhost:8080/> で公開する。

```bash
just dev-compose       # バックグラウンドで起動
just logs-compose      # ログを追跡
just down-compose      # 停止して削除
```

### 個別に起動する

別々のターミナルで実行する場合は、共有 PostgreSQL を用意し、API と `worker` プロセスの両方へ `PERSISTENCE=postgres` と `DATABASE_URL` を渡す。`just dev-api` だけを実行した場合は、引き続きインメモリモードを使う。

```bash
# ターミナル 1: Go API
WEBAUTHN_RP_ID=localhost \
WEBAUTHN_RP_ORIGINS=http://localhost:5173 \
ADDR=:8081 \
ISSUER=http://localhost:5173 \
just dev-api

# ターミナル 2: React UI
just dev-ui
```

### 主なコマンド

このリポジトリはコマンド一覧として `just` を使う。主なコマンド:

```bash
just --list
just setup
just verify
just dev
just verify-go
just verify-ui
just test-ui-e2e
```

### ビルドとバージョン

IdMagic は Go の `-ldflags` を使い、ビルド時にバージョンのメタデータを埋め込める。

```bash
VERSION=1.0.0 just build-go
```

`VERSION` を指定しない場合は `0.0.0-dev` をデフォルトとする。Docker ビルドでは、ビルド引数（`VERSION`、`GIT_COMMIT`、`BUILD_DATE`）でバージョンメタデータを渡せる。

## 設定

ローカルのデフォルトでは、メモリ永続化とコンソールへのメール出力を使う。すべての起動時環境変数について、型、デフォルト、条件付き要件、所有するプロセス、シークレット区分を確認するには、生成済みの [設定リファレンス](CONFIGURATION.md) を参照する。不正な起動設定は、リスナー、依存先への接続、seed の適用を始める前に、エラーとして報告する。

### ジョブの `worker` プロセスと定期バッチ

`idmagic-worker` はレーン（`latency_sensitive`、`default`、`bulk` など）ごとにキューに積まれたジョブを非同期で処理していく。

`idmagic-batch` は 1 つの運用バッチを実行して終了する。外部スケジューラーは `retention-sweep` を毎時、`signing-key-lifecycle` を毎日実行する。

### エンベロープ暗号化とデータ鍵

データベースに残す必要がある可逆なシークレット（現在は MFA の TOTP seed）は、保存時に[エンベロープ暗号化](spec/persistence.md#envelope-encryption-for-reversible-secrets)する。テナントごとの `DataEncryptionKey`（DEK）が各シークレットを直接暗号化し、差し替え可能な `DATA_KEY_PROVIDER` が保持するマスターキーで DEK をラップする。

- **開発用フォールバック**: `DATA_KEY_PROVIDER` を設定しない場合は、プロセス内の Tink 平文鍵セットを使うため、開発時に外部サービスは不要である。本番では決して選択してはならない。
- **マスターキーを失うと復旧できない。** OpenBao の Transit 鍵をバックアップせずに失うと、すべてのテナントのラップ済み DEK を恒久的にアンラップできなくなる。これは `DestroyTenantDataKey` が意図的に実現する暗号学的消去と同じ性質が、事故によって発生することを意味する。OpenBao の Transit エンジンのストレージは OpenBao 自身のバックアップ手段で保護する。`tenant_data_encryption_keys` の PostgreSQL バックアップには、マスターキーで暗号化した形しか含まれないため、それだけでは復旧できない。
- **ローテーションと再暗号化**: テナントの DEK を内部操作でローテーションすると、すべての参照を新しいバージョンへ移す再開可能な `data_key_reencryption` ジョブ（`backend/jobs`）を投入する。ジョブが保留中の参照なしを報告した後に限り、古いバージョンを破棄できる。再暗号化を再開または再走査するには `idmagic-batch data-key-reencryption-sweep` を実行する。冪等なので再実行や定期実行が可能である。

### WebAuthn の設定

WebAuthn はパスキーをブラウザーのオリジンと RP ID に束縛する。ローカル以外の環境では HTTPS を使い、`WEBAUTHN_RP_ID` にはユーザーが訪れる登録可能なドメインに設定する。`WEBAUTHN_RP_ORIGINS` には UI が使用するすべての公開オリジンを含める。

### テナントエンドポイントの形式

各テナントは正規ロケーションと発行者を 1 つだけ持つ。デフォルトの `path` 形式は `{ISSUER}/realms/{realm}` で提供し、ワイルドカード DNS や証明書を必要としない。`subdomain` 形式は `https://{realm}.{TENANT_BASE_DOMAIN}` で提供し、`TENANT_BASE_DOMAIN` が設定されている場合だけ利用できる。イングレス層はワイルドカード DNS と対応するワイルドカード TLS 証明書を提供しなければならない。

テナントを `path` から `subdomain` へ、または逆方向へ変更すると、発行者とプロトコルメタデータの URL が変わる。RP の再設定、既存パスキーの再登録、進行中のブラウザーセッションの終了が必要になるため、ID 移行として計画する。利用できるのは IdMagic が管理するサブドメインだけであり、顧客所有ドメインには対応しない。

### 通知メールテンプレート

通知メールのメッセージは 2 段階で解決する。選択した言語に組み込まれたデフォルト文面を使い、テナント固有のカスタマイズが存在すれば上書きする。すべてのメッセージを、プレーンテキストと HTML の両方を含む `multipart/alternative` として送信する。

テンプレートキー:

| キー | 送信条件 | プレースホルダー（`product_name`、`tenant_display_name`、`user_display_name` に加えて使用可能） |
| --- | --- | --- |
| `password_reset` | ユーザーがパスワードのリセットをリクエストしたとき | `reset_url`, `expires_in_minutes` |
| `email_verification` | メールアドレスの検証が必要なとき | `verification_url`, `expires_in_minutes` |
| `email_change_confirmation` | ユーザーがメールアドレスの変更をリクエストしたとき | `confirmation_url`, `expires_in_minutes`, `new_email` |
| `account_security_alert` | 既知でないデバイスからのサインイン、パスワード・認証要素・メールアドレスの変更、明示的なセッション失効、管理者によるなりすましの開始 | `event_description`, `occurred_at`, `device_summary`, `security_review_url` |
| `lifecycle_workflow_notification` | ライフサイクルワークフローが `send_email` 操作を実行したとき | `notification_key` |

プレースホルダーは `{{name}}` と書く。各キーが許可する集合を宣言し、管理 API はテンプレートとともにその集合を返す。集合外のプレースホルダーを参照するカスタマイズは、送信時に暗黙に空文字へ変えず、**保存時に拒否する**。これにより、復旧リンクが欠けたメッセージを送ることはない。HTML 本文へ代入する値はレンダラーがエスケープし、リンクはテンプレートではなくサーバーが組み立てる。

言語は、受信者の `locale` ユーザー属性、テナントのデフォルト言語（設定 → 一般）、`DEFAULT_LOCALE`（デフォルトは `en`）の順に解決し、組込み翻訳が存在する最初の言語を選ぶ。組込み翻訳は `ja` と `en` を提供する。

テナントは件名、プレーンテキスト本文、HTML 本文、送信者の表示名をカスタマイズできる。件名と両方の本文は 1 組として保存するため、一部だけ上書きしたテンプレートは存在しない。送信元メールアドレス、外側の HTML 文書、基本スタイルはサーバーが所有する。カスタマイズを削除して「デフォルトに戻す」と組み込み文面へ戻り、バージョン履歴は持たない。

### アカウントのセキュリティ通知

`account_security_alert` は、アカウントに起きたセキュリティ上の変化を本人に知らせる。ユーザーは自分のアカウント画面（セキュリティ → 通知）で種別ごとに受信を止められるが、乗っ取りの直後に攻撃者が最初に消すのが通知であるため、資格情報・認証要素・連絡先・なりすましの通知は止められない。

| 種別 | 送信条件 | 受信の停止 |
| --- | --- | --- |
| 新しいデバイスからのサインイン | 過去にサインインしたことのないブラウザーで認証が成功したとき。同じブラウザーの 2 回目以降は送らない | できる |
| 資格情報の変更 | パスワードを変更したとき | できない |
| 認証要素の変更 | 認証アプリ・パスキー・復旧コードの登録と解除、管理者による認証器のリセット、サインイン時に第二要素を省略するデバイスの記憶 | できない |
| 連絡先の変更 | メールアドレスの変更を要求したとき（変更前のアドレスへ届く）と、変更が確定したとき（新しいアドレスへ届く） | できない |
| セッションの失効 | 本人または管理者がサインイン中のセッションを明示的に失効させたとき | できる |
| なりすまし | 管理者がそのユーザーとして操作を開始したとき | できない |

宛先は、そのとき保存されている検証済みのメールアドレスに固定する。検証済みのアドレスを持たないユーザーには送らない。送信は最大限努力であり、配送に失敗しても認証や資格情報の変更そのものは成立する。本文に載るのはブラウザーと OS の系統（例 `Chrome / macOS (JP)`）までで、生の IP アドレスも生の User-Agent も載せない。

### ローカルでのメールテスト（SMTP）

開発中の SMTP テストには [Mailpit](https://mailpit.axllent.org/) が適している:

```bash
mailpit --smtp 127.0.0.1:1025 --listen 127.0.0.1:8025

EMAIL_SENDER=smtp \
SMTP_HOST=127.0.0.1 \
SMTP_PORT=1025 \
SMTP_TLS=none \
SMTP_FROM=noreply@idmagic.test \
./dev.sh
```

## API の安定性、バージョニング、廃止

IdMagic の管理 API とセルフサービスのアカウント API は外部契約である。テナントは統一された RFC 9068 JWT API アクセストークンで認証する。

**安定性区分。** 外部インターフェースは TypeSpec 契約で分類する:

- `stable` — バージョン付きの外部契約。下記の互換性保証の対象。
- `beta` — まだ互換性保証の対象でない外部契約。将来のエンドポイント用に予約する。
- `internal` — 外部契約に含まれない。ファーストパーティーのブラウザーセッションからだけ到達でき、API アクセストークンでは到達できない対話型フロー (ログイン、MFA 登録、同意など) と、現在は API アクセストークン経路を持たない管理コンソール画面が該当する。`internal` インターフェースは予告なく変更できる。

**互換性の定義。** 後方互換な変更は、フィールドの追加、任意パラメーターの追加、新しいエンドポイントの追加である。破壊的変更は、フィールドの削除または名前変更、フィールド型の変更、フィールドの必須化、エラーコードの変更、デフォルト値の変更である。

**バージョン管理。** 管理 API (`/api/admin/v1/...`) とセルフサービス API (`/api/account/v1/...`) はパスでバージョン管理する。`/v1/` だけを提供し、バージョンなしの形は設けない。破壊的変更は既存パスを変更せず、新しい `/v2/` 接頭辞で導入する。同時にサポートするバージョンは最大 2 個とする。

**IdMagic のバージョン方式の対象外**: OAuth2 / OIDC、SAML、WS-Federation、SCIM、SharedSignals（SSF）の各プロトコルエンドポイント。互換性とバージョン管理は各標準が規定し、この方式ではなく Discovery Metadata（`/.well-known/...`）、`/scim/v2/ServiceProviderConfig`、SAML / WS-Fed メタデータを正とする。

**非推奨化。** 非推奨のインターフェースは TypeSpec に予定を記録する。レスポンスには `Deprecation` ヘッダーを付け、廃止時期が決まった後は `backend/shared/http/support_http.DeprecationHeadersMiddleware` を通じて `Sunset` ヘッダーも付ける。

**現在非推奨の API** (生成された TypeSpec OpenAPI を確認し、別の一覧を手作業で保守しない):

```bash
jq '[.paths[][] | select(.deprecated == true) | {operationId, deprecated}]' spec/generated/openapi/idmagic.openapi.json
```

**破壊的変更の検出。** `just check-api-compat` は TypeSpec から生成した OpenAPI を、固定済みのリリース基準 `spec/idmagic.openapi.baseline.json` と比較し、破壊的な差分があれば失敗する。生成物は追跡しない。**リリース後**は、以後の変更を実際に配布した内容と比較できるよう基準を更新する:

```bash
just spec-render
cp spec/generated/openapi/idmagic.openapi.json spec/idmagic.openapi.baseline.json
```

この手順を省くと基準が古くなり、実際の後退を検出できなくなる。一方、実際のリリースなしに基準を更新してコミットしても、実際の後退を検出できなくなる。基準はリリース作業の一部としてだけ更新する。

## 文書案内

各領域の詳細は次の文書を参照する:

- **仕様と設計**: [spec/README.md](spec/README.md)
- **API とモデルの仕様**: [spec/main.tsp](spec/main.tsp)
- **仕様の HTML 化**: `just spec-render` で `spec/generated/docs/index.html` を生成する。手法、システム全体、Bounded Context、Swagger UI API、検索可能な TypeSpec モデルを別々のページとして含む
- **インフラストラクチャと Kubernetes**: [infra/README.md](infra/README.md)
- **seed プロファイル**: [seed/README.md](seed/README.md)
- **UI 設計とローカライズ**: [frontend/README.md](frontend/README.md)
- **PostgreSQL 手順**: [infra/schema/README.md](infra/schema/README.md)
