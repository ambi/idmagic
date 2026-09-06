---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-30
change_kind: bugfix
priority: p2
depends_on: []
evidence_policy: risk-based-v3
documentation_impact:
  level: removal_notice
  reason: 管理 API が既に返している 403 Problem Details の error code を公開契約へ追加し、OAuth 本文用の旧 InsufficientScopeError 型名を用途固有名へ置き換える。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-454-declared-403-body-vs-guard-error-codes.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-454-declared-403-body-vs-guard-error-codes.md }
initial_context:
  specification:
    - docs/scenarios.feature.md#REQ-PLATFORM-004
    - docs/contexts/api-tokens/scenarios.feature.md#REQ-APITOKENS-004
    - docs/api-rules.md#http-error-responses
    - docs/api-rules.md#declared-status-codes
  typespec:
    - IdMagic.IdManagement.Operations.GetAdminUser
    - IdMagic.Contract.AccessDeniedError
  source:
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/admin_scope.go
    - backend/shared/http/support_http/csrf.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - tools/check/src/status-drift.ts
    - tools/check/src/security-controls.ts
  tests:
    - backend/shared/http/support_http/admin_scope_test.go
    - backend/shared/http/support_http/csrf_test.go
    - backend/shared/http/server_http/tenant_quota_csrf_test.go
    - tools/check/src/status-drift.test.ts
    - tools/check/src/security-controls.test.ts
  stop_before_reading:
    - frontend
    - backend/sourcing
primary_use_cases:
  - id: api-token-insufficient-scope-body
    requirement: REQ-APITOKENS-004
    observable_result: 管理 API の 403 契約が API アクセストークン guard の insufficient_scope Problem Details を含む。
    unit_test: { path: tools/check/src/status-drift.test.ts, name: reports a guard error code omitted from the declared 403 body, task: test-tools }
    e2e_test: { path: tools/check/src/status-drift.test.ts, name: matches the repository 403 bodies to guard error codes, task: test-tools }
    unit_fault_model: 403 schema から InsufficientScopeError を除いても本文ドリフト検査が合格する。
    e2e_fault_model: 管理 API の TypeSpec が access_denied だけを宣言したまま実ソースの WriteAccessTokenError と照合されない。
  - id: browser-origin-csrf-body
    requirement: REQ-PLATFORM-004
    observable_result: 状態変更 operation の 403 契約が invalid_origin と csrf_failed の Problem Details を含み、拒否後の状態変更は起きない。
    unit_test: { path: tools/check/src/status-drift.test.ts, name: follows browser guard error codes into the declared 403 body, task: test-tools }
    e2e_test: { path: backend/shared/http/server_http/tenant_quota_csrf_test.go, name: TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF, task: test-go-race }
    unit_fault_model: VerifyBrowserRequest が書く error code を responder の伝播対象から落とす。
    e2e_fault_model: VerifyBrowserRequest が 403 を書いた後も保存処理へ進む。
affected_spec:
  - { path: docs/scenarios.feature.md, requirement: REQ-PLATFORM-004 }
  - { path: docs/contexts/api-tokens/scenarios.feature.md, requirement: REQ-APITOKENS-004 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminUser }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.AccessDeniedError }
---

# 403 が名乗る本体を、guard が実際に返すエラーコードに合わせる

## Motivation

`wi-386` はステータスコード集合を閉じたが、403 の**本体**が実装と食い違っていることを、その過程で数え上げた。403 を宣言する 272 operation のうち 270 が `AccessDeniedError` (`urn:idmagic:error:access_denied`) だけを名乗る。しかし 403 を書く guard は 3 つあり、返すコードは 3 通りである。

- `support_http.VerifyBrowserRequest` は `invalid_origin` と `csrf_failed` を返す。ブラウザーから呼ばれる 130 の handler がこれを通る。
- `support_http.WriteAccessTokenError` は `insufficient_scope` を返す。API アクセストークンで到達しうる管理 API のほぼ全部がこれを通るが、`InsufficientScopeError` を宣言しているのは 2 operation だけである。
- `Authenticator.WriteAdminAccessError` は `access_denied` を返す。宣言が合っているのはこれだけである。

`docs/api-rules.md` は「個々のエラーは `model <Name>Error is ProblemDetails;` と書き、どの `type` URN 接尾辞に対応するかを `@doc` で名指しする」と定めている。同じ 403 に 3 つのコードが乗っているのに 1 つしか宣言していないので、`type` を鍵に翻訳する UI は、実際に来る 2 つを辞書で引けない。

`wi-386` はこれを Out of Scope に置いた。本体の形は `wi-382` と `wi-385` の担当で、132 operation のうち 2 つだけを直せば残りとの不一致を新しく作ることになるためである。

## Scope

- `invalid_origin` / `csrf_failed` / `insufficient_scope` の本体モデルを宣言する。
- 各 operation の `<Op>Error403Body` を、その operation の手前に立つ guard が実際に返しうるコードの union にする。
- `check-security-controls` の R4 は 403 の本体型で scenario と突き合わせている。新しい型を足すと R4 が「その拒否を宣言する scenario が無い」と言うので、CSRF と origin の拒否を規範的シナリオとして書き、その id を検査から名指しする。`docs/development/specification-first-workflow.md` の「拒否のテスト」に従い、拒否が何を残さなかったかまで見る。
- 本体の一致を機械で見る手段を検討する。`wi-385` の `check-contract-drift` は成功応答の本体しか見ておらず、エラー本体は誰も突き合わせていない。

## Out of Scope

- ステータスコード集合。`wi-386` が閉じた。

## Design

Problem Details の `InsufficientScopeError`、`InvalidOriginError`、`CsrfFailedError` を共有契約に置き、OIDC の OAuth 本文は `OAuthInsufficientScopeError` として分離する。`status-drift` の responder は `(status, error code)` の組を収集し、guard の呼び出し鎖を通じて operation へ伝播する。OpenAPI の 403 schema を `$ref` / `anyOf` 越しに解決して `@doc` の URN 接尾辞と突き合わせる。入力は生成 OpenAPI と Go ソース、出力は差分 finding であり、時刻、乱数、永続化、通知は関与しない。

## Plan

1. 共有エラーモデルと REQ-PLATFORM-004 を仕様へ追加する。
2. 403 本文ドリフトの Unit RED を置き、検査器を実装して現在の不一致一覧を得る。
3. guard が実際に返すコードだけを各 operation の 403 union へ反映し、実リポジトリ照合を GREEN にする。
4. R4 が共有シナリオの正確な ID を要求するようにし、拒否の効果テストと全体検証を通す。

## Tasks

- [x] T001 [Spec] 3 つの Problem Details モデル、OAuth 本文との分離、REQ-PLATFORM-004 を追加する。
- [x] T002 [Unit] 403 本文から guard code が欠落する fixture の RED を確認する (REQ-APITOKENS-004 / REQ-PLATFORM-004)。
- [x] T003 [Tooling] responder と OpenAPI schema の本文コード照合を実装する。
- [x] T004 [Spec] 実際の guard code を各 operation の 403 union へ反映する。
- [x] T005 [Acceptance] 実リポジトリの本文ドリフト検査と拒否の効果テストを通す。
- [x] T006 [Verify] 代表的な誤契約と guard 素通りを注入して検出し、`mise run verify` を通す。

## Verification

- `mise run check-spec`
- `mise run check-security-controls`
- `mise run verify`

## Risk Notes

`<Op>Error403Body` の union にメンバーを足すのは、生成クライアントにとっては受け取りうる型が増えることであり、既存の分岐は壊れない。ただし R4 が要求する scenario を書かずに型だけ足すと `mise run check` が落ちるので、シナリオと検査を同じ変更に含める。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `REQ-PLATFORM-004`、`InsufficientScopeError`、`InvalidOriginError`、`CsrfFailedError` などの追加と `REQ-APITOKENS-004` の変更を報告した。各 operation の 403 本文を、到達可能な管理権限、API トークンスコープ、origin、CSRF、ステップアップ、リンク解除の拒否コードへ合わせた。`check-status-drift` は Go の guard 呼び出し鎖から `(status, code)` を伝播し、OpenAPI の `$ref` と union を解決して不足する 403 型を検出する。
- **Primary Use Case Evidence**:
  - id: api-token-insufficient-scope-body
    unit_red: >-
      `reports a guard error code omitted from the declared 403 body` は、fixture の 403 本文に `insufficient_scope` が無いことを実装前に検出できず失敗した。
    e2e_red: >-
      `matches the repository 403 bodies to guard error codes` は、実リポジトリの管理 API 契約が guard の `insufficient_scope` を宣言していないため B1 finding を報告した。
    unit_fault_injection: >-
      fixture の宣言から `InsufficientScopeError` を除くと、単体検査が不足コードを含む B1 finding を返した。
    e2e_fault_injection: >-
      `GetAdminUserError403Body` から `InsufficientScopeError` を除くと、`mise run check-status-drift` が同 operation の B1 finding で失敗した。
  - id: browser-origin-csrf-body
    unit_red: >-
      `follows browser guard error codes into the declared 403 body` は、実装前に `VerifyBrowserRequest` の `invalid_origin` と `csrf_failed` が operation へ伝播せず失敗した。
    e2e_red: >-
      `TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF` の表駆動検査により、invalid origin と CSRF 欠落の正確な拒否本文、および保存・利用量更新が起きないことを固定した。
    unit_fault_injection: >-
      browser guard の code 伝播を欠落させると、単体検査が期待する `invalid_origin` と `csrf_failed` の B1 finding を構成できず失敗した。
    e2e_fault_injection: >-
      invalid-origin 分岐を成功扱いにすると、割当量テストが `SetQuota calls=1` と保存値の 100 から 20000 への変更を検出した。
- **Change-Resistance Results**:
  実契約から `InsufficientScopeError` を除く故障と、invalid-origin guard を素通りさせる故障を注入した。前者は契約ドリフト検査が operation 名と不足コードを示して検出し、後者は拒否後に起きてはならない保存効果を受け入れテストが検出した。復元後はいずれも成功した。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-security-controls` - passed (61 refusals, 82 error types)
  - `mise run check-status-drift` - passed (0 findings)
  - `mise run check-api-compat` - passed
  - `mise run verify` - passed
