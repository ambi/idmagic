---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-13
depends_on: [wi-49-agent-identity-first-class-principal, wi-50-token-exchange-delegation-actor-chain]
change_kind: feature
initial_context:
  specification:
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-046
    - spec/contexts/tenancy/SPECIFICATION.md#REQ-TENANCY-019
  typespec:
    - IdMagic.Contract.Tenant
    - IdMagic.Contract.PasswordPolicyOverride
    - IdMagic.Contract.AdminSettingsResponse
    - IdMagic.Contract.TenantUpdateRequest
    - IdMagic.Contract.TenantSummaryResponse
    - IdMagic.Contract.DelegationMode
    - IdMagic.Contract.IntrospectionResponse
    - IdMagic.Contract.TokenExchanged
  source:
    - backend/oauth2/domain/delegation_mode.go
    - backend/oauth2/token/usecases/exchange_token.go
    - backend/oauth2/token/usecases/introspect_token.go
    - backend/oauth2/domain/events.go
    - backend/oauth2/ports/delegation_policy.go
    - backend/oauth2/ports/token_introspector.go
    - backend/oauth2/policy_tenancy/delegation_policy.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/tenancy/domain/tenancy.go
    - backend/tenancy/usecases/manage_tenants.go
    - backend/tenancy/handlers_http/admin_settings_handler.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/tenancy/db_postgres/tenants.go
    - backend/tenancy/db_postgres/tenants.sql
    - frontend/src/features/admin-settings/AdminSettingsPage.tsx
    - frontend/src/features/admin-settings/AdminSettingsPage.i18n.ts
    - frontend/src/features/admin-settings/DelegationPolicyTab.tsx
    - frontend/src/api/admin.ts
    - frontend/src/types.ts
    - infra/schema/postgres.sql
  tests:
    - backend/oauth2/domain/delegation_mode_test.go
    - backend/oauth2/token/usecases/exchange_token_delegation_policy_test.go
    - backend/oauth2/token/usecases/delegation_mode_agreement_test.go
    - backend/oauth2/handlers_http/token_exchange_handler_test.go
    - backend/tenancy/usecases/manage_delegation_depth_test.go
    - backend/tenancy/db_postgres/tenants_test.go
    - backend/tenancy/handlers_http/admin_settings_handler_test.go
    - frontend/src/features/admin-settings/DelegationPolicyTab.test.tsx
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
    - backend/provisioning
affected_spec:
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.Tenant }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.AdminSettingsResponse }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantUpdateRequest }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantSummaryResponse }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.DelegationMode }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.IntrospectionResponse }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.TokenExchanged }
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-048 }
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-049 }
  - { path: spec/contexts/tenancy/SPECIFICATION.md, requirement: REQ-TENANCY-021 }
---

# 委譲深さの上限をテナントポリシーにし、トークンが自律実行か代理実行かを判別可能にする

## Motivation

2 つの問題が同じ場所にある。

**(1) 仕様と実装が乖離している。** `spec/contexts/oauth2/SPECIFICATION.md` は Token Exchange の
制約として "a configurable maximum delegation depth" と書いているが、実装は
`backend/oauth2/token/usecases/exchange_token.go` の `const MaxDelegationDepth = 3` で
ハードコードされている。テナントごとにリスク許容度は異なる — 社内の閉じた
エージェント連携と、外部 SaaS を跨ぐ連携では妥当な委譲段数が違う — のに、その差を表現できない。

**(2) 委譲モードが判別できない。** OpenID Foundation の "Identity Management for Agentic AI"
(2025-10) は、エージェント ID の 3 大ギャップの 1 つとして「エージェントが自律実行と
ユーザー代理を行き来するのに、**今どちらのモードで動いているかを追跡できない**」ことを挙げている。
idmagic は `principal_type=agent` / `agent_id` クレームと `act` チェーンを既に持つため
原理的には導出可能だが、introspection 応答にも監査イベントにも**モードとして明示されていない**。
リソースサーバーと監査担当者が同じ導出規則を各自で再実装することになり、解釈がずれる。

同じ OIDF 文書のギャップ (3)「再帰的委譲が原理的な上限を持たない」に対しては、idmagic は
delegation-only + 深さ上限 + `may_act` 強制で既に先行している。上限をポリシーとして
表現できるようにすることは、その先行を運用可能な形にする差分である。

この項目は [[wi-369-agent-capability-survey-2026-08]] の棚卸しで P1 と判断した。

## Scope

- `spec/contexts/tenancy/models.tsp` の `Tenant` に委譲深さ上限の override を追加する。
- `spec/contexts/oauth2/SPECIFICATION.md` に、テナント上限を超える委譲が拒否される
  normative scenario (REQ-OAUTH2-048) と、委譲モードを一貫して返す scenario
  (REQ-OAUTH2-049) を追加する。
- `exchange_token.go` の `const MaxDelegationDepth` をテナント設定からの解決に置き換える。
  判定箇所 (`act` 入れ子深さの検査) と、既存の `evaluateTokenExchangePolicy` が AuthZEN
  リクエストに渡している `DelegationDepth` の経路を再利用する。
- 委譲モード (自律実行 / ユーザー代理) を introspection 応答と Token Exchange の監査イベントで
  明示する。
- 管理コンソールのテナント設定に委譲深さ上限を追加する。

## Out of Scope

- 委譲チェーンの内容に対するポリシー評価 (どの主体からどの主体への委譲を許すか)。
  それは [[wi-53-rebac-fine-grained-authorization]] と
  [[wi-59-agent-governance-guardrails-audit-inventory]] の領域で、本 work item は深さと
  モードの表現だけを扱う。
- impersonation モードの Token Exchange。`act` チェーンが消えて監査が壊れるため、
  意図的に未実装のまま維持する ([[wi-50-token-exchange-delegation-actor-chain]] の判断)。
- Transaction Tokens によるサービス間の文脈伝播。draft が未成熟なため
  [[wi-369-agent-capability-survey-2026-08]] で見送りと記録した。

## Design

- **既存の `PasswordPolicyOverride` の形に倣う**。`Tenant` は既に
  「テナント値が global default より**厳しい方向にのみ**働く」上書きの前例を持っている
  (`spec/contexts/tenancy/models.tsp`)。委譲深さも同じ扱いにする: テナントは上限を
  **下げる**ことはできるが、system ceiling を超えて上げることはできない。これにより
  「テナント設定でセキュリティを緩められる」経路を作らない。
- **既定値は現行と同じ 3 を維持する**。省略時は global default を継承し、既存の挙動を変えない。
- **モードは新しい状態を持たず、既存のクレームから導出する**。`act` の有無と `sub` の
  principal 種別で決まる — `sub` が User でありエージェントが `act` にいれば「ユーザー代理」、
  `sub` 自身がエージェントなら「自律実行」。**導出規則を 1 箇所に閉じ**、introspection と
  監査イベントが同じ関数を通ることを保証する。新しい永続状態を足すと、クレームと
  食い違う第二の真実ができる。
- 拒否は fail-closed。テナント設定の解決に失敗した場合は global default ではなく拒否する
  ([[wi-50-token-exchange-delegation-actor-chain]] の Token Exchange 全体の方針に揃える)。

## Plan

- 先に spec (Tenant の override + REQ-OAUTH2-048 / REQ-OAUTH2-049) を確定させ、そのあと実装する。
- テナント設定の解決経路は `exchange_token.go` の中で既にテナントを解決しているため、
  新しい依存を足さずに済むかを最初に確認する。足りない場合もポートを 1 本増やすに留める。
- モード導出は domain の純関数として置き、introspection と監査イベントの両方が
  それを呼ぶ形にする。表示側で個別に導出する形は採らない。
- 未決定: モードの表現を introspection のトップレベルフィールドにするか、既存の
  `principal_type` の値域を広げるかは実装時に決める。既存フィールドの意味を変えると
  リソースサーバー側の解釈が壊れるため、新規フィールドを既定とする。

## Tasks

- [x] T001 [Spec] `Tenant` に委譲深さ上限の override (厳しい方向のみ・system ceiling 付き) を追加し、`REQ-OAUTH2-048`、`REQ-OAUTH2-049`、`REQ-TENANCY-021` を追加した。
- [x] T002 [Domain] 委譲モードの導出を純関数として実装した。RED → GREEN: `TestDeriveDelegationMode` (`REQ-OAUTH2-049`) が自律実行、利用者代理、多段委譲を固定する。
- [x] T003 [UseCases] 固定の委譲深さをテナントポリシー解決に置き換えた。RED → GREEN: `TestExchangeTokenHonoursTenantDelegationDepth` (`REQ-OAUTH2-048`) が上限超過、上限内、解決失敗時の fail-closed を固定する。
- [x] T004 [Adapters] introspection 応答と `TokenExchanged` 監査イベントにモードを含めた。RED → GREEN: `TestDelegationModeAgreesBetweenAuditAndIntrospection` と `TestTokenExchangeIssuesDelegatedToken` (`REQ-OAUTH2-049`) が同じ導出結果と HTTP 応答を固定する。
- [x] T005 [Persistence] memory / PostgreSQL で委譲深さ上限を保存・読出しできるようにした。RED → GREEN: `TestUpdateMaxDelegationDepthOnlyTightens` と `TestTenantRepositoryPersistsMaxDelegationDepth` (`REQ-TENANCY-021`) が往復、解除、ceiling 超過拒否を固定する。
- [x] T006 [UI] テナント設定画面に委譲ポリシータブを追加した。RED → GREEN: `DelegationPolicyTab` の unit test (`REQ-TENANCY-021`) が厳しい上書き、解除、上限超過の送信前拒否を固定する。
- [x] T007 [Verify] Verification を緑にし、`just spec-render` で派生仕様を再生成した。

## Verification

- `just check` / `just check-spec` / `just check-work-items`
- `just verify-go` / `just test-go-race`
- `just verify-ui`
- 手動: `just dev` で (1) 既定 (未設定) のテナントで従来どおり深さ 3 まで委譲できること、
  (2) 上限を 1 に下げたテナントで 2 段目が拒否されること、(3) ceiling を超える値が
  設定 API で拒否されること、(4) introspection がユーザー代理と自律実行を区別して返すことを確認する。

## Risk Notes

リスクは medium。委譲深さは**認可の境界**であり、テナント設定で緩められる経路を作ると
セキュリティ設定が下方向に破られる。`PasswordPolicyOverride` と同じ「厳しい方向にのみ」規則と
system ceiling をテストで固定する。

既定値を変えると既存の委譲チェーンが壊れる。省略時は現行と同じ 3 を継承することを回帰テストで固定する。

モード導出を 2 箇所に書くと、introspection と監査ログが食い違う。これは調査時に最も
質の悪い形の不整合になるため、導出を純関数 1 箇所に閉じ、両経路がそれを通ることをテストで固定する。

## Completion
- **Completed At**: 2026-08-15
- **Summary**:
  `just spec-diff` が示す意味差分は `REQ-OAUTH2-048`、`REQ-OAUTH2-049`、`REQ-TENANCY-021` と `DelegationMode` 宣言の追加である。Token Exchange の委譲深さはテナントがシステム既定 3 以下へ厳しくでき、解決失敗では拒否される。自律実行、利用者代理、直接実行は 1 つの純関数から introspection と `TokenExchanged` 監査イベントへ出力される。設定は memory / PostgreSQL / テナント API を往復し、管理 UI では実効値と継承状態を表示して上書きと解除を行える。
- **Verification Results**:
  - `just check-spec` - passed
  - `just check-api-compat` - passed; no breaking changes against the release baseline
  - `just spec-render` - passed; OpenAPI and 813 specification pages generated
  - `just check-schema` - passed; schema convergence confirmed
  - `just verify-go` - passed; lint 0 issues and all race-enabled Go tests passed
  - `just verify-ui` - passed; format, lint, 572 unit tests, and production build passed
  - `just verify` - passed; all 10 repository checks passed
  - 手動 `just dev` の 4 ケースは対話実行せず、同じ条件を上記の domain / use case / HTTP / UI 自動テストで固定した
