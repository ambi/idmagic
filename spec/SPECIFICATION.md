---
context: repository
updated_at: 2026-08-15
---

# Whole-System Specification

## Overview

本書はシステム全体の仕様であり、横断的な設計の記録である。システムが現在どう作られていて、なぜその形なのかを述べる。単一の Bounded Context に属する振る舞いと設計は、その Context 自身の `spec/contexts/<context>/SPECIFICATION.md` に置く。リポジトリのパスとインポート文がモジュール構成を表し、軽量なチェックが禁止された外向きの依存だけを拒否する。許可する依存を 1 件ずつ列挙した台帳は持たない。

規範的な振る舞いと現在の設計は、所有する `SPECIFICATION.md` が共に持つ。API とモデルの契約は隣接する TypeSpec にあり、変更ごとの代替案と実装の経緯は work item にある。現在の文書は自己完結しており、過去の意思決定記録は現在の仕様の一部ではない。

移り変わる一覧、すなわちエンドポイント、フィールド、画面はここに置かない。それらはコード、`spec/contexts/*/*.tsp`、UI 文書を正とする。

### Reading order

機能の変更では、次の順に読む。

1. `spec/SPECIFICATION.md`。システム全体の設計と所有権の所在をつかむ。
2. 所有する Context の `SPECIFICATION.md`、`models.tsp`、`main.tsp`。変更は仕様が先である。
3. 進行中の work item。変更ごとの代替案と実装の経緯が載っている。
4. Go の実装。`domain/`、`usecase/`、`ports/`、関係する `<role>_<technology>/` アダプターの順。
5. `backend/shared/` と `backend/cmd/internal/bootstrap/`。横断的な HTTP や永続化の振る舞いに触れるときだけ読む。
6. UI に触れるときは `spec/contexts/system/SPECIFICATION.md` と `frontend/src/features/README.md` を先に読む。

逆に実装から仕様へさかのぼる場合は、パッケージ名が Context 名とほぼ対応する。例外は `backend/shared/` に集約されている。

## Design

### Structure

```text
.
├── backend/           # Go Bounded Contexts, shared, cmd/
├── frontend/          # React UI and gateway
├── spec/              # TypeSpec, canonical specification/design Markdown, and release baseline
│   └── contexts/<context>/SPECIFICATION.md
├── infra/             # container, local runtime, and database schema assets
├── load/k6/           # tenant-local OAuth SLO smoke
├── tools/             # specification, boundary, compatibility, and rendering tools
└── work-items/        # units of work, decision history, and completion records
```

依存は `spec` から実装と派生成果物へ向かって流れる。`backend` のドメイン層とユースケース層のパッケージが、アダプターやランタイムへ逆向きに依存することはない。

#### Development flow mapping

このリポジトリは、仕様、変更の記録、実装、ランタイムの境界を区別して保つが、それらに方法論固有の層の名前を割り当てない。

| 関心事 | 置き場所 | 読み方 |
| --- | --- | --- |
| 仕様と設計 | `spec/**/*.tsp`, `spec/**/SPECIFICATION.md` | 規範的な振る舞い、契約、現在の根拠。変更はここから始まる。 |
| 変更の記録 | `work-items/*.md` | 1 つの変更についての代替案、計画、作業、完了の記録。 |
| Application logic | `backend/<context>/domain`, `backend/<context>/usecase` | フレームワークに依存しないドメインとユースケース。 |
| Adapter | `backend/<context>/{handlers_http,db_postgres,...}`, `backend/shared/<capability>/<role>_<technology>` | HTTP、永続化、暗号、ポリシー、通知などの外向きの接続。 |
| ランタイムと基盤 | `backend/cmd/`, `backend/cmd/internal/bootstrap`, `infra/`, `frontend/`, `docker compose` | 起動、依存注入、配信、プロセス境界。 |

生成される OpenAPI は追跡対象外であり、TypeSpec から作り直される。追跡しているほうの OpenAPI のベースラインは最後にリリースしたワイヤ契約を表し、リリースの一部としてのみ変更される。

### Stack

- Go、React/TypeScript、Bun、PostgreSQL、Docker Compose、Kubernetes、Prometheus、Grafana、Loki、 Promtail、k6。
- User の属性に対する Dynamic Group のメンバーシップ式は、制限された CEL 環境 (`cel-go`) で評価する。安全でない式を受理できないよう環境を絞っており、規則の版が食い違えば fail-closed になる。

### Context Map

この図は選別した DDD の Context Map であり、ドメインに面した関係と統合の境界を示す。ソースコードのインポートをすべて示すものではない。矢印は supplier（上流）から customer（下流）へ向かう。`OHS/PL` は Published Language を伴う Open Host Service、`C/S` は Customer/Supplier、`ACL` は Anti-Corruption Layer、`Events` は公開イベントによる関係を意味する。コードの依存関係についてはリポジトリのパスとインポートを引き続き正とし、この図をアーキテクチャ台帳にはしない。

```mermaid
flowchart LR
  Tenancy[Tenancy]
  IdManagement[IdManagement]
  IdGovernance[IdGovernance]
  Authentication[Authentication]
  OAuth2[OAuth2]
  Application[Application]
  ClaimMapping[ClaimMapping]
  Provisioning[Provisioning]
  Sourcing[Sourcing]
  ApiTokens[ApiTokens]
  Jobs[Jobs]
  Seeding[Seeding]
  SigningKeys[SigningKeys]
  DataKeys[DataKeys]
  WsFederation[WsFederation]
  Saml[Saml]
  WorkloadIdentity[WorkloadIdentity]
  SharedSignals[SharedSignals]
  Audit[Audit]
  System[System]

  Tenancy -->|OHS/PL: tenant boundary| IdManagement
  Tenancy -->|OHS/PL: tenant settings| Application
  IdManagement -->|OHS/PL: principals| Authentication
  IdManagement -->|Events: lifecycle| IdGovernance
  IdGovernance -->|C/S: governed mutations| IdManagement
  IdManagement -->|Events: lifecycle| Provisioning
  Sourcing -->|ACL: authoritative identity| IdManagement
  Authentication -->|OHS/PL: authenticated subject| OAuth2
  Application -->|C/S: protocol binding and gate| OAuth2
  Application -->|C/S: protocol binding and gate| Saml
  Application -->|C/S: protocol binding and gate| WsFederation
  ClaimMapping -->|OHS/PL: released claims| OAuth2
  ClaimMapping -->|OHS/PL: released claims| Saml
  ClaimMapping -->|OHS/PL: released claims| WsFederation
  SigningKeys -->|OHS/PL: signing service| OAuth2
  SigningKeys -->|OHS/PL: XML signing service| Saml
  SigningKeys -->|OHS/PL: XML signing service| WsFederation
  SigningKeys -->|OHS/PL: SET signing service| SharedSignals
  DataKeys -->|OHS/PL: encryption-key lifecycle| Authentication
  WorkloadIdentity -->|ACL: workload attestation| OAuth2
  ApiTokens -->|OHS/PL: API principal| System
  Jobs -->|OHS/PL: durable execution| IdGovernance
  Jobs -->|OHS/PL: durable execution| Provisioning
  Jobs -->|OHS/PL: durable execution| SharedSignals
  Seeding -->|C/S: published commands| Tenancy
  Seeding -->|C/S: published commands| IdManagement
  Seeding -->|C/S: published commands| Application
  IdManagement -->|Events: audit facts| Audit
  Authentication -->|Events: audit facts| Audit
  OAuth2 -->|Events: audit facts| Audit
  System -->|C/S: UI and runtime composition| Authentication
  System -->|C/S: UI and runtime composition| Application
```

次の表が、全 Bounded Context の責務と実装場所の索引である。

| Specification context | Go package | Responsibility |
| --- | --- | --- |
| `System` | `backend/cmd/internal/bootstrap`, `backend/shared/http/server_http`, `frontend/` | 横断的な利用体験、起動、経路の組み立て、健全性。 |
| `Tenancy` | `backend/tenancy` | Tenant と realm、テナント単位の設定、ユーザーの属性スキーマ、制御面のテナント管理。 |
| `IdManagement` | `backend/idmanagement` | User、Group、Agent、自身のプロフィール、アイデンティティのライフサイクル、CEL による動的メンバーシップ規則と再評価。 |
| `IdGovernance` | `backend/idgovernance` | LifecycleWorkflow のポリシーとオーケストレーション。記録の正は IdManagement に残る。 |
| `Authentication` | `backend/authentication` | 資格情報の検証、MFA、ログインセッション、ステップアップ認証、パスワードの変更とリセット、認証イベント。 |
| `OAuth2` | `backend/oauth2` | OAuth 2.0 / OIDC のプロトコルのエンドポイント、クライアント、同意、トークン、ロールのポリシー。 |
| `Application` | `backend/application` | Application のカタログ、プロトコルの束縛、割り当て、ポータルの並び順と分類。 |
| `Audit` | `backend/audit` | 全 Context にまたがる監査イベントの Read Model。検索属性の登録簿、個人識別情報の変換、管理 API、保持期間を所有する。 |
| `ClaimMapping` | `backend/claimmapping` | プロトコルに依存しないクレーム開示ポリシー、アイデンティティ属性からクレームへのマッピング、フェイルクローズな検証。 |
| `Provisioning` | `backend/provisioning` | SCIM 2.0 による外向きのプロビジョニング。下流の SaaS へ反映するライフサイクルを担う。idmagic の User と Group が正となる情報源であり、下流はその写しである。 |
| `Sourcing` | `backend/sourcing` | 上流の権威からの内向きのアイデンティティ取り込み。取り込み元のバインディング、外部の不変 ID との相関、上流の権威に追随する削除と無効化を所有する。取り込み元ごとに 1 つの機能単位として構成し、現在は `sourcing/scim` だけを持つ。 |
| `ApiTokens` | `backend/apitoken` | 管理 API と SCIM API を認証するテナント単位の API アクセストークン（`idmagic_pat_` で始まる）。発行、失効、一覧、スコープの語彙を担う。 |
| `Jobs` | `backend/jobs` | テナント境界を保つ汎用の非同期ジョブ基盤。設計は [Jobs specification](contexts/jobs/SPECIFICATION.md) を参照。 |
| `Seeding` | `backend/seeding` | 環境ごとの構成、プレビュー、機密情報を伏せた計画、適用ポリシー。業務データとその永続化は、記録を所有する各 Context に残る。 |
| `SigningKeys` | `backend/signingkeys` | テナントと用途で区切られた鍵のメタデータ、X.509 資格情報、ローテーション、Repository のポート、管理 API と JWKS の HTTP エンドポイント、メモリ・PostgreSQL・Vault の各アダプター。JWT と XML の署名処理はプロトコル側のアダプターに残す。設計は [Signing Keys specification](contexts/signing-keys/SPECIFICATION.md) を参照。 |
| `DataKeys` | `backend/datakeys` | アプリケーションのデータベースに残さざるを得ない可逆なシークレット（MFA の TOTP seed など）のための、テナントごとの `DataEncryptionKey`（DEK）のメタデータとライフサイクル（初期化、ローテーション、無効化、破棄）。署名鍵 (`SigningKeys`) は所有せず、`EnvelopeCrypto` のポート自体も所有しない。後者は技術的な共通アダプターとして `backend/shared/security` に置く。 |
| `WsFederation` | `backend/wsfederation` | WS-Federation のパッシブプロファイル、WS-Trust のアクティブ STS、フェデレーションメタデータ、MEX、RP の信頼、リクエスト元テナントによる XML 署名。設計は [WS-Federation specification](contexts/ws-federation/SPECIFICATION.md) を参照。 |
| `Saml` | `backend/saml` | SAML 2.0 IdP、SP の信頼、メタデータ、SSO と SLO、リクエスト元テナントによる XML 署名。設計は [SAML specification](contexts/saml/SPECIFICATION.md) を参照。 |
| `WorkloadIdentity` | `backend/workloadidentity` | エージェントの実行環境に対するワークロードアイデンティティフェデレーション。登録済みの外部アテステーション発行者 (`WorkloadTrustBundle`) と、subject のパターンから `Agent` へのマッピング (`AgentWorkloadBinding`) を持つ。OAuth2 のトークン交換がこれを使い、長期的なシークレットなしに外部の JWT-SVID を idmagic のトークンへ交換する。 |
| `SharedSignals` | `backend/sharedsignals` | OpenID Shared Signals Framework（SSF）と RFC 8417 の Security Event Token（SET）による継続的アクセス評価（CAEP）およびエージェントのほぼ即時の失効。`Agent` ごとの `AgentRevocationEpoch` を所有し、OAuth2 の `Introspect` がトークンの `issued_at` と突き合わせてフェイルクローズなローカル失効を行う。外部との CAEP イベント交換に使う `SsfStream`、`SsfTransmitterConfig`、`SsfReceiverConfig` も所有する。ローカル失効は常に外部への伝播に先行し、伝播を待たない。`AgentRevocationReactor` は IdManagement がすでに発行するライフサイクルイベント（停止、無効化、資格情報の解除、所有者の無効化・論理削除・削除）に `idmanagement/deps_http.EventReactor` 経由で反応するため、IdManagement のユースケースは SharedSignals への依存や明示的な呼び出しを持たない。外向きの SET 送信フローは実装済みである。`AgentRevocationReactor` のベストエフォートな射影 (`ProjectAgentAccessRevoked`) がローカル失効を `session-revoked` を購読する有効な送信用 `SsfStream` へ配り、`ports.SecurityEventTokenSigner`（`sign_jose` が実装し、独自の鍵素材ではなく SigningKeys のローテーションと JWKS を再利用する）で SET に署名して `SecurityEventDelivery` を投入する。定期実行する `worker` の `ProcessDueDeliveries` が指数バックオフで再試行し、`max_delivery_attempts` を使い切ると配信不能として退避する。SSRF 対策を施した HTTP 送信は `sharedsignals/push_http` が担う。ドメインモデル、メモリと PostgreSQL の永続化、`Introspect` への接続、`Agent` の失効強制、SET の送信は実装済みである。SSF の受信（内向きの `ReceiveSecurityEvent`）と管理 UI（ストリームの作成・更新・削除、配信状況）は未実装である。 |

Context の所有権はここ、および各 Context の要件に記述する。直接のインポートを追加する前に境界チェックを実行し、Context 間の依存は公開インターフェースに限定する。

### Conventions

Bounded Context は通常この形を取る。

```text
backend/<context>/
  domain/            # エンティティ、値オブジェクト、状態遷移、純粋な検証
  usecase/           # 仕様で定めた操作を行うアプリケーションロジック
  ports/             # Repository、ストア、外部サービスの抽象
  handlers_http/     # 受信 HTTP アダプター
  db_memory/         # メモリ版 Repository アダプター
  db_postgres/       # PostgreSQL 版 Repository アダプター
```

`domain/` は Echo、PostgreSQL、HTTP のリクエストとレスポンスを一切知らない。`usecase/` は `ports/` に依存し、具体的なアダプターには決して依存しない。`handlers_http` はワイヤ形式の変換、HTTP ステータスコード、Cookie とヘッダー、CSRF や Origin といった境界の関心事を所有する。`usecase/` がアダプターをインポートしない規則はすべての Context で成り立つ。署名、割り当て可否の判定、認証の解決などの外向きの能力は、`ports/` の抽象またはユースケースパッケージで宣言したインターフェースとして渡し、アダプターが具体的な実装を注入する。例として `oauth2` の `ports.TokenIssuer`、`saml` と `wsfederation` の `ApplicationGate` がある。

アダプターはそれを所有する Context または機能の直下に置き、snake_case の `<role>_<technology>` で命名する。間に `adapters/` や `persistence/` といった分類用ディレクトリは挟まない。パッケージ名だけで、役割がハンドラー、Repository、発行者、クライアントのいずれか、また技術が HTTP、PostgreSQL、S3、SCIM のいずれかを判別できるようにする。分類用ディレクトリを挟むとパッケージ名から働きが分からなくなる。

Context が `domain/` と `usecase/` を持つかどうかは、その Context 自身のロジックの有無で決まる。パッケージを機械的には配置しない。共有する実行時契約の補助は `backend/shared/spec` に置き、Context 固有の業務上の型はその Context の `domain/` が所有する。バインディング以外にドメインロジックを持たない `tenancy` のような Context には、Context 固有の `domain/` を置かない。自身のロジックを持つ Context、たとえば `idmanagement`（User / Group / Agent の Aggregate、属性スキーマ、項目の検証）や `saml` / `wsfederation`（プロトコル固有の解析とクレームのマッピング）は `domain/` を持つ。SSO やサインインを組み立てる Context（SP と RP の解決、署名検証、割り当て可否の判定、クレームの発行）は `usecase/` を持つ。ブラウザー経由のフェデレーションにおける発行判断はすべて `usecase/` に置き、`handlers_http` はワイヤ形式と HTTP の境界に閉じ込める。

`backend/shared/` は、複数の Context が実際に共有する技術的な能力のための場所である。都合だけで Context 固有の概念を置くと、次の変更で読むべき範囲が広がる。具体的なドメインイベントの構造体は、それを所有する Context の `domain/events.go` に置く。`backend/shared/spec/events.go` はイベントのエンベロープとなるインターフェースと、そのワイヤ表現への変換だけを持つ。Audit は具体的な型の登録簿ではなく、安定したイベント種別の識別子で分類する。

#### Feature vertical slices

2 つ以上の独立した部分領域（機能）を持つ Context は、4 層の構成に機能ごとの垂直分割を追加してよい: `backend/<context>/<feature>/{domain,ports,usecase,<role>_<technology>}/`。単一の機能しか持たない Context には置かない。`<context>/<context>/` という重複はディレクトリ構造から目的を読み取りにくくするためである。`idmanagement` はこの構成を採用し、`user`、`group`、`agent` に分割する。

```text
backend/idmanagement/
  module.go                 # Context ごとに 1 つ置く DI の組み立て
  domain/                   # 機能間で共有する型だけ（列挙、DomainEvent）
  usecase/                  # 機能をまたぐユースケース補助とエラー値だけ
  deps_http/                # Deps 型を定義する末端パッケージ
  handlers_http/            # ルート登録と機能をまたぐ統合テスト
  user/  group/  agent/     # 各機能の domain、ports、usecase、アダプター
```

`handlers_http` と `db_postgres` は、Go の言語規則とコード生成の単位に合わせ、`domain`、`ports`、`usecase` とは異なる境界で分割する。

- **handlers_http**: `Deps` の型定義は独立した末端パッケージ `deps_http` に置き、各機能のハンドラーは `func handleX(d Deps, c *echo.Context) error` 形式の自由関数とする。Go ではメソッドをレシーバー型と同じパッケージでしか定義できないため、自由関数にすることで共有する `Deps` を分割せず、実装を機能ごとに分離できる。`routes.go` は `type Deps = httpdeps.Deps` として型エイリアスを公開し、起動処理とテストは `idmagic.Deps{...}` を組み立てる。
- **db_postgres**: `sqlc.yaml` は IdManagement の機能ごとに設定を分け、`queries/*.sql` と生成された `sqlcgen/` を各機能のディレクトリに置く。Go の `_test.go` はパッケージをまたげないため、機能をまたぐテスト準備用の補助（`seedTenant`、`seedUser`）は各機能のパッケージが所有する。`lifecycle_workflows` のクエリと `sqlcgen` は IdGovernance Context が所有し、`backend/idgovernance/db_postgres/` に置く。

中核のパッケージ名は層の名前 (`domain` / `ports` / `usecase`) のままにし、アダプターパッケージは `<role>_<technology>` (`handlers_http`、`db_memory`、`db_postgres`) のままにする。1 つの Context の複数機能をまとめてインポートする箇所では、名前付きインポート (`userDomain`、`groupDomain`) で区別する。衝突をインポートの別名で解決すれば中核パッケージの共通語彙が保たれるが、ディレクトリ名を長くする方法では保たれない。

#### Frontend Component Structure

仕様上の機能とそろえた UI の境界は `frontend/src/features/<feature>/` に置く。その機能のビュー、ローカルコンポーネント、ヘルパー、テスト、ローカライズ辞書 (`*.i18n.ts`) は必ずそのディレクトリに置く。`slices/` という別名は使わない。特定の機能境界にひも付かない、横断的で再利用可能なコンポーネントは `frontend/src/components/` に置く。

### Cross-cutting Concerns

#### HTTP routing

経路は `backend/shared/http/server_http/routes.go` で組み立てる。ここがテナント単位のルートをデフォルトのテナントと `/realms/:tenant_id` の両方に登録し、制御面のテナント管理だけを `/realms/default/admin/tenants` に分離する。

各 Context のルートは `backend/<context>/handlers_http/routes.go` にある。正確なエンドポイントの一覧はそのファイルを参照する。新しい HTTP API は、それを所有する Context の `routes.go` に、同じ `handlers_http` 配下のハンドラーとともに登録する。Context 固有の Repository とルーティングの接続は `backend/<context>/module.go` に集約し、中央のルーターは Module を呼ぶだけにする。

#### Request correlation

すべてのリクエストに `request_id` を割り当て、`X-Request-ID` レスポンスヘッダーで返し、そのリクエストのすべてのアプリケーションログ行に付与する (`OBSERVABILITY=otel` のときは `trace_id` と `span_id` も併せて付く)。

`X-Request-ID` は攻撃者が制御できるため、デフォルトでは **id を自前で生成し、受信した値を無視する** —安全側のデフォルトであり、直接到達できるクライアントが相関 id を偽ったり衝突させたりできない。 `REQUEST_ID_TRUST_INBOUND=true` は、信頼できる境界のプロキシがヘッダーを生成する (つまり無害化する) 場合にのみ設定する。それがあって初めて、プロキシとアプリケーションの層で単一の id を共有する価値が生まれる。クライアントの値をそのまま素通しするプロキシを信頼してはならない。いずれの場合も、再利用する受信値は長さと文字種を制限し、ヘッダーとログへの注入に対する多層防御とする。

#### Cursor pagination

管理用の一覧 API は、署名済みで版の付いたキーセット方式のカーソルを RFC 8288 の `Link` レスポンスヘッダーで運ぶ。カーソルは自身のテナント、問い合わせと並び順の同一性、方向、行の境界を束縛する。新しいカーソルは期限切れにならない。これは認可の能力ではなく、指し示せる一覧上の位置を表すからである。ハンドラーは利用可能な `prev` と `next` の方向だけを出力し、リクエストは依然として通常の認証、認可、 テナントの分離をすべて通過する。

#### HTTP error responses

汎用 API のエラーレスポンスには、デフォルトの形式として RFC 9457 Problem Details（`application/problem+json`、`type`、`title`、`status`、`detail`、`instance`）を使う。`instance` には上記のリクエスト相関用の `request_id` を載せる。HTTP ステータスコードは RFC 9110 に従い、400 はリクエストを解析できないこと（不正な JSON、必須構造の欠落）を、422 は解析できた内容が業務規則に違反すること（不正なロール、参照の不一致、ポリシー違反）を表す。

OAuth2（`backend/oauth2/handlers_http`）、SCIM（`backend/sourcing/scim/handlers_http`）、Dynamic Client Registration（RFC 7591、`backend/oauth2/handlers_http` の一部）は、各標準が定めるエラーレスポンスを返す。標準に従うクライアントとの相互運用性を保つため、これらには Problem Details を適用しない。SharedSignals の管理 API には汎用 API の規則を適用する。

#### Metrics

`GET /metrics` が Prometheus / OpenMetrics 形式のメトリクスを公開する。すべてのルートのひな形についての HTTP の RED (件数、`status_code` によるエラー率、所要時間、処理中の数) に加え、SLO と警報のための認証の golden signal を含む。

| Metric | Labels | Verifies |
| --- | --- | --- |
| `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight` | `route`, `method`, `status_code` | 接点ごとの遅延とエラー率の目標 |
| `authn_login_attempts_total` | `outcome`, `reason_class`, `method` | ログインの成否という golden signal |
| `authn_login_throttle_total` | `policy`, `outcome` | ログインのスロットルの発動率 |
| `endpoint_rate_limit_total` | `policy`, `outcome` | エンドポイントの流量制限の発動率 |
| `oauth2_token_issuance_total`, `oauth2_token_issuance_duration_seconds` | `grant_type`, `outcome` | grant 別の `/token` の発行率と遅延 |
| `http_request_aborts_total`, `operation_detached_completion_failures_total` | `kind` | 中断の扱い |

ラベルはいずれも有限の集合に限る。`tenant_id`、`user_id`、`client_id`、解決済みのリクエストのパスは決してラベルにしない。取りうる値が限られないからである。同じ理由で、このエンドポイントはテナントを解決するミドルウェアの外で収集され、アプリケーションの API とは分離されている。エンドポイントは常に登録されるが、プロセスが起動時に Prometheus の登録簿を組み立て終えるまでは `503` を返す。また `OBSERVABILITY` とは独立に機能する。収集側から取りに来る方式では収集器の設定が要らないからである。公開は折り返しアドレスや管理用のネットワーク上、あるいは認証付きのプロキシの背後に限ること。

#### Logging

アプリケーションのログは標準出力への構造化された JSON Lines である (`timestamp`、`level`、`service`、 `message`、および相関のための `trace_id` / `span_id` / `request_id` — `backend/shared/logging`)。このプロセスはログをそれ以外のどこにも書かない。レプリカやノードをまたいで集約し検索することは、外部から観測する別の関心事であり、OpenTelemetry Collector とは独立に保つ。そうすればログ基盤の障害が trace とメトリクスの送出に影響しない。

**ローカル** (`infra/docker/docker-compose.dev.yaml`): Promtail が Docker Engine API (`docker_sd_configs`) ですべてのコンテナを検出し、そのログを Loki へ送る。したがってホストのログディレクトリを結び付ける必要はなく、Docker のソケットだけでよい。Grafana は初回起動時に Prometheus と Loki の両方をデータソースとして、また既存の golden signal のダッシュボードとともに設定される。 `docker compose up` だけでメトリクスとログを一緒に閲覧できる。

**Kubernetes** (`infra/k8s/monitoring/loki/`): Promtail は DaemonSet として動き、`kubernetes_sd_configs` で pod を検出して `/var/log/pods` を追尾する。Loki は永続ボリュームを持つ単一レプリカの StatefulSet として動く (ファイルシステムへの保存は開発向けのデフォルトであり、実運用の cluster ではオブジェクトストレージを使う保持設定の重ね合わせに置き換える)。Grafana 自体はこのリポジトリでは配備しない。Loki のデータソースは既にあるどの Grafana に対しても、`grafana-dashboard.yaml` が既に依拠しているのと同じ ConfigMap の付き添いコンテナの慣行 (cluster の Grafana の付き添いが監視するラベル) で登録する。 `grafana-dashboard.yaml` のダッシュボードの内容そのものを使うわけではない。

| Field | Loki treatment | Why |
| --- | --- | --- |
| `service`, `level` | index label | 有限の集合である |
| `trace_id`, `span_id`, `request_id` | structured metadata (not a label) | 値が限られない — ここでインデックスのラベルにすると組み合わせが爆発する。[Metrics](#metrics) が `tenant_id` と `user_id` について述べたのと同じ理由 |

#### HTTP server hardening

境界の HTTP サーバーは、本番で安全なタイムアウトとリクエスト本体の上限を適用する。単一の遅い、あるいは過大なクライアントが接続やメモリを枯渇させられないようにするためである (`gosec G112` / CWE-400)。上限を超えた本体は `413` で拒否する。

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_READ_HEADER_TIMEOUT` | `10s` | リクエストのヘッダーを読む上限時間 (slowloris の抑止) |
| `HTTP_READ_TIMEOUT` | `30s` | リクエスト全体を読む上限時間 |
| `HTTP_WRITE_TIMEOUT` | `60s` | レスポンスを書く上限時間 |
| `HTTP_IDLE_TIMEOUT` | `120s` | 持続接続の待機時間の上限 |
| `HTTP_MAX_BODY_BYTES` | `1048576` | リクエスト本体の最大バイト数 (1 MiB) |

これは多層防御であって、境界のプロキシの代わりではない。大量の氾濫と TLS handshake の slowloris に対する第一線は前段の逆プロキシであり、全体の通信量を見て安価に濫用を止められる。idmagic がそれでも自身のタイムアウトと本体の上限を課すのは、プロキシなしで動かしても安全であるため、そしてプロキシからアプリケーションへの区間と cluster 内の直接アクセスも覆うためである。

#### Security response headers

境界のミドルウェアが、すべての backend のレスポンスに `X-Content-Type-Options: nosniff`、 `Referrer-Policy: no-referrer`、`X-Frame-Options: DENY`、そして厳格な `Content-Security-Policy` (`default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'`) を適用する。 `frame-ancestors 'none'` と `X-Frame-Options: DENY` の組み合わせが枠内への埋め込みを禁じるので、ログイン、同意、ポータルの画面は clickjacking されない。CSP は `'unsafe-inline'` を使わない。idmagic が描画する唯一の埋め込みスクリプトは SAML ACS と WS-Fed の POST 束縛の書式の固定された自動送信であり、これはそのレスポンス上の `script-src 'sha256-…'` の要約値で固定し、`form-action` を宛先のエンドポイントに絞っている。

**ヘッダーの所有。** CSP と `frame-ancestors` はルートごとの判断を要するため idmagic が所有する。これにより最小限のプロキシの背後でも、プロキシがなくても成り立つ。単一ページアプリケーションはゲートウェイが配信し、ゲートウェイが静的な HTML 用に自身の `script-src 'self'` の CSP を設定する。

**HSTS は TLS を終端する側のもの。** `Strict-Transport-Security` はデフォルトで無効である。平文の `http` での開発が汚染されないようにするためである。TLS がこの区間か、その手前で終端される場合にのみ有効化する。通常の構成では境界のプロキシに任せ (`HSTS_ENABLED=false`)、アプリケーション自身が表明すべき場合に `HSTS_ENABLED=true` とする (`HSTS_MAX_AGE_SECONDS` と `HSTS_INCLUDE_SUBDOMAINS` で調整する)。

画面を壊さずに CSP を厳しくするには、`CSP_REPORT_ONLY=true` で `Content-Security-Policy-Report-Only` を出し、`CSP_REPORT_URI=<url>` で違反を収集し、観察してから強制へ戻す。

#### Persistence

永続化ポートと Repository の実装は、それを所有する Context に属する。Context 固有のメモリと PostgreSQL のアダプターは `backend/<context>/{db_memory,db_postgres}` に置き、共有のデータベース接続プール、行の読み取り、トランザクションのヘルパーは `backend/shared/storage/db_postgres` に置く。一時的な状態も PostgreSQL に統合するため、2 種類目のデータストアは運用しない。

`db_postgres` の静的な SQL 文はすべて `sqlc` で型安全な問い合わせを生成しなければならない。文字列を伴う生の `Pool.Query` と `Pool.Exec` は、`sqlc` に利点がない極めて動的な問い合わせに限って許され、デフォルトではなく脱出口として使う。

PostgreSQL に構造を追加するには、まず `infra/schema/postgres.sql` の現行スキーマを更新する。構造差分は `psqldef` のプレビューで確認し、配備前の処理で適用する（手順は `infra/schema/README.md` を参照）。CI はさらに、空のデータベースに対して `postgres.sql` が `psqldef` で収束することを強制する (`just check-schema`)。適用後のプレビューが空になること、再適用後のプレビューも空のままであることを確認する。これは一度限りの確認ではなく、`psqldef` のリグレッションを継続的に検出するためのものである。対象とする不具合の類型は `infra/schema/README.md` の Rules を参照する。構造差分では表現できない変更、たとえば既存データのバックフィル、値の変換、削除前のデータ退避は、作業項目の手順または専用 SQL に明記する。アプリケーション起動時にスキーマを移行する仕組みは存在しない。メモリアダプターはテストとローカルデモの参照実装でもあるため、PostgreSQL 側だけを更新してはならない。

`postgres.sql` には SQL コメントを書かない。テーブルや列の設計根拠は、本書または所有する Context の `SPECIFICATION.md` の Design 節に置く。設計根拠の正本を一箇所に保ち、`psqldef` が文の依存順序を解決する際にコメントの影響を受けないようにするためである。詳細な運用規則は `infra/schema/README.md` を参照する。

#### Database design policy

##### 1. Column type selection

テーブルを追加するたびに判断が再現できるよう、選択の規則を固定する。

- **自由形式の文字列、長さ無制限**: `TEXT` を使う。制約のない `varchar` は決して使わない。
- **長さの上限がある文字列**: 固定の列ごとの長さ上限の方針に従い、`TEXT` + `CHECK (char_length(col) <= N)` か `varchar(N)` のいずれかを一貫して使う。書式が固定された識別子は `CHECK (... ~ regex)` で守る。
- **内部で生成する id**: idmagic が `spec.NewUUIDv4()` で生成する列は `UUID` とする。Go 側は `string` で保持し、pgx のテキスト用符号器の登録 (`RegisterUUIDAsText`) が両者を橋渡しする。
- **外部が決める id**: 値を外部が決める id (`entity_id`、`wtrealm`、`scim_id`、`kid`) は `TEXT` のままにする。idmagic が採番せず、UUID でもないからである。
- **時刻**: すべて `TIMESTAMPTZ` とし、マイクロ秒の精度を正とする。スキーマ側で丸めない。
- **有限の値集合**: `TEXT` + `CHECK (col IN (...))` とする。PostgreSQL の列挙型は避ける。値の追加に `ALTER TYPE` が必要で、宣言的なスキーマの差分取りと相性が悪いためである。
- **JSONB**: 外部の仕様に由来するメタデータ、claim とポリシーの設定、追記のみの内容に限る。結合や絞り込みが必要な値、外部キーや一意性の制約を持つ値、状態遷移に参加する値は JSONB の中に置かない。

##### 2. tenant_id retention classes

`users.id` と `oauth2_clients.client_id` はシステムが生成する全体で一意な識別子なので、子の行はその全体で一意な鍵で親を参照し、**テナント単位の複合外部キーは使わない**。全体で一意な親をたどればテナントに到達できるというだけの理由で `tenant_id` を追加してはならない。検索、制約、保持期間、監査のいずれかに役立つときに追加する。

- **テナントが所有する aggregate とテナント単位の設定**: `tenant_id` を持ち、通常は主キーか一意キーの一部にする (`users`、`groups`、`oauth2_clients`、`applications`、`agents`、`signing_keys`、 `application_categories`、`saml_service_providers`、`wsfed_relying_parties`、`*_sign_in_policies`)。
- **テナント単位で外部に由来する自然キー**: 外部の id がテナント内でしか一意でないため、`tenant_id` を主キーの一部にする (`scim_user_refs` と `scim_group_refs` は `(tenant_id, scim_id)`)。
- **全体で一意な親の子**: 全体で一意な鍵 (`user_id` と `client_id`) で識別し、テナントごとの検索や保持期間が必要でない限り `tenant_id` を持たない (`consents`、`application_orderings`、`mfa_factors`、 `password_history`、`password_reset_tokens`、`email_change_tokens`、`group_members`)。例外が 2 つある。`authentication_sessions` は、セッションの id がすべてのリクエストで解決される中身の見えない cookie の値なので、`tenant_id` はその照合における fail-closed な多層防御の条件であり、同時にテナントごとの有効なセッション一覧のためのインデックスでもある。中身の見えないトークン、code、challenge を鍵とする一時的な認証と OAuth2 のストア (`oauth2_authorization_requests`、`oauth2_authorization_codes`、 `oauth2_par_requests`、`oauth2_device_codes`、`oauth2_replay_jtis`、 `oauth2_access_token_denylist`、`webauthn_sessions`、`login_throttle_counters`、 `saml_authnrequest_replays`、`endpoint_rate_limit_counters`) も同じ理由で保持する。どの照合も攻撃者が影響を与えられる中身の見えない鍵に対する高頻度で fail-closed な解決なので、テナントの境界をデータベースの層で強制する。
- **追記のみ、監査、流量の抑制**: 発行時のテナント、問い合わせの境界、保持期間から判断する (`audit_events`、`authentication_event_buckets`)。

##### 3. Envelope encryption for reversible secrets

アプリケーションのデータベースに残す必要がある可逆なシークレットは、決して平文で保存しない。方式は 2 段である。差し替え可能な `EnvelopeCrypto` プロバイダーが保持するマスターキーでテナントごとの `DataEncryptionKey`（DEK）をラップし、その DEK で各シークレットを直接 AEAD 暗号化する。AEAD と鍵セットの扱いはすべて [Tink](https://developers.google.com/tink) に委ね、nonce、認証タグ、追加認証データの組み立てを自作しない。すべての暗号文は `(tenant, context, table, record id, field)` と DEK のバージョンを追加認証データとしてバインドするため、テナント、テーブル、フィールドの境界を越えて複製した暗号文は復号に失敗する。

- `EnvelopeCrypto`（Tink を使う AEAD と鍵セットのポート、および OpenBao と平文鍵セットによるマスターキー提供元のアダプター）は、`certificates_mtls`、`passwords_argon2id`、`tokens_jose` と並べて `backend/shared/security` に置く。これは技術的な能力であり、業務上の Aggregate ではない。
- `backend/datakeys`（`DataKeys` Context）は、ラップされた DEK のメタデータとライフサイクル（初期化、ローテーション、無効化、破棄）だけを所有し、`EnvelopeCrypto` ポート自体は所有しない。`SigningKeys` が `transit/sign` を暗号化、復号、データ鍵の機能から分離しているのと同じ構成である。
- ローテーションでは新しい DEK の版を以後の書き込み用に有効化し、直前の版を復号可能な `retiring` のまま残す。`backend/jobs` の `JobKind` と `HandlerRegistry` に登録した再開可能な再暗号化ジョブがすべての参照を移行し終えた後にだけ、古い版を破棄できる。`FieldMigrator` ポート（`backend/datakeys/ports`）により、各 Context は自身の一括再暗号化処理と残件数の算出を登録する。これにより、`DataKeys` は利用側のスキーマへ依存しない。ローテーションは登録された移行処理ごとにジョブを自動投入し、いずれかの移行処理が残件を報告している間はラップされた DEK の消去を拒否する。
- 復号の失敗 (包みを解けない、提供元へ到達できない、追加認証データの不一致や改竄) は fail-closed である。呼び出し側は平文へ退避したり項目を読み飛ばしたりせず、アクセスを拒否する。
- マスターキーの提供元は OpenBao（Vault Transit 互換の HTTP API）である。開発環境とローカル環境では Tink の平文鍵セットを使うため、OpenBao は不要である。提供元は設計上差し替え可能である。
- 唯一の HTTP 接点は、読み取り専用で `system_admin` に限定した `GET /api/admin/data-keys/health`（`backend/datakeys/handlers_http`）である。各テナントで有効な DEK の版とステータス、マスターキー提供元の名前と到達性を報告し、鍵素材は決して返さない。ローテーション、無効化、破棄は内部操作とし、管理用エンドポイントを公開しない。

`tenant_data_encryption_keys.wrapped_dek` は破棄時に行を削除するのではなく消去 (`NULL` に設定) する —暗号による細断である。これにより鍵素材そのものが失われた後も、DEK のライフサイクルの履歴 (`status` が `active` / `retiring` / `disabled` / `destroyed` と遷移した記録) を問い合わせられる。

##### 4. Cross-context schema integration notes

以下は複数の Context にまたがる永続化の選択を比較する記述である。個別の Context の文書や、コメントを持たない `infra/schema/postgres.sql` には複製せず、リポジトリ全体のデータベース方針とともに置く。

- **Tenancy** — `tenant_brandings` は 1 対 1 で対応する外装設定であり、`tenants` の列ではなく独立したテーブルに置く。機能ごとの設定が増えても中核のテナントの行を肥大させないためである。 `tenant_branding_assets` と `tenant_user_attribute_schemas` も同じ理由に従う。`tenant_brandings` の行がない、あるいはすべての列が `NULL` であることは外装が未設定であることを意味し、呼び出し側はシステムのデフォルトへ退避する。`tenant_branding_assets` はロゴとファビコンの画像データを `application_icons` と同じ形で検証・保存するが、テーブルと `object_key` の空間を分けているので、外装の資産の所有が Application のアイコンの保存と交差することはない。`notification_templates` は通知メールのカタログに対するテナントの上書きを `(tenant_id, template_key, locale)` で保持する。解決の仕組みは [Notification template catalog and locale resolution](#5-notification-template-catalog-and-locale-resolution) を参照。 JSONB ではなく個別の列にすることで、列ごとの長さの上限を `CHECK` 制約として保てる。 `subject` / `body_text` / `body_html` はまとめて `NOT NULL` なので、半分だけ上書きされたひな形は存在し得ない。一方 `from_display_name` は `NULL` を許す。システムデフォルトの送信者名が妥当な選択肢だからである。`tenants.default_locale` は通知の言語の解決 (受信者 → テナント → システム) のテナントの段であり、`NULL` は「システムのデフォルトを使う」を意味する。`tenants.endpoint_style` はテナントの正式な所在の形を固定する (issuer、cookie の適用範囲、WebAuthn の RP ID はいずれもここから導かれる)。デフォルトが `'path'` なのは、ワイルドカードの DNS も証明書も要らないためである。
- **OAuth2** — `oauth2_client_secrets` は `client_secret` の資格情報をクライアント本体の行から分離する。`refresh_tokens.sid` は OIDC のブラウザーセッションを表し、`authentication_sessions.id` と同じ値を持つ。`client_credentials` などブラウザーセッションを伴わない発行では `NULL` になる。セッションの保持期間に基づく物理削除とリフレッシュトークンの失効を独立させるため、`authentication_sessions` への外部キーは設定しない。`refresh_tokens.resource`（RFC 8707）は認可コードの交換時にバインドしたリソース指示子であり、ローテーション後も保持する。`NULL` は `resource` が指定されなかったことを表す。`mcp_resource_servers` は、ツールやデータを提供する MCP リソースサーバーのテナント単位の登録である。`resource` はテナント内で一意な正規リソース URI であり、Protected Resource Metadata（RFC 9728）とリソース指示子（RFC 8707）の検証基準になる。`oauth2_authorization_requests` は `/authorize` の処理中の状態を `payload` の JSONB に保持し、単一トランザクション内の `SELECT ... FOR UPDATE` で遷移を直列化する。`oauth2_authorization_codes` は `UPDATE ... WHERE state = 'issued' RETURNING` による比較交換で一度だけ消費する。`oauth2_par_requests` も `used` を比較交換の条件として一度だけ消費する。`oauth2_device_codes` は `device_code_hash` を鍵とし、`user_code` をテナント内で一意な副次的な照合先とする。承認時に `user_id` を設定し、交換時は `state` を比較交換の条件とする。`oauth2_replay_jtis.kind` は `dpop` と `client_assertion` の再送防止を区別し、`INSERT ... ON CONFLICT DO NOTHING` で初回利用だけを記録する。`oauth2_access_token_denylist` は、切り替え時にも失効情報を失わないよう `LOGGED` テーブルとする。
- **Audit** — `audit_event_search_attributes` は付随する検索用のインデックスであり、 `(event, attr_name, 変換後の値)` ごとに 1 行を持つ。`attr_name` は `AuditSearchRegistry` の `Field` である。PII の属性はここへ保存する前に要約値にするか丸める。平文は `audit_events.payload` にのみ存在し、それも失敗のイベントについて短い保持期間の下でのみである。`audit_events` の削除に連鎖し、照合用のインデックスは等価一致のために `(tenant_id, attr_name, attr_value)` を、走査のために `occurred_at DESC` を並べる。
- **ApiTokens** — `api_tokens` は管理対象の RFC 9068 JWT アクセストークンのライフサイクル記録を保持する。JWT 本体は保存せず、`jti` を照合キーとする。`scopes` は許可された `<resource>:<action>` 形式の権限を列挙する（`spec/contexts/api-tokens/models.tsp` の `ApiTokenScope`）。テーブルの `CHECK` 制約にも同じ列挙を反映し、Go 側の検証と合わせて多層防御とする。
- **IdGovernance** — `lifecycle_workflow_revisions` は追記のみである。実行の記録 (`lifecycle_workflow_runs` と `_steps`) は可変な JSON ではなく、展開元の版を参照する。
- **Provisioning** — `provisioning_connections.credential_secret` は開発およびテスト専用の平文列であり、本番環境ではこの列に依拠しない。

##### 5. Notification template catalog and locale resolution

通知メールの内容は、版の履歴を持たない厳密に 2 段で解決する。組み込みのカタログ (システムが同梱する日本語と英語の文面) と、任意の `(tenant_id, template_key, locale)` ごとの上書きである。上書きの削除 (`ResetNotificationTemplate`) は常に組み込みのデフォルトへ戻る。「直前の上書きへ戻す」段階はない。ひな形が復旧の流れを壊したとき、管理者にとって最速の修正は版の選択ではなく既知の正常な退避先だからである。 `template_key` は仕様上の固定された列挙であり — テナントが鍵を追加することはできない — すべての鍵がちょうど 1 つの送信経路へたどれる。送信者のいない孤立したひな形は存在し得ない。

差し込みの記号 (`{{name}}`) は保存時に鍵ごとの許可一覧と照合する。宣言されていない差し込みを参照する上書きは、値を空にして描画するのではなく、その場で拒否する。実行時に空になった導線は、利用者がアカウントを復旧できずに初めて発見されるからである (fail-closed)。許可一覧は `backend/shared/notification/template` で定義し、API が返すので、編集者が推測する必要はない。

| Key | Placeholders |
| --- | --- |
| all keys | `product_name`, `tenant_display_name`, `user_display_name` |
| `PasswordReset`, `EmailVerification`, `EmailChangeConfirmation` | 1 つの `*_url` の導線, `expires_in_minutes` |
| `EmailChangeConfirmation` (additional) | `new_email` |
| `LifecycleWorkflowNotification` (additional) | `notification_key` |
| `AccountSecurityAlert` (additional) | `event_description`, `occurred_at` |

資格情報、ダイジェスト値、TOTP シークレット、API トークン、生の IP アドレスは決して差し込みにしない。メールは受信者によって転送され、引用され、無期限に保持されるため、そこに置いたものは後の受信箱の侵害ですべて露出する。

描画側の契約は件名、本文の平文、本文の HTML を 1 つの単位としてまとめて返す — 3 つのうち 1 つや 2 つだけが存在する状態はない — し、上書きも同様に 3 つを一度に置き換え、`multipart/alternative` として送る。これにより、片方だけを編集したせいで 1 通のメールの 2 つの部分が黙って食い違うことがなくなる。特殊文字の処理はひな形ではなく描画側の責務である。HTML の出力は差し込んだ値を無害化し、平文の出力はしない。導線の URL は呼び出し側の ユースケース がリクエスト自身の issuer から組み立て、単一の差し込みの値として渡す。したがってひな形は URL を配置できるが連結はできず、無害化の義務がテナントの編集者へ及ぶことはない。上書きできる項目は件名、本文の HTML の断片、本文の平文、送信者の表示名に限る。HTML 文書の外枠と送信元のアドレスはシステムが所有し続けるので、悪意あるテナントの管理者が上書きの仕組みを通じてできる最悪のことは、自分のテナントのメールの見た目を壊すことだけであり、そこへ注入することは決してできない (外装の設定が使うのと同じ分割である)。

言語は受信者の `User.locale` → テナントの `default_locale` → システムのデフォルト (`DEFAULT_LOCALE` の環境変数、デフォルトは `en`) の順に解決し、カタログが翻訳を持つ最初の言語を採る。テナントの段を、どの言語に上書きが存在するかから推し量るのではなく明示的な列にしているのは、1 つの言語で 1 つのひな形を編集したことが、すべての通知の言語を黙って変えてしまわないようにするためである。

試し送りは操作している管理者自身の確認済みのアドレスにのみ配送する — エンドポイントは宛先を受け取らない。任意の宛先を許せば、テナントの管理者の権限が、テナントの外装をまとったメールを誰にでも送る手段になってしまうからである。下書きの確認は読み取り専用で、実際の利用者のデータではなく固定の見本の値で描画するので、編集画面が利用者のデータを読む手段になることはない。

##### 6. Endpoint rate limiting

`backend/shared/ratelimit` (`ports`、`db_memory`、`db_postgres`) は業務上の aggregate ではなく技術的な能力である — `backend/shared/security` の `EnvelopeCrypto` と同じ配置である。OAuth2 と Authentication の両方の context にまたがるエンドポイント (`/authorize`、`/token`、`/par`、`/device_authorization`、 `/bc-authorize`、`/api/auth/password_reset/*`) を保護するからである。上述のアカウント単位・IP 単位のログインのスロットルとは別物であり、それを置き換えるものでもない。

ポートは単一の `Allow(ctx, tenantID, policyID, key, now)` の呼び出しである。`(tenant_id, policy_id, key_hash)` を鍵とする固定された時間枠のカウンターで、結果によらずリクエストごとに 1 回加算する (失敗だけを数えるログインのスロットルとは異なる)。`endpoint_rate_limit_counters` は `UNLOGGED` である。すべてのリクエストがこれに計上されるため一時的なテーブルの中で最も更新が激しく、切り替え時にカウンターを失っても時間枠が戻るだけで、安全性の保証が弱まるわけではないからである (これに対し `login_throttle_counters` と access トークンの拒否リストは、失えば保証が弱まるので `LOGGED` のままにする)。 fail-closed は一様に適用する。保護対象のすべてのエンドポイントにとって PostgreSQL は既に必須の依存なので、ストアへ到達できないときに拒否しても新たな障害の型は増えない。

閾値はポリシーごとの固定された時間枠 `(最大リクエスト数, 時間枠の秒数)` であり、環境変数で設定できる (`server.go` に直接書かれたままのログインのスロットルとは異なる)。運用者が配備なしで調整し直せる。

| Policy | Env (max / window) | Default |
| --- | --- | --- |
| `token` | `RATE_LIMIT_TOKEN_MAX_REQUESTS` / `RATE_LIMIT_TOKEN_WINDOW_SECONDS` | 60 / 60s |
| `authorize` | `RATE_LIMIT_AUTHORIZE_MAX_REQUESTS` / `RATE_LIMIT_AUTHORIZE_WINDOW_SECONDS` | 30 / 60s |
| `par` | `RATE_LIMIT_PAR_MAX_REQUESTS` / `RATE_LIMIT_PAR_WINDOW_SECONDS` | 30 / 60s |
| `device_authorization` | `RATE_LIMIT_DEVICE_AUTHORIZATION_MAX_REQUESTS` / `RATE_LIMIT_DEVICE_AUTHORIZATION_WINDOW_SECONDS` | 20 / 60s |
| `backchannel_authentication` | `RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_MAX_REQUESTS` / `RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_WINDOW_SECONDS` | 20 / 60s |
| `password_reset` | `RATE_LIMIT_PASSWORD_RESET_MAX_REQUESTS` / `RATE_LIMIT_PASSWORD_RESET_WINDOW_SECONDS` | 5 / 900s |

キーは `client_id`、IP、`identifier_hash` の組み合わせである。`client_id` はシークレットではないが、IP とパスワード再設定の識別子は保存前に SHA-256 でダイジェスト化する。ログインスロットルの `hashThrottleIdentifier` と同じ方式である。閾値を超えると、`Retry-After` と TypeSpec の `RateLimitedError` を伴う HTTP 429 を返す。

### Runtime Composition

`backend/cmd/idmagic/` の main パッケージが起動を行い、`backend/cmd/internal/bootstrap` が起動時の依存注入を所有する。`backend/cmd/idmagic-worker/` は永続化されたジョブを取得してハンドラーを走らせるだけであり、API とは独立に水平に台数を増やせる。`backend/cmd/idmagic-batch/` は外部の予定表から 1 回限りで起動され、保持期間の一掃か署名鍵のライフサイクルの処理を 1 度行って終了する。どのランタイムの単位も同じ Go のモジュールと Bounded Context の実装を再利用する。ランタイムの単位は台帳に列挙するのではなく、起点となるプログラムと対応する `just` の構築手順から導く。

すべての Bounded Context の実装を単一の Go モジュールに置き、複数の実行単位を共通実装を再利用する薄いエントリーポイントとする現在の形は、**Modular Monolith** である。Context の論理境界は厳密に保ち、Context 間は公開された言語とポートを通じて接続する。デフォルトでは複数の Context を 1 つのプロセスに組み合わせる。現在の実行単位の分割は、認証と OAuth2 の同期的な依存を API プロセス内に留めたうえで、リソースとレイテンシーの特性（レーンごとの `worker`）および横断的なバッチ処理の実行境界に限る。組織上の契機はまだ生じていないため、独立したデータ所有、チーム、SLO がそろうまではサービスを分割しない。これは現状の記述であり、将来の様式を規定するものではない。

`backend/cmd/internal/bootstrap/deps.go` の `Dependencies` は HTTP 層へ渡す境界の集約であり、メモリ、PostgreSQL、コンソール、OpenTelemetry といった実行時の選択を吸収する。Context 固有の Repository は各 `Module` に束ね、中央の `Dependencies` とサーバーの `Deps` がその Module を受け取る。ポートを追加した後は、少なくともその Context の `ports/`、メモリアダプター、PostgreSQL アダプターとスキーマ差分の要否、`bootstrap.Dependencies`、`assembleMemory` と `assemblePostgres`、`support.Deps`、関係する HTTP ハンドラーまたはユースケースの構築処理を確認する。

#### Health probes and graceful drain

Kubernetes に面する健全性は、生存確認と受付可否で 1 つを共有するのではなく 3 つのエンドポイントに分ける。元の `/health` は起動時の設定のラベルを返すだけだったので、両方に使うと PostgreSQL の一過性の不調がその pod を再起動の繰り返しに陥れつつ、実際には応答できないレプリカに通信を流し続けることになった。受付可否は、その依存が取り除かれる前は共有の Valkey のストアにも問い合わせていた。

- **`/livez`** は行き詰まりのような回復不能な状態でのみ失敗する。一過性の依存の障害では `200` を返し続けるので、放っておけば回復する pod を生存確認が再起動しない。
- **`/readyz`** は必須の依存 (PostgreSQL) へ短い制限時間 (デフォルト `1s`) で並行に問い合わせ、到達できなければ `503` を返す。`?verbose` を付けると依存ごとの状態の語彙 (`healthy` / `degraded` / `unavailable`) が加わる。
- **`/startupz`** はアプリケーションの初期化 (初期データの確認を含む) が完了すると `200` を返す。
- **`/health`** は後方互換のために残しており、従来どおり起動時の設定のラベルだけを返す。

`SIGTERM` と `SIGINT` を受けると停止の印が立ち、`/readyz` は直ちに `503` (`unavailable`) を返し始める。プロセスが接続の受け付けを止める前に、負荷分散装置が振り分けを外す時間を与えるためである。その後サーバーは退避の猶予期間 (`DRAIN_GRACE_PERIOD_SECONDS`、デフォルト `5s`) を待ってから、HTTP サーバー自身の停止を開始する。

#### Availability and shared state

レプリカを複数動かすには `postgres` のランタイム (`PERSISTENCE=postgres`、`DATABASE_URL`) が必要である。共有される状態は永続的なものも一時的なものも、レプリカごとのプロセスのメモリではなくすべて PostgreSQL に置く。

- **永続的**: リフレッシュトークン、監査イベント、認証イベントの集計バケット、ログインセッション。ログイン済みのブラウザーセッションは `authentication_sessions` を唯一の正とするため、API レプリカを再起動または順次入れ替えても有効なセッションは失われない。利用者の操作、ログアウト、アカウントの無効化による失効では行を削除せず、`revoked_at` と `revoke_reason` を記録する。このため、失効リクエストを再送しても安全である。
- **一時的**: 認可リクエスト、認可コード、PAR、デバイスコード、DPoP とクライアントアサーションの再送防止、アクセストークンの拒否リスト、WebAuthn のチャレンジ、ログイン試行のスロットル、エンドポイントのレート制限カウンター。いずれも短命で、再試行しても安全である。すべての行が `expires_at` を持ち、読み取りを `expires_at > now()` で絞り込むため、有効期限の正しさは `idmagic-worker` が領域回収のために行う最大限努力の掃除に依存しない。

とりわけログインのスロットルは共有されなければ *ならない*。レプリカごとのカウンターでは、攻撃者の失敗の試行が `N` 個のレプリカに分散するので、アカウント単位と IP 単位の締め出しの閾値が全体では最大 `N` 倍に緩む — 気づかれない安全性の後退である。共有された PostgreSQL のカウンターでは、直列化された `SELECT ... FOR UPDATE` の更新で全体として数え、アカウントと IP の識別子は SHA-256 で要約するので、平文の利用者名や IP は保存されない。

スロットルは重要な経路上にあるため、その劣化は **fail-closed** である。ストアへ到達できない場合、スロットルの状態を確認できないログインの試行は通すのではなく拒否する。複数レプリカの配備では、この経路が落ちないよう PostgreSQL を高可用の構成 (地域冗長、同期的な待機系) で運用すること。

`memory` のランタイムはこの状態をプロセス内に保持するので、**単一レプリカとテスト専用**である。

### Specification publishing

正式な Markdown と TypeSpec は、`spec/generated/docs/index.html` を起点とする、追跡対象外で再現可能な静的サイトとして公開する。このサイトは方法の手引き、本書、各 Bounded Context の文書、API の参照、 TypeSpec のモデル目録を、それぞれ独立に指し示せるページへ分ける。内部の導線は生成された単一のページ目録から解決し、切れた導線や到達できないページがあると生成が失敗する。

API の操作とワイヤ表現の提示は、生成された OpenAPI の文書を読むリポジトリ内の Swagger UI に委ねる。仕様の描画側は OpenAPI の第二の解釈を保守しない。モデルの目録はそれとは別で、リポジトリが所有する TypeSpec のモデル、列挙、合併、スカラーの宣言から導く。HTTP の操作から到達できない宣言も含める。 `Operations` の名前空間にある伝送用の包みは、目録のページとして重複させず API の参照に留める。

正式な文書の Mermaid の記述は、リポジトリ内の資産で描画する。状態遷移の図は規範的な `From / Event / Guard / To / Effects` の表から導くので、図と表が別々の正になることはあり得ない。シナリオの語は、正式な Markdown の文法を変えないまま、意味を持つ HTML と視覚的なラベルを受け取る。このサイトは配信網やホスティングへの依存を持たず、仕様のもとではなく派生的な閲覧用の見え方であり続ける。

### Structural Decisions

- `backend/` と `frontend/` は別々の配備成果物の境界である。Go のエントリーポイントは `backend/cmd/` に置き、ランタイムの組み立てが Context のパッケージへ漏れないようにする。
- ランタイム、コンテナ、Kubernetes、データベース基盤の資産は `infra/` に置き、運用上の組み立てをアプリケーションのソースから分離する。
- 技術的な共有 Context を Context が所有するアダプターから分離し、Context 固有の永続化アダプターはその Context とともに置く。Context 固有の概念をため込んだ共有パッケージは、以後あらゆる変更で読むべき範囲を広げるからである。
- エンドポイントの流量制限は業務上の Aggregate ではなく共有の技術的能力とし、トークンバケットや移動窓ではなく固定時間枠を使い、フェイルクローズかつ PostgreSQL だけで動かす。
- 要求 ID と TypeSpec シンボルにより、リポジトリ全体の追跡表を置かずに仕様を直接参照できる。テストと作業項目は、証跡として有用な箇所でそれらの安定した名前を引用する。生成サイトは、人が保守する索引ではなく、その引用からシナリオとコードの対応を導く。
- リポジトリのパスとインポートが実行可能な構造である。チェックは、許可されるモジュールと依存を二重に宣言させることなく、禁止された外向きの依存を拒否する。
- LifecycleWorkflow のポリシーとオーケストレーションは IdGovernance が所有し、IdManagement はユーザーとグループの記録の正であり続ける。これによりガバナンスポリシーを記録の Context の外に保つ。
- 環境ごとの初期データ方針と実行の組み立ては記録の Context から分離し、各 Context の公開コマンドを通じて適用する。
- 外向きのプロビジョニング（SCIM クライアント）は、内向きの SCIM サーバーとは別の Context とする。情報の正となる側が逆で、ライフサイクルも異なるためである。`User` やアプリケーション割り当ての変更を保存するときは、同じ PostgreSQL トランザクションで `ProvisioningDelivery` も保存し、変更だけが確定して配信予定が失われる事態を防ぐ。
- 内向きのアイデンティティ取り込みは、方向や実行時の形ではなく、永続的な取り込み元とのバインディングを持つ権威の有無でまとめる。取り込み元ごとに 1 つの機能単位を持つ単一の `Sourcing` Context とし、取り込み元に依存しない中核は 2 つ目の取り込み元が現れるまで作らない（薄いルート）。
- データベースに残る可逆なシークレットのエンベロープ暗号化では、技術的な `EnvelopeCrypto` ポート（Tink の AEAD と鍵束、主鍵のプロバイダー）を、業務に面したテナントごとの DEK のライフサイクルから分ける。これは `SigningKeys` が `transit/sign` に使うのと同じ分割である。ポートは `backend/shared/security` に、ライフサイクルは `DataKeys` Context に置き、どちらも `SigningKeys` には統合しない。`SigningKeys` の `KeyStore` ポートは操作の形もライフサイクルも異なるからである。
- Dynamic Group のメンバーシップの規則は、独自の式の言語や本格的な処理系ではなく、制限された CEL の環境で評価する。
- `ApiTokens` は SCIM と管理 API のアクセストークンを、別々のトークン種別ではなく、単一の発行方式とスコープモデルに統一する。
- `WorkloadIdentity` は外部ワークロードのアテステーション（JWT-SVID）を、独立した資格情報体系ではなく、OAuth2 のトークン交換を介して idmagic のトークンへ連携する。
- アダプターのパッケージは所有する Context または機能の直下に平らに置き、`<role>_<technology>` で命名する。`adapters/` や `persistence/` のような分類ディレクトリは置かず、パッケージ名だけで役割が分かるようにする。
- Context 固有の業務上の型は共有パッケージではなく、その Context の `domain/` が所有する。
- 独立した部分領域を持つ Context は、機能ごとの垂直分割（`backend/<context>/<feature>/{domain,ports,usecase,<role>_<technology>}/`）を追加してよい。`idmanagement` はこの構成を採用する。
- Context 固有の Repository とルーティングの接続は `backend/<context>/module.go` に集約し、中央のルーターは Module を呼ぶだけにする。
- エラー応答の本体は RFC 9457 Problem Details へ移行する。ただしプロトコルの仕様が独自のエラーの形を義務づける箇所 (OAuth2、SCIM、DCR) は除く。
- 指し示せる管理用の一覧の位置は、開始位置の数値や短命な継続のトークンではなく、期限切れしない双方向の署名済みキーセット方式のカーソルを使う。
- 監査のログ (不変、長期保持、法務や SIEM の証跡) とアプリケーションのログ (運用向け、`trace_id` / `span_id` / `request_id` を持つ標準出力の JSON Lines) は、単一のログの流れにまとめず、保存先も保持期間も異なる 2 つの系統として保つ。
- PostgreSQL の列の型の選択は固定された規則の集合に従う (自由形式と上限付きの文字列は `CHECK` 付きの `TEXT`、内部で生成する id は `UUID`、時刻は `TIMESTAMPTZ`、列挙型より `TEXT`+`CHECK`、JSONB は仕様に由来するものと追記のみの内容に限る)。テーブルを追加するたびに選択が再現できるようにするためである。
- `users.id` と `oauth2_clients.client_id` は全体で一意なので、子の行はそれらを直接参照し、テナント単位の複合外部キーは使わない。
- `authentication_sessions` と一時的な認証・OAuth2 の中身の見えない鍵のストアは、親の鍵が既に全体で一意であっても `tenant_id` を fail-closed な多層防御の条件として保持する。これは、それ以外では全体で一意な親の鍵を直接使う方針に対する意図的な例外である。
- 揮発的な認証と OAuth2 の状態（認可リクエスト、認可コード、PAR、デバイスコード、再送防止、拒否リスト、WebAuthn のチャレンジ、ログイン試行のスロットル）は、別の状態ストアを設けず PostgreSQL に統合する。運用するデータストアを 1 つに保つためである。
- 通知メールの内容は版の履歴を持たない厳密に 2 段 (組み込みのカタログ、任意のテナントごとの上書き) で解決し、上書きできる項目を件名・本文・送信者の表示名に限る。これにより悪意あるテナントの管理者にできるのは自分のメールの見た目を壊すことだけで、共有の外枠へ注入することは決してできない。外装の設定が使うのと同じ、システムが持つ外枠とテナントが持つ内容の分割である。
- `application_icons` とテナントの外装の資産は、テーブルと `object_key` の空間を分けて保持する。両者の所有が交差しないようにするためである。
- テナントはパス接頭辞またはサブドメインのどちらか一方による正規の所在を持ち、その所在から発行者、Cookie の適用範囲、WebAuthn の RP ID を決定する。デフォルトを `'path'` とし、ワイルドカード DNS や証明書を不要にする。
- `refresh_tokens.sid` は、リフレッシュトークンを OIDC のブラウザーセッションへ外部キーなしで関連付ける。セッションの保持期間に基づく削除を、トークンの失効状態とは独立に行うためである。`refresh_tokens.resource` と `mcp_resource_servers` は、idmagic が MCP 認可サーバーとして RFC 8707 のリソース指示子に対応するために使う。
- `idmagic-worker` は永続化されたジョブを取得して実行し、API とは独立に台数を増やせる。`idmagic-batch` は保持期間と鍵のライフサイクルに関する定期処理を 1 回実行して終了する。ジョブは資源とレイテンシーの特性を分離するため、レーンごとの `worker` プロセスで実行する。
- Kubernetes の健全性は生存確認と受付可否を 1 つのエンドポイントで共有せず、`/livez` と `/readyz` と `/startupz` に分ける。一過性の依存の不調が、pod を再起動の繰り返しに陥れながら応答できないレプリカに通信を流し続けることがないようにするためである。

### Documentation Policy

何を書きたいかが決まったら、この表がその置き場所を決める。軸は、各文書が答える問いである。

| What you want to write | Where it goes | Question it answers |
| --- | --- | --- |
| Overview, requirement, scenario, glossary, standard, state transition | `spec/**/SPECIFICATION.md` | 何が成り立たねばならないか、Context が何を意味するか |
| Model, API contract, HTTP binding, authentication | `spec/**/*.tsp` | 何が交換されるか |
| Current design of one context | その `spec/contexts/<context>/SPECIFICATION.md` の Design 節 | 今どうなっていて、なぜそうなのか |
| Cross-cutting design, conventions, cross-cutting policy | 本書の Design 節 | 同じことを、Context をまたぐ事柄について |
| Forbidden dependency rules | `tools/check/src/check-boundaries.ts` | どの外向きのインポートが拒否されるか |
| How to use or run something | そのディレクトリの `README.md` | どう使うか、どう動かすか |
| What to do when something happens | `infra/runbooks/*.md` | 障害時に何をするか |
| A one-off implementation record | `work-items/` | 今回何をしたか |

過去の記録は現在の仕様の道案内の外に置く。

コードやスキーマから機械的に読める一覧を手で写さない。網羅的なエンドポイントの表、テストの一覧、環境変数の表は作らない。

文書とコードコメントの言語は `AGENTS.md` の `Language` 節が定める。本書と各 Context の `SPECIFICATION.md` の散文は日本語で書き、見出し、表の見出し行、識別子、標準名称、仕様が定める語、設計パターン名は原語のままにする。
