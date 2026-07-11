---
context: repo
updated_at: 2026-07-11
modules:
  backend:
    path: backend
    responsibility: Go の bounded contexts、共有 adapter、起動 entry point を提供する。
  frontend:
    path: frontend
    responsibility: React の browser UI と gateway 配信設定を提供する。
  specification:
    path: spec
    responsibility: SCL の規範仕様と派生契約を提供する。
---

# Architecture: repo

## Overview

この文書は、AI エージェントが `idmagic` の変更に必要な文脈を小さく取得するための索引である。人間向けの包括的な設計説明ではない。詳細な仕様は SCL、判断理由は ADR、完了済みの変更履歴は work item を読む。

更新コストを抑えるため、ここには頻繁に増減するエンドポイント一覧・フィールド一覧・画面一覧を置かない。それらはコード、`spec/contexts/*.yaml`、`README.md`、UI 側の文書を正とする。

## Structure

```text
.
├── backend/       # Go bounded contexts、shared adapter、起動 entry point
├── frontend/      # React UI と gateway
├── spec/          # SCL と派生契約
├── deploy/        # コンテナ・ローカル実行・schema 資材
├── decisions/     # Architecture Decision Records
└── work-items/    # 作業単位と完了記録
```

依存は `spec` から各実装・派生物へ向かい、`backend` の domain/usecases は adapter と runtime へ逆依存しない。

## Stack

- Go、React/TypeScript、Bun、PostgreSQL、Valkey、Docker Compose。

## Structural Decisions

- `backend/` と `frontend/` の成果物境界および Go entry point の配置は [ADR-092](decisions/ADR-092-backend-and-frontend-top-level-directories.md) に従う。
- technical shared context と context-owned adapter の分離、および context 固有の永続化 adapter の同居は [ADR-070](decisions/ADR-070-technical-shared-context-for-cross-context-adapters.md) と [ADR-090](decisions/ADR-090-context-local-persistence-and-sqlc.md) に従う。

## 読む順序

機能変更では次の順に読む。

1. `spec/scl.yaml` の `context_map` で対象 bounded context と依存先を特定する。
2. 対象 context の `spec/contexts/<context>.yaml` を読む。機能追加・挙動変更は SCL-first で行う。
3. 該当 ADR を読む。迷ったら `decisions/` をファイル名検索し、古い work item の要約だけで判断しない。
4. Go 実装は対象 context の `domain/`、`usecases/`、`ports/`、`adapters/` の順に読む。
5. HTTP や永続化の横断挙動を触る場合だけ `backend/shared/` と `backend/bootstrap/` を読む。
6. UI を触る場合は `frontend/ARCHITECTURE.md` と `frontend/src/features/README.md` を先に読む。

実装から仕様へ逆引きする場合は、パッケージ名と SCL context 名がほぼ対応する。例外的な共有物は `backend/shared/` に集約される。

## RA レイヤ対応

`idmagic` は Regenerative Architecture の同心円を Go の package 境界で表す。

| RA レイヤ | 保存・実装場所 | 読み方 |
| --- | --- | --- |
| Specification Core | `spec/scl.yaml`, `spec/contexts/*.yaml` | 規範仕様。変更は原則ここから始める。 |
| Decision Record | `decisions/*.md` | SCL だけでは分からない採用理由・除外理由。 |
| Application Logic | `backend/<context>/domain`, `backend/<context>/usecases`, `backend/shared/spec` | フレームワーク非依存のドメイン・ユースケース・SCL binding。 |
| Adapter Layer | `backend/<context>/adapters`, `backend/shared/adapters` | HTTP、persistence、crypto、policy、notification など外界との接続。 |
| Runtime & Infrastructure | `backend/cmd/`, `backend/bootstrap`, `deploy/`, `frontend/`, `docker compose` | 起動、DI、配信、プロセス境界。 |

`backend/shared/spec` は SCL の Go binding と派生検証であり、仕様核そのものではない。SCL の内容を変える代わりに Go binding だけを調整しない。

## Context Map

SCL context と Go package の主な対応は次の通り。

| SCL context | Go package | 主な責務 |
| --- | --- | --- |
| `System` | `backend/bootstrap`, `backend/shared/adapters/http/server`, `frontend/` | 横断 UX、起動、ルーティング集約、health。 |
| `Tenancy` | `backend/tenancy` | tenant / realm、tenant-scoped settings、user attribute schema、control-plane tenant 管理。 |
| `IdentityManagement` | `backend/identitymanagement` | User、Group、Agent、自己プロフィール、identity lifecycle。 |
| `Authentication` | `backend/authentication` | 資格情報検証、MFA、ログインセッション、step-up、パスワード変更・リセット、認証イベント。 |
| `OAuth2` | `backend/oauth2` | OAuth 2.0 / OIDC protocol endpoint、client、consent、token、audit、role policy。 |
| `Application` | `backend/application` | Application catalog、protocol binding、assignment、portal ordering/category。 |
| `ClaimMapping` | 現状は protocol context と persistence adapter に分散 | Claim release policy の概念境界。protocol-neutral へ切り出すときは SCL を先に調整する。 |
| `Scim` | `backend/scim` | SCIM 2.0 Inbound Provisioning サーバー、外部プロバイダからのユーザー・グループ同期、Bearer Token 認証、soft-delete 統合。 |
| `SigningKeys` | `backend/oauth2`, `backend/shared/adapters/crypto`, persistence adapters | 鍵ライフサイクルの規範は SCL。JWK/JWT/XML signer は adapter。 |
| `WsFederation` | `backend/wsfederation` | WS-Fed passive、WS-Trust active STS、federation metadata、MEX、RP trust。 |
| `Saml` | `backend/saml` | SAML 2.0 IdP、SP trust、metadata、SSO/SLO。 |

context 間の公開語彙と依存は `spec/scl.yaml` の `context_map` が正である。新しい依存を追加する場合は、直接 import を増やす前に context map の `depends_on` を見直す。

## Go Package Conventions

各 bounded context は原則として次の形を取る。

```text
backend/<context>/
  domain/      # エンティティ、値オブジェクト、状態機械、純粋な検証
  usecases/    # 仕様上の操作を実行するアプリケーション論理
  ports/       # repository、store、外部 service への抽象
  adapters/    # HTTP、wire format、外部 protocol 固有処理
```

`domain/` は Echo、PostgreSQL、Valkey、HTTP request/response を知らない。`usecases/` は `ports/` に依存し、具体 adapter には依存しない。`adapters/http` は入力の wire 変換、HTTP status、cookie/header、CSRF/Origin など境界処理を持つ。`usecases/` が adapter を import しない依存方向は全 context 共通で、外界の能力（署名・割当ゲート・認証解決など）は `ports/` の抽象か usecase パッケージ内の interface で受け、adapter が具体実装を注入する（例: `oauth2` の `ports.TokenIssuer`、`saml` / `wsfederation` の `ApplicationGate` interface）。

`domain/` と `usecases/` の有無は「その context 固有ロジックの有無」で決まり、4 層すべてを機械的に置くわけではない。共有される SCL Go binding は `backend/shared/spec` に残し（ADR-070）、context 固有の業務型は各 context の `domain/` が所有する（ADR-089）。`identitymanagement` / `tenancy` のように binding を超える固有ドメインロジックを持たない context は per-context `domain/` を持たない。逆に `saml` / `wsfederation` のようにプロトコル固有の解析・claim mapping を持つ context は `domain/` を、SSO/sign-in のオーケストレーション（SP/RP 解決・署名検証・割当ゲート・claim 発行）を持つ context は `usecases/` を持つ。ブラウザ federation の発行判断はすべて `usecases/` にあり、`adapters/http` は wire と HTTP 境界に閉じる。

`backend/shared/` は「複数 context が本当に共有する technical capability」だけに使う。context 固有の概念を便利だからという理由で `shared` に置くと、次の変更で読む範囲が広がる。

## HTTP Routing

HTTP route の集約点は `backend/shared/adapters/http/server/routes.go` である。ここで default tenant と `/realms/:tenant_id` の両方に tenant-scoped routes を登録し、control-plane tenant 管理だけを `/realms/default/admin/tenants` に分ける。

各 context の route は `backend/<context>/adapters/http/routes.go` に置く。エンドポイントの正確な一覧はそのファイルを読む。新しい HTTP API は、所有 context の `routes.go` に登録し、handler は同じ `adapters/http` 配下に置く。context 固有の repository とルート配線は `backend/<context>/module.go` に集約し、中央 router は Module を呼び出すだけにする（ADR-091）。

## Bootstrap And Adapters

`backend/cmd/idmagic/main.go` は `bootstrap.Run()` を呼ぶだけに保つ。起動時 DI は `backend/bootstrap` が所有する。また、`backend/cmd/idmagic-relay/main.go` は outbox → Kafka リレープロセスを起動するもので、`backend/relay.Run()` を呼ぶ。

`backend/bootstrap/deps.go` の `Dependencies` は HTTP 層へ渡す境界の集約で、memory / postgres_valkey / outbox / otel などの runtime 選択を吸収する。context 固有の repository は各 `Module` に束ね、中央 `Dependencies` と server `Deps` には Module を渡す。新しい port を追加したら、少なくとも次を確認する。

- 対象 context の `ports/`
- memory adapter
- postgres adapter と migration が必要か
- `bootstrap.Dependencies`
- `assembleMemory` / `assemblePostgresValkey`
- `support.Deps`
- 対象 HTTP handler または usecase の constructor

## Persistence

永続化 port と repository 実装は所有 context 側に置く。context 固有の memory / postgres adapter は `backend/<context>/adapters/persistence/{memory,postgres}` に同居し、`backend/shared/adapters/persistence/` は DB pool、row scanner、transaction helper、Valkey client などの技術的共通部品だけを持つ（ADR-090）。

PostgreSQL の構造を増やすときは、まず `deploy/schema/postgres.sql` の現在形 schema を更新する。構造差分は `psqldef` の dry-run で確認し、デプロイ前ジョブで適用する。既存データの backfill、値変換、削除前の退避など、構造差分だけでは表せない変更は、対象 WI の runbook または専用 SQL script として明示する。アプリ起動時の migration runner は持たない。memory adapter はテスト・ローカル demo の基準にもなるため、postgres だけを更新しない。

### データベース設計ポリシー (ADR-082 / ADR-084)

データベースのスキーマやテーブル構造を設計する際は、以下の方針を遵守する。

#### 1. 列型選定ルール
- **自由文字列 (上限なし)**: `TEXT` 型を使用する。`varchar` (制約なし) は使用しない。
- **上限のある文字列**: `TEXT` 型に `CHECK (char_length(col) <= N)` 制約を付与するか、`varchar(N)` に統一する。使い分けと具体的な最大文字数は `wi-128-string-length-limits-policy` に従う。
- **内部生成 ID**: `idmagic` が `spec.NewUUIDv4()` で内部生成する ID 列（`users.id`, `clients.client_id`, `groups.id`, `agents.id`, `audit_events.id`, `scim_tokens.id` 等）は、すべて `UUID` 型とする。Go 側では `string` 型のまま扱い、pgx 接続時の text codec 登録 (`RegisterUUIDAsText`) によって自動変換する。
- **外部決定 ID**: 外部（SP/RP メタデータ等）が値を決定する ID（`entity_id`, `wtrealm`, `scim_id`, `kid` 等）は `TEXT` 型を維持する。
- **時刻**: 一貫して `TIMESTAMPTZ` 型を使用する（マイクロ秒精度を真値とし、schema で丸めない）。
- **有限集合 (ステータス等)**: `TEXT` + `CHECK (col IN (...))` で値集合を表現し、PostgreSQL enum は原則使用しない。

#### 2. tenant_id 保持の 4 分類ルール
外部から parent 経由で辿れるという理由だけで機械的に `tenant_id` を全テーブルに追加しない。以下の分類に従って判断する。
- **tenant-owned aggregate**: `tenant_id` を PK または UNIQUE キーに含める（例: `users`, `groups`, `clients`）。
- **tenant-scoped natural key を参照する child**: 参照先が `(tenant_id, local_id)` の複合キーで識別される場合、child にも `tenant_id` を持たせ、composite FK (複合外部キー) でテナント不一致を DB 制約で防ぐ（例: `consents`, `refresh_tokens`）。
- **globally unique parent に従属する child**: 親のキーが UUID などでグローバル一意である場合は `tenant_id` を重複保持しない（例: `mfa_factors`, `password_history`）。
- **append-only / audit**: クエリ境界や監査隔離単位として必要な場合にのみ保持する（例: `audit_events`, `outbox`）。

## UI Boundary

React UI は Go API とは別成果物・別プロセスで、gateway によって同一オリジンへ統合される。詳細は `frontend/ARCHITECTURE.md` を読む。

UI の画面実装は `frontend/src/features/`、route は `frontend/src/routes/` が中心である。API の wire contract を変える場合は、Go handler/usecase と UI API client (`frontend/src/api*.ts`) の両方を確認する。

## Verification Entry Points

通常の Go 変更では次を使う。

```bash
GOCACHE=/tmp/idmagic-cache go test ./...
GOCACHE=/tmp/idmagic-cache go test -race ./...
```

UI 変更では `frontend/README.md` と `frontend/tests/e2e/README.md` の検証手順を読む。SCL や work item を変更した場合は、ルートの `tools/yaml-check` 系の検証も対象に含める。

## Documentation Policy

新しい説明を追加する前に、次を確認する。

- SCL に書くべき規範要件ではないか。
- ADR に書くべき再導出不能な判断理由ではないか。
- work item に書くべき一回限りの実施記録ではないか。
- コードや schema から機械的に読める一覧を手書き複製していないか。

この文書に追加してよいのは、AI が読む入口を狭める安定した地図だけである。機能ごとの詳細、最新のエンドポイント網羅表、全テスト一覧、全環境変数一覧は置かない。
