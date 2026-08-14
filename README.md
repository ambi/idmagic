# IdMagic

**本番運用に対応する、エンタープライズ向けアイデンティティプロバイダー。**

IdMagic は Go で実装したアイデンティティプロバイダーであり、OAuth 2.0、OpenID Connect、SAML、WS-Federation、テナント分離、アプリケーションポータル、アイデンティティ管理を堅牢に提供する。仕様を先に定める開発手順を採用し、API とモデルの契約は TypeSpec、各 Context の振る舞いと現在の設計は 1 つの正規仕様書に置き、実装は Bounded Context を中心に構成する。

複雑なエンタープライズ認証フローを扱える、信頼性の高い実用的なアイデンティティ基盤を目指している。

## 主な機能

- **包括的なアイデンティティプロトコル**: PKCE、PAR、デバイスフロー、DPoP、動的クライアント登録、トークンローテーションを含む、完全な OAuth 2.0 / OpenID Connect 認可サーバー。
- **エンタープライズフェデレーション**: SAML 2.0 IdP、WS-Federation Passive Profile、WS-Trust STS、Microsoft Entra ドメインフェデレーションのプリセットを標準で提供する。
- **マルチテナントアーキテクチャ**: レルム単位のルート、テナントごとに自動ローテーションする署名鍵、テナント固有のアプリケーションカタログ、カスタマイズ可能なブランドポータルによって、テナントを厳密に分離する。
- **高可用性とスケーラビリティ**: PostgreSQL の共有状態、堅牢な分散ジョブ処理、OpenTelemetry のネイティブ統合によってスケールに対応する。
- **モダンな管理体験**: Vite、Tailwind CSS、Radix UI を使った、高性能な React 製の管理コンソールとアカウントポータル。
- **設定可能な資格情報ポリシー**: デフォルトでは NIST SP 800-63B-4 に沿ったパスワード規則を採用する。長さ、履歴、漏洩パスワード検査を適用し、文字種の構成規則や強制ローテーションは設けない。テナントごとの上書きは厳格化だけを許し、ローテーション要件があるテナントはパスワード有効期限を明示的に有効化できる。

## アーキテクチャとリポジトリ構成

IdMagic は軽量な仕様優先の開発手順を採用する。主な Bounded Context は `tenancy`、`idmanagement`、`authentication`、`oauth2`、`application`、`wsfederation`、`saml` である。共有アダプターは `backend/shared`、ランタイムの構成は `backend/cmd/internal/bootstrap` に置く。

| 区分 | 場所 |
| --- | --- |
| 製品仕様と現在の設計 | `spec/**/*.tsp`, `spec/**/SPECIFICATION.md` |
| 変更の設計と履歴 | `work-items/*.md` |
| アプリケーションロジック | `backend/<context>/domain`, `backend/<context>/usecases` |
| 中核とアダプター | `backend/<context>/{domain,usecases,ports}`, `backend/<context>/{handlers_http,db_postgres,...}` |
| ランタイムとインフラストラクチャ | `backend/cmd/`, `backend/bootstrap`, `infra/`, `frontend/` |

`infra/schema/postgres.sql` は現在状態を宣言するスキーマである。アプリケーションは起動時に移行を実行せず、配備時に `psqldef` でスキーマ変更を適用する。

## 開発の始め方

### ローカルでのクイックスタート

組み込み PostgreSQL、API、`worker` プロセス、UI を含む Docker 不要のローカルスタックを起動する:

```bash
just dev
```

初回実行時に、約 190 MB の組み込み PostgreSQL バイナリをダウンロードしてキャッシュする。開発データは一時的なもので、スタックの停止時に削除される。API と `worker` は別のプロセスとして動きながら PostgreSQL のジョブキューを共有するため、この構成でも永続ジョブが動作する。PostgreSQL のローカルエンドポイントは `127.0.0.1:55432` である。

永続ジョブとバックグラウンドの `worker` プロセスを使わない、最小の API と UI の構成:

```bash
just dev-memory
```

<http://localhost:5173/> を開き、ローカルデモ認証を選択する。使用できるアカウント:

- **`alice`** (パスワード: `demo-password-1234`): テナント管理者のデモユーザー
- **`root`** (パスワード: `demo-password-1234`): テナント管理者とシステム管理者を兼ねるユーザー

*注意: `/login` を直接開かないこと。ログイン画面には進行中の認可トランザクションが必要である。*

CIBA のポーリング方式による承認フローを試すには、`alice` でサインインし、別のターミナルで次を実行する:

```bash
just demo-ciba
```

表示された案内に従ってアカウントの承認ページを開き、リクエストを承認してターミナルへ戻る。

### Docker 開発環境

Compose スタックは PostgreSQL、OpenTelemetry Collector、Prometheus、Go API、UI ゲートウェイを起動する。Caddy が統合アプリケーションを <http://localhost:8080/> で公開する。

```bash
just dev-compose       # バックグラウンドで起動
just logs-compose      # ログを追跡
just down-compose      # 停止して削除
```

### 個別に起動する

別々のターミナルで実行する場合は、共有 PostgreSQL を用意し、API と `worker` プロセスの両方へ `PERSISTENCE=postgres` と `DATABASE_URL` を渡す。`just dev-api` だけを実行した場合は、引き続きメモリモードを使う。

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

ローカルのデフォルトでは、メモリ永続化とコンソールへのメール出力を使う。すべての起動時環境変数について、型、デフォルト、条件付き要件、所有するプロセス、シークレット区分を確認するには、生成済みの [設定リファレンス](CONFIGURATION.md) を参照する。不正な起動設定は、リスナー、依存先への接続、seed の適用を始める前に、1 つの集約エラーとして報告する。

### ジョブの `worker` プロセスと定期バッチ

本番では `JOB_WORKER_LANES` 変数を使い、`idmagic-worker` をレーン（`latency_sensitive`、`default`、`bulk` など）ごとの Deployment に分ける。

`idmagic-batch` は 1 つの運用バッチを実行して終了する。外部スケジューラーは `retention-sweep` を毎時、`signing-key-lifecycle` を毎日実行する。いずれも、水平スケールする永続ジョブの `worker` プロセスとは独立している。

### エンベロープ暗号化とデータ鍵

アプリケーションデータベースに残す必要がある可逆なシークレット（現在は MFA の TOTP seed）は、保存時に[エンベロープ暗号化](spec/SPECIFICATION.md#3-envelope-encryption-for-reversible-secrets)する。テナントごとの `DataEncryptionKey`（DEK）が各シークレットを直接暗号化し、差し替え可能な `DATA_KEY_PROVIDER` が保持するマスターキーで DEK をラップする。

- **開発用フォールバック**: `DATA_KEY_PROVIDER` を設定しない場合は、プロセス内の Tink 平文鍵セットを使うため、開発時に外部サービスは不要である。本番では決して選択してはならない。
- **マスターキーを失うと復旧できない。** OpenBao の Transit 鍵をバックアップせずに失うと、すべてのテナントのラップ済み DEK を恒久的にアンラップできなくなる。これは `DestroyTenantDataKey` が意図的に実現する暗号学的消去と同じ性質が、事故によって発生することを意味する。OpenBao の Transit エンジンのストレージは OpenBao 自身のバックアップ手段で保護する。`tenant_data_encryption_keys` の PostgreSQL バックアップには、マスターキーで暗号化した形しか含まれないため、それだけでは復旧できない。
- **鍵の健全性**: `GET /api/admin/data-keys/health` (`system_admin` 専用) は、鍵素材を返さずに、各テナントの有効な DEK のバージョンとステータス、設定済みプロバイダーの名前と到達性を報告する。
- **ローテーションと再暗号化**: テナントの DEK を内部操作でローテーションすると、すべての参照を新しいバージョンへ移す再開可能な `data_key_reencryption` ジョブ（`backend/jobs`）を投入する。ジョブが保留中の参照なしを報告した後に限り、古いバージョンを破棄できる。再暗号化を再開または再走査するには `idmagic-batch data-key-reencryption-sweep` を実行する。冪等なので再実行や定期実行が可能である。

### WebAuthn の設定

WebAuthn はパスキーをブラウザーのオリジンと RP ID に束縛する。ローカル以外の環境では HTTPS を使い、`WEBAUTHN_RP_ID` をユーザーが訪れる登録可能なドメインに設定する。`WEBAUTHN_RP_ORIGINS` には UI が使用するすべての公開オリジンを含める。

### 上流の OIDC アイデンティティプロバイダー

テナント管理者は **設定 → 外部アイデンティティプロバイダー** で受信 OIDC / SAML 接続を設定する。OIDC 接続には固定の HTTPS 発行者と、最後に正常だった認可、トークン、JWKS の各エンドポイントが必要である。上流プロバイダーへ登録するコールバック URI:

```text
https://<idmagic-origin>/realms/<realm>/api/auth/federation/oidc/callback
```

クライアントシークレットは IdMagic のデータベースに保存しない。値を API プロセスの環境へ置き、接続には `env:` 参照だけを保存する:

```bash
CONTOSO_CLIENT_SECRET=replace-with-the-provider-secret
```

接続入力の例:

```json
{
  "display_name": "Contoso Workforce",
  "protocol": "oidc",
  "issuer": "https://login.contoso.example",
  "client_id": "idmagic-production",
  "secret_reference": "env:CONTOSO_CLIENT_SECRET",
  "authorization_endpoint": "https://login.contoso.example/oauth2/authorize",
  "token_endpoint": "https://login.contoso.example/oauth2/token",
  "jwks_uri": "https://login.contoso.example/oauth2/jwks",
  "claim_mapping": {
    "subject": "sub",
    "username": "preferred_username",
    "email": "email",
    "email_verified": "email_verified",
    "name": "name"
  },
  "linking_policy": "none",
  "jit_provisioning": false
}
```

下書きの接続は有効化する前にテストする。ログインリクエストは保存済みのプロバイダーとエンドポイントの設定だけを使い、ブラウザーから任意の Discovery URL やトークン URL を指定することはできない。

JIT プロビジョニングはデフォルトで無効である。有効にすると、最初に受理したアップストリームアイデンティティから、パスワード資格情報を持たない有効なローカルユーザーを作成する。作成対象は `allowed_email_domains` で絞り込む。検証済みメールアドレスによる自動リンクは別の明示的なポリシーであり、アップストリームの `email_verified` クレームと、検証済みローカルメールアドレスとの一意な一致を必要とする。アップストリームプロバイダーのメール検証とアカウント復旧の保証を信頼できない限り、無効のままにする。外部トークンと SAML アサーションはログイン時に検証し、保持しない。

### テナントエンドポイントの形式

各テナントは正規ロケーションと発行者を 1 つだけ持つ。デフォルトの `path` 形式は `{ISSUER}/realms/{realm}` で提供し、ワイルドカード DNS や証明書を必要としない。`subdomain` 形式は `https://{realm}.{TENANT_BASE_DOMAIN}` で提供し、`TENANT_BASE_DOMAIN` が設定されている場合だけ利用できる。イングレス層はワイルドカード DNS と対応するワイルドカード TLS 証明書を提供しなければならない。IdMagic は証明書の発行や更新を行わない。

テナントを `path` から `subdomain` へ、または逆方向へ変更すると、発行者とプロトコルメタデータの URL が変わる。リライングパーティーの再設定、既存パスキーの再登録、進行中のブラウザーセッションの終了が必要になるため、アイデンティティ移行として計画する。利用できるのは IdMagic が管理するサブドメインだけであり、顧客所有ドメインには対応しない。

### 通知メールテンプレート

通知メールはコード内で組み立てず、テンプレートカタログから生成する。各メッセージは 2 段階で解決する。選択した言語に組み込まれたデフォルト文面を使い、テナント固有のカスタマイズが存在すれば上書きする。すべてのメッセージを、プレーンテキストと HTML の両方を含む `multipart/alternative` として送信する。

テンプレートキー:

| キー | 送信条件 | プレースホルダー（`product_name`、`tenant_display_name`、`user_display_name` に加えて使用可能） |
| --- | --- | --- |
| `password_reset` | ユーザーがパスワードのリセットをリクエストしたとき | `reset_url`, `expires_in_minutes` |
| `email_verification` | メールアドレスの検証が必要なとき | `verification_url`, `expires_in_minutes` |
| `email_change_confirmation` | ユーザーがメールアドレスの変更をリクエストしたとき | `confirmation_url`, `expires_in_minutes`, `new_email` |
| `account_security_alert` | まだ発行しない。文面を準備できるようカタログ項目だけを用意している | `event_description`, `occurred_at` |
| `lifecycle_workflow_notification` | ライフサイクルワークフローが `send_email` 操作を実行したとき | `notification_key` |

プレースホルダーは `{{name}}` と書く。各キーが許可する集合を宣言し、管理 API はテンプレートとともにその集合を返す。集合外のプレースホルダーを参照するカスタマイズは、送信時に暗黙に空文字へ変えず、**保存時に拒否する**。これにより、復旧リンクが欠けたメッセージを送ることはない。HTML 本文へ代入する値はレンダラーがエスケープし、リンクはテンプレートではなくサーバーが組み立てる。

言語は、受信者の `locale` ユーザー属性、テナントのデフォルト言語（設定 → 一般）、`DEFAULT_LOCALE`（デフォルトは `en`）の順に解決し、組込み翻訳が存在する最初の言語を選ぶ。組込み翻訳は `ja` と `en` を提供する。

テナントは件名、プレーンテキスト本文、HTML 本文、送信者の表示名をカスタマイズできる。件名と両方の本文は 1 組として保存するため、一部だけ上書きしたテンプレートは存在しない。送信元メールアドレス、外側の HTML 文書、基本スタイルはサーバーが所有する。カスタマイズを削除して「デフォルトに戻す」と組み込み文面へ戻り、バージョン履歴は持たない。

テンプレートエディターから送るテストメッセージは、操作した管理者本人の検証済みメールアドレスへ必ず送信する。宛先を選択できないため、テナント管理者権限を任意のアドレスへのメールリレーとして悪用できない。

### ローカルでのメールテスト（SMTP）

開発中の SMTP テストには Mailpit が適している:

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

IdMagic の管理 API とセルフサービスのアカウント API は外部契約である。テナントは統一された RFC 9068 JWT API アクセストークンで認証し、自動化、プロビジョニング、IaC をこれらの API に対して構築する。この節は、バージョン管理と非推奨化の運用要約である。

**安定性区分。** 外部インターフェースは TypeSpec 契約で分類する:

- `stable` — バージョン付きの外部契約。下記の互換性保証の対象。
- `beta` — まだ互換性保証の対象でない外部契約。将来のエンドポイント用に予約する。
- `internal` — 外部契約に含まれない。ファーストパーティーのブラウザーセッションからだけ到達でき、API アクセストークンでは到達できない対話型フロー (ログイン、MFA 登録、同意など) と、現在は API アクセストークン経路を持たない管理コンソール画面が該当する。`internal` インターフェースは予告なく変更できる。

API アクセストークンで到達できる (`ManagementApiClient*`、`SelfApiClient*`、SCIM スコープ)、外部標準が規定するプロトコルエンドポイント (OAuth2 / OIDC、SAML、WS-Federation、SCIM、SSF)、または未認証の公開資産・運用エンドポイント (ヘルスプローブ、メトリクス、ブランド資産) のいずれかに該当する場合だけ、インターフェースを `stable` または `beta` とする。

**互換性の定義。** 後方互換な変更は、フィールドの追加、任意パラメーターの追加、新しいエンドポイントの追加である。破壊的変更は、フィールドの削除または名前変更、フィールド型の変更、フィールドの必須化、エラーコードの変更、デフォルト値の変更である。`BackendErrorResponse` で返すエラーコードは契約の一部となる。

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

- **製品仕様と設計**: [spec/SPECIFICATION.md](spec/SPECIFICATION.md)
- **API とモデルの仕様**: [spec/main.tsp](spec/main.tsp)
- **閲覧用仕様サイト**: `just spec-render` で `spec/generated/docs/index.html` に生成する。手法、システム全体、Bounded Context、Swagger UI API、検索可能な TypeSpec モデルを別々のページとして含む
- **インフラストラクチャと Kubernetes**: [infra/README.md](infra/README.md)
- **seed プロファイル**: [seed/README.md](seed/README.md)
- **UI 設計とローカライズ**: [frontend/README.md](frontend/README.md)
- **PostgreSQL 手順**: [infra/schema/README.md](infra/schema/README.md)
