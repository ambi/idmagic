---
depends_on: [wi-49-agent-identity-first-class-principal, wi-50-token-exchange-delegation-actor-chain, wi-51-rich-authorization-requests-agent-scopes, wi-354-evaluate-cedar-authorization]
status: completed
authors: ["tn"]
risk: high
created_at: 2026-06-22
change_kind: feature
initial_context:
  specification:
    - spec/SPECIFICATION.md
    - spec/contexts/authorization/SPECIFICATION.md#REQ-AUTHORIZATION-001
    - spec/contexts/oauth2/SPECIFICATION.md
  typespec:
    - IdMagic.Contract.AuthorizationModel
    - IdMagic.Contract.RelationTuple
    - IdMagic.Contract.FgaCheckRequest
    - IdMagic.Contract.FgaCheckResult
  source:
    - backend/authorization
    - backend/shared/spec/policy.go
    - backend/shared/policy/authorization_local
    - backend/oauth2/ports/authorizer.go
    - backend/oauth2/token/usecases/role_policies.go
    - backend/cmd/internal/bootstrap
    - infra/schema/postgres.sql
  tests:
    - backend/authorization
    - backend/shared/policy/authorization_local
  stop_before_reading:
    - frontend
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { path: spec/contexts/authorization/SPECIFICATION.md, requirement: REQ-AUTHORIZATION-001 }
  - { path: spec/contexts/authorization/SPECIFICATION.md, requirement: REQ-AUTHORIZATION-004 }
  - { path: spec/contexts/authorization/SPECIFICATION.md, requirement: REQ-AUTHORIZATION-005 }
  - { path: spec/contexts/authorization/models.tsp, symbol: IdMagic.Contract.RelationTuple }
  - { path: spec/contexts/authorization/main.tsp, symbol: IdMagic.Authorization.Operations.CheckAccess }
---

# エージェントのデータアクセス向け関係ベース細粒度認可 (ReBAC / FGA)

## Motivation
エージェント、特に RAG (検索拡張生成) パイプラインは大量の文書・レコードへ横断アクセスするため、「代行しているユーザーが本来見られるものだけを取得する」細粒度の認可が不可欠になる。粗い RBAC では「ユーザー U が文書 D を読めるか」をリソース単位で判定できない。Google Zanzibar に始まる関係ベースアクセス制御 (ReBAC) と、その実装である OpenFGA (Auth0 / Okta の Fine-Grained Authorization) が、エージェントの RAG データアクセスを per-resource で絞る標準的手法になっている。

idmagic は AuthZEN スタイルの `Authorizer` ポートを持つが、判定はクライアント認可ルール中心で、リソース×主体の関係タプルに基づく判定を持たない。本 WI は AuthZEN の `{subject, action, resource, context}` インターフェースを拡張し、関係タプル (user/agent ⇄ resource) に基づく ReBAC 判定を追加する。これにより [[wi-50-token-exchange-delegation-actor-chain]] で代行する actor チェーンを考慮した「ユーザーとして、かつエージェント経由で」のアクセス判定が成立する。

## Scope
- **仕様**:
  - 新しい Bounded Context `Authorization` を追加する。`spec/contexts/authorization/{SPECIFICATION.md,models.tsp,main.tsp}` が規範的シナリオ、モデル、管理 API を所有する。
  - モデル: `AuthorizationModel` / `ResourceTypeDefinition` / `RelationDefinition` / `RelationRewrite` / `RelationTuple` / `FgaCheckRequest` / `FgaCheckResult` / `FgaListAccessibleResourcesResult`。
  - 操作: `PutAuthorizationModel` / `GetAuthorizationModel` / `WriteRelationTuples` / `ListRelationTuples` / `CheckAccess` / `ListAccessibleResources`。
  - 権限 `AdminAuthorizationModelManage` (AuthZEN action `admin:authorization_model_manage`) と、AuthZEN 判定 context への actor チェーンおよび関係事実 (relationship facts) の追加。
  - ルート `spec/SPECIFICATION.md` の Context Map、責務表、構造上の決定、データベース設計の同期。
- **Go**:
  - `backend/authorization/domain`: タプル・モデル・書き換え規則と、深さ制限つき・循環検出つきのグラフ評価器。
  - `backend/authorization/ports` と `db_memory` / `db_postgres`: テナント境界を持つタプル/モデルの永続化と整合トークン。
  - `backend/authorization/usecases`: 管理系ユースケースと、AuthZEN `Authorizer` へ関係事実を渡す `CheckAccess` / `ListAccessibleResources`。
  - `backend/shared/spec/policy.go`: action `resource:access` と関係事実・actor チェーンの規則、`admin:authorization_model_manage` の追加。
- **HTTP**:
  - `backend/authorization/handlers_http`: 認可モデルと関係タプルの管理 API、および内部・診断用途の判定エンドポイント。

## Out of Scope
- 外部 OpenFGA / Zanzibar サービスとの本番接続実装。差し替え可能なポート境界の提供までとする。
- アプリ側 RAG パイプラインそのものの実装。
- 大規模タプルのシャーディング・キャッシュ最適化 (まず正しさ優先)。列挙は上限つきの走査とし、逆引きインデックスの最適化は行わない。
- 管理 UI (React) の画面追加。API までとする。
- `ApiTokenScope` への `authorization-model:*` の追加。管理 API のスコープ配線は [[wi-320-agent-management-api-scope-wiring]] が扱う。

## Design

### 配置: 新しい Bounded Context
関係タプルと認可モデルはテナントが所有する集約であり、独自のライフサイクル、管理 API、監査イベントを持つ。よって `backend/shared/` の技術的能力ではなく `Authorization` Context として置く。`backend/shared/policy/authorization_*` に残るのは AuthZEN アダプターのままとし、ReBAC はその判定へ事実を供給する側になる。

### 関係の言語
Zanzibar の名前空間設定を最小限に絞った形を採る。

- 型と関係の名前は `^[a-z][a-z0-9_]*$` で最大 64 文字。
- `RelationDefinition` は書き換え規則の**和**で表す。規則は 3 種のみ。
  - `this`: 直接タプル。`direct_subject_types` で受け入れる主体型を宣言する。
  - `computed_userset`: 同じオブジェクトの別関係へ委ねる (`viewer` ← `editor`)。
  - `tuple_to_userset`: `tupleset_relation` で親オブジェクトへたどり、そこで `computed_relation` を判定する (`document#parent` → `folder#viewer`)。
- 主体は `type:id`、`type:id#relation` (subject set)、`type:*` (ワイルドカード) の 3 形。ワイルドカードは `this` 規則が明示的に許した型でのみ受け付ける。
- 交差 (intersection) と差集合 (exclusion) は初期対応に含めない。和だけなら評価器は単調で、規則の増加が既存の許可を取り消さない。取り消しを表現したくなった時点で改めて仕様を変える。

### 評価
- 深さ制限つきの深さ優先探索。既訪問の `(オブジェクト, 関係)` 対を記録して循環を検出する。
- 上限 (既定 8) を超えた、循環した、未知の型・関係を参照した、ストアが読めない、いずれの場合も**許可しない**。呼び出し側は「判定不能」を許可として扱ってはならない。
- 判定結果は許可・不許可に加えて、たどった関係名だけの経路 (`document#viewer` → `folder#viewer`) を返す。オブジェクト id は含めない。

### 代行 (actor chain) の合成
`CheckAccess` は代行チェーンを受け取り、次を **AND** で合成する。

1. 主体 (`sub`、通常は代行されるユーザー) が対象リソースに対して関係を持つ。
2. チェーン上の**すべての** actor が同じ関係を持つ。エージェントはユーザーの権限を超えられない。
3. チェーン上のすべての actor が有効である (`Agent.status` は `PrincipalStatusResolver` ポートで解決する。解決できなければ不許可)。
4. 要求した関係に対応するスコープが、提示されたトークンのスコープ集合に含まれる (RAR / スコープの下位集合判定)。
5. 主体とリソースのテナントが一致する。

合成そのものは既存の AuthZEN 評価器が行う。`Authorization` は 1〜3 を関係事実として `AuthZRequest.Context` に載せ、`Authorizer` ポートを呼ぶだけである。これによりリモート PDP へ差し替えても合成規則の所在が 2 か所に割れない。関係事実が欠けている要求は、規則 `relationship_facts_present` が不許可にする。

### 整合 (consistency)
テナントごとに単調増加する書き込み版を持ち、タプル書き込みとモデル登録が同じトランザクションで進める。書き込みは不透明な整合トークン (テナント id と版を束縛した base64url) を返し、`CheckAccess` は `minimum_consistency` としてそれを提示できる。ストアの版がトークンより古い場合、または他テナントのトークンだった場合は fail-closed で拒否する。単一 PostgreSQL では読み取りは元から強整合なので、このトークンは「書いた直後の管理操作が確かに自分の書き込みを見た」ことの検証であり、キャッシュ層を入れた時の契約でもある。

### 監査
`RelationTupleWritten` / `RelationTupleDeleted` / `AuthorizationModelPublished` / `FgaCheckEvaluated` を発行する。`FgaCheckEvaluated` はリソース型、関係、許可・不許可、モデル版、関係名だけの経路要約、拒否理由コードを持ち、リソース id は SHA-256 の先頭 16 桁のダイジェストにする。タプルの全量、主体 id、リソース名は監査へ複製しない。

## Plan
1. 仕様を先に変える。`Authorization` Context の TypeSpec と `SPECIFICATION.md`、ルート `SPECIFICATION.md` の Design 同期、`just check-spec`。
2. ドメイン (モデル検証、タプル解析、評価器) を実装し、RED を確認してから緑にする。
3. `ports` とメモリアダプター、続けて PostgreSQL アダプター (スキーマ、`sqlc`、整合版) を同じ契約テストで揃える。
4. AuthZEN 規則 (`resource:access`、`admin:authorization_model_manage`) を `backend/shared/spec` へ追加し、ユースケースから合成する。
5. 管理 API とルーティング、`bootstrap` の配線。
6. 越境・障害・打ち切りの検証を足し、`just verify`。

## Tasks
- [x] T001 [Spec] `Authorization` Context の TypeSpec とシナリオを追加し、ルート `SPECIFICATION.md` の Context Map・責務表・構造上の決定・データベース設計を同期する。`just check-spec` を通す。
- [x] T002 [Domain] `AuthorizationModel` の検証、タプルの解析と検証、深さ制限つきグラフ評価器を実装する。RED を確認する。テスト: `backend/authorization/domain` の `TestCheckTraversesGroupAndParent` / `TestCheckDeniesOnCycleAndDepth` / `TestValidateRejectsUnknownRelation` (REQ-AUTHORIZATION-001, REQ-AUTHORIZATION-003, REQ-AUTHORIZATION-005)。
- [x] T003 [Persistence] `ports` とメモリ・PostgreSQL アダプター、テナント境界つきスキーマ、索引、トランザクション書き込みと整合版を実装する。テスト: `backend/authorization/db_memory` と `backend/authorization/db_postgres` の `TestRelationTupleRepositoryContract` (REQ-AUTHORIZATION-002, REQ-AUTHORIZATION-006, REQ-AUTHORIZATION-008)。
- [x] T004 [Policy] AuthZEN に `resource:access` と関係事実・actor チェーンの規則を追加し、`CheckAccess` ユースケースから fail-closed に合成する。テスト: `backend/shared/spec` の `TestEvaluateResourceAccess*` と `backend/authorization/usecases` の `TestCheckAccessRequiresSubjectAndActorChain` (REQ-AUTHORIZATION-004, REQ-AUTHORIZATION-005)。
- [x] T005 [Management] 管理 API (モデル・タプル) と判定 API、`AdminAuthorizationModelManage` 権限、監査イベントを追加し、`bootstrap` に配線する。テスト: `backend/authorization/handlers_http` の `TestAuthorizationAdminRoutes` (REQ-AUTHORIZATION-001, REQ-AUTHORIZATION-009)。
- [x] T006 [Verify] 入れ子グループ・所有者、削除の波及、越境タプルの注入、ストア障害、列挙の打ち切り、判定経路を検証し、`just verify` を通す (REQ-AUTHORIZATION-006, REQ-AUTHORIZATION-007, REQ-AUTHORIZATION-008)。

## Verification
- `just test-go`
  - reason: タプルに基づく許可 / 拒否、グラフ探索 (継承・グループ)、actor 考慮、テナント越境拒否の境界。
- `just lint-go`
- `just check-spec`
- `just verify`
- 手動: ユーザーに文書 A のみ許可するタプルを書く → エージェント代行で A は許可・B は拒否されることを確認する。

## Risk Notes
ReBAC は判定ロジックの中枢で、グラフ探索の誤りや既定許可は情報漏洩に直結する。既定拒否 (fail-closed) を徹底し、actor チェーンを判定 context に明示的に載せる。既存 `Authorizer` アダプター境界を踏襲し、ローカル実装と外部 PDP を同一契約で検証する。列挙 API は上限つきの走査なので、打ち切りを黙って成功として返さない。

## Completion
- **Completed At**: 2026-08-15
- **Summary**:
  新しい Bounded Context `Authorization` を追加し、リソース 1 件ごとの関係ベース認可 (ReBAC) を通した。規範的シナリオ REQ-AUTHORIZATION-001〜009 と、認可モデル・関係タプル・判定・列挙の TypeSpec 契約 (`AuthorizationModel` / `RelationTuple` / `FgaCheckRequest` / `FgaCheckResult` ほか計 22 モデルと 6 操作) が新設された。判定は Zanzibar 風の書き換え規則 3 種 (`direct` / `computed_userset` / `tuple_to_userset`) の和で表し、深さ制限つきの探索で評価する。深さ超過・循環・未知の関係・ストア障害・事実の欠落はいずれも許可へ退避しない。
  合成規則は新しい経路を作らず、既存 AuthZEN の `Authorizer` ポートへ寄せた。`resource:access` アクションと 4 つの新規規則 (`relationship_facts_present` / `relationship_permits_subject` / `relationship_permits_actor_chain` / `actor_chain_principals_active`) を `backend/shared/spec` に追加し、Authorization は関係の事実と代行チェーンを判定 context に載せるだけになった。これによりエージェントは代行するユーザーの権限を超えられない。
  永続化はテナント境界を主キーに持つ `authorization_relation_tuples`、追記のみの `authorization_models`、テナントごとの `authorization_write_versions` の 3 テーブルで、メモリと PostgreSQL が同じ契約テストを共有する。書き込みが返す整合トークンはテナントを束縛しており、他テナントのトークンや先行する版の提示は fail-closed で拒否する。監査は `AuthorizationModelPublished` / `RelationTupleWritten` / `RelationTupleDeleted` / `FgaCheckEvaluated` / `FgaResourcesEnumerated` を発行し、リソース識別子はテナントと型を混ぜたダイジェストに落とす。列挙は 1 件ごとの判定を監査へ展開せず、走査全体を 1 件にまとめる。
  権限 `AdminAuthorizationModelManage` (`admin:authorization_model_manage`) を追加し、`/api/admin/v1/authorization/*` の 6 エンドポイントを管理者に限定した。ルート `SPECIFICATION.md` の Context Map、責務表、構造上の決定、データベース設計も同期した。
- **Verification Results**:
  - `just verify` - passed
  - `just test-go` - passed (domain / db_memory / db_postgres / usecases / handlers_http)
  - `just lint-go` - passed
  - `just check-spec` - passed
  - `just check-schema` - passed (psqldef converges)
  - 手動: 文書 A のみを alice とその代行エージェントへ許可し、HTTP 経由で A は許可・B は拒否になることを確認した。この手順は `backend/authorization/handlers_http` の `TestAuthorizationAdminRoutes/an_agent_delegate_reaches_only_the_document_the_user_was_granted` として自動化した。
