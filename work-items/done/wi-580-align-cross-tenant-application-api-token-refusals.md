---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-14
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: Application API の別テナントトークン拒否を 403 access_denied から 401 invalid_token へ訂正し、クライアントが資格情報の再取得を判断する公開契約が変わる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-580.md }
initial_context:
  specification:
    - docs/contexts/application/scenarios.feature.md#REQ-APPLICATION-003
    - docs/contexts/application/scenarios.feature.md#REQ-APPLICATION-004
    - docs/contexts/api-tokens/standards.md#RFC9700-API-TOKEN-AUDIENCE
    - docs/design/security/authorization.md
    - docs/design/application/api-rules.md
  typespec:
    - IdMagic.Application.Operations.ListAdminApplications
    - IdMagic.Application.Operations.ListMyApplications
    - IdMagic.Application.Operations.GetMyApplicationOrder
    - IdMagic.Application.Operations.ReorderMyApplications
    - IdMagic.Contract.AuthenticationRequiredResponse
    - IdMagic.Contract.InvalidAccessTokenError
  source:
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/error_handler.go
    - backend/application/handlers_http/account_application_handler.go
    - backend/application/module.go
    - backend/shared/http/testing_stack/stack.go
  tests:
    - backend/shared/http/server_http/api_token_standards_test.go
    - backend/shared/http/support_http/auth_admin_test.go
    - backend/shared/http/support_http/error_handler_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/application/db_postgres
affected_spec:
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-003 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-004 }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.ListAdminApplications }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.ListMyApplications }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.GetMyApplicationOrder }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.ReorderMyApplications }
  - { path: spec/contexts/system/models.tsp, symbol: IdMagic.Contract.AuthenticationRequiredResponse }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.InvalidAccessTokenError }
primary_use_cases:
  - id: foreign-tenant-account-token-is-invalid
    requirement: REQ-APPLICATION-003
    observable_result: 発行元では有効な API アクセストークンを別テナントの Application account API へ提示すると 401 invalid_token になり、保存済みの並び順は変わらない。
    unit_test: { path: backend/shared/http/support_http/error_handler_test.go, name: TestErrorHandlerWritesInvalidTokenAsUnauthorized, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/application_api_token_tenant_test.go, name: TestForeignTenantApiTokenCannotReorderApplications, task: test-go-race }
    unit_fault_model: 共通エラーハンドラーが InvalidTokenError を未処理として 500 にするか、authentication_required へ潰す。
    e2e_fault_model: account 認証境界が audience 検証エラーを未認証として潰すか、拒否後も処理を続けて並び順を保存する。
  - id: foreign-tenant-admin-token-is-invalid
    requirement: REQ-APPLICATION-004
    observable_result: 発行元では有効な API アクセストークンを別テナントの Application admin API へ提示すると 401 invalid_token になり、Application は作成されない。
    unit_test: { path: backend/shared/http/support_http/auth_admin_test.go, name: TestWriteAdminAccessErrorPreservesInvalidTokenClassification, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/application_api_token_tenant_test.go, name: TestForeignTenantApiTokenCannotCreateApplication, task: test-go-race }
    unit_fault_model: admin のエラー写像が InvalidTokenError を認可失敗として 403 access_denied に変える。
    e2e_fault_model: admin 認証境界が audience 検証エラーを未認証として潰すか、拒否後も処理を続けて Application を作成する。
---

# 別テナントの Application API へ提示した API トークンの拒否を宣言と一致させる

## Motivation

`EX-APPLICATION-003-02` と `EX-APPLICATION-004-03` は、API アクセストークンのテナントが操作対象と一致しない場合に `AccessDeniedError` で拒否すると宣言している。

現在の共通認証境界は、発行元と異なるレルムへ提示された API アクセストークンを Application の認可判断より前に `InvalidTokenError` として扱い、401 `invalid_token` を返す。

拒否自体は効いているが、認証失敗と認可失敗のどちらとして公開するかがシナリオと実装で一致していない。

## Scope

- `EX-APPLICATION-003-02` と `EX-APPLICATION-004-03` について、別テナントの Application account API と admin API へ API アクセストークンを提示したときの公開エラーを一致させる。
- account API の「`user_id` が操作対象と一致しない」が、本人専用で対象 ID を入力に持たない現在の operation で何を指すかを確定する。
- 製品と同じ組み立てを使う受け入れテストで、拒否応答と対象テナントの Application、割り当て、並び順が変わらないことを確認する。
- 公開エラーを変更する場合は TypeSpec と正準シナリオを実装より先に更新する。

## Out of Scope

- 同一テナント内の不足スコープを `insufficient_scope` で拒否する振る舞い。
- Application 以外の account API と admin API のエラー分類を一括して変更すること。
- 別テナントのトークンから Application 操作を許可すること。
- API アクセストークンの署名方式、有効期限、送信者制約の変更。

## Design

現在の経路では、トークンの audience と要求先 realm が一致しない時点で主体を認証できないため、Application handler は呼ばれない。
この失敗は RFC 6750 の `invalid_token` であり、401 に分類する。
無効な audience の資格情報から主体を確定して 403 を返す案は、発行元 realm に閉じたトークンを別のリソースサーバーで認証することになるため採用しない。

404 は、要求先テナントを解決できない場合と、認証および認可を終えた主体が存在しない資源または別テナントの資源 ID を指定した場合に使う。
WI-580 の操作は、既知の要求先 realm へ不適合な資格情報を提示する段階で失敗するため、資源の不在を表す 404 にはしない。

account API の操作対象は、入力された `user_id` ではなく認証コンテキストの主体自身から決まる。
したがって、本人専用 operation に外部入力の `user_id` を追加せず、表現不能な `user_id` 不一致を具体例から除く。

## Plan

1. Application の対象 operation、API トークンの audience 検証、認証と認可のエラー写像を同じ要求について照合する。
2. 公開エラーの正準を決め、必要なら TypeSpec とシナリオを先に更新する。
3. account API と admin API の受け入れ RED、およびエラー分類を所有する単体 RED を確認する。
4. 選んだ境界で最小の変更を実装し、拒否後の Application 状態を読み直す。
5. 契約検査と標準検証を通す。

## Tasks

- [x] T001 [Design] 401 と 403 のどちらを正準とするかを、TypeSpec、threat model、自己サービス境界から決める。
- [x] T002 [Spec] 決定に応じて TypeSpec または正準シナリオを更新し、`mise run check-spec` を通す。
- [x] T003 [Acceptance] `testing_stack` を使い、別テナントの account Application API への提示が宣言したエラーになり、並び順を変えないことを確かめる。
- [x] T004 [Acceptance] `testing_stack` を使い、別テナントの admin Application API への提示が宣言したエラーになり、Application を変えないことを確かめる。
- [x] T005 [Unit] 認証または認可の選択した境界でエラー分類の単体 RED を確認し、GREEN にする。
- [x] T006 [Verify] `mise run check-contract-drift`、`mise run check-api-compat`、変更パッケージのテスト、`mise run verify` を通す。

## Verification

- `mise run test-go-test -- ./backend/shared/http/support_http <対象テスト名>`
- `mise run test-go-package -- ./backend/shared/http/support_http`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run test-go-package -- ./backend/shared/http/testing_stack`
- `mise run check-spec`
- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

別テナントのトークンを拒否する性質は、エラー分類を直す途中でも維持する。
401 応答は audience 不一致、期限切れ、失効、改ざんをすべて同じ `invalid_token` に写し、有効な資格情報の存在を区別できる情報を増やさない。

応答だけでは、拒否後に handler が動いて状態を変えた欠陥を検出できない。
account API では並び順、admin API では Application の保存状態を読み直す。

## Verification Blockers

2026-09-15 の `mise run verify` では、WI-580 の変更先ではない二つの Go テストが失敗した。
どちらも単独再実行で再現する。

- `TestRevokeOneScopesToTheOwnerAndIsIdempotent` は、2026-08-15 に発行して有効期間を 30 日とした端末を実時間で絞り込み、2026-09-15 には 0 件になる。
- `TestResilientDBQueryRowReportsDeadlineExceeded` は、期限切れの `Scan` 直後にも PostgreSQL 接続プールの取得数が 1 のまま残る。

WI-581 は一覧取得の評価時刻を呼び出し側から渡すようにし、WI-582 は非同期の接続破棄完了を上限付きで待つようにした。
両修正を独立コミットへ分離した後、WI-580 だけを含む隔離作業ツリーで全体 `verify` が通過した。

## Completion

- **Completed At**: 2026-09-16
- **Summary**:
  `mise run spec-diff` は REQ-APPLICATION-003 と REQ-APPLICATION-004 のシナリオ変更、および `InvalidAccessTokenError` と `AuthenticationRequiredResponseBody` の TypeSpec 宣言追加を報告した。
  発行先テナントとは異なる既知のテナントへ API アクセストークンを提示した場合は、account API と admin API の双方で 401 `invalid_token` を返す公開契約になった。
  資格情報がない場合は 401 `authentication_required`、同一テナントで認証済みの主体に権限がない場合は 403、未知のテナントまたは認証および認可後に見つからない資源は 404 とする区分も正準設計へ記録した。
- **Primary Use Case Evidence**:
  - id: foreign-tenant-account-token-is-invalid
    unit_red: TestErrorHandlerWritesInvalidTokenAsUnauthorized は旧宣言の 403 を期待した段階で、実際の 401 `invalid_token` を観測して失敗した。
    e2e_red: TestForeignTenantApiTokenCannotReorderApplications は旧宣言の 403 を期待した段階で、実際の 401 `invalid_token` を観測して失敗し、契約との不一致を検出した。
    unit_fault_injection: WriteAccessTokenError の invalid token 分岐を 403 `access_denied` へ差し替えると単体テストが失敗した。
    e2e_fault_injection: inactive な introspection 結果を invalid token ではなく資格情報なしへ写すと、E2E テストが 401 `authentication_required` を検出して失敗した。
  - id: foreign-tenant-admin-token-is-invalid
    unit_red: TestWriteAdminAccessErrorPreservesInvalidTokenClassification は旧宣言の 403 を期待した段階で、実際の 401 `invalid_token` を観測して失敗した。
    e2e_red: TestForeignTenantApiTokenCannotCreateApplication は旧宣言の 403 を期待した段階で、実際の 401 `invalid_token` を観測して失敗し、契約との不一致を検出した。
    unit_fault_injection: admin の invalid token 写像を 403 `access_denied` へ差し替えると単体テストが失敗した。
    e2e_fault_injection: inactive な introspection 結果を資格情報なしへ写すと、E2E テストが 401 `authentication_required` を検出して失敗した。
- **Change-Resistance Results**:
  - 単体境界で invalid token を 403 `access_denied` へ変える故障を注入し、account と admin の両テストが検出した。
  - 組み立て済み境界で inactive token を資格情報なしへ変える配線故障を注入し、account と admin の両 E2E テストがエラー型の違いを検出した。
  - `mise run test-go-mutation -- backend/shared/http/support_http` は coverage 準備中に `runtime/coverage: package testmain: cannot find package` で停止した。
    `mise run clean-go-test-cache` 後も同じため、構文変異の結果は取得できず、変異器が表現できない代表故障の手動注入だけを記録した。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-contract-drift` - passed
  - `mise run check-api-compat` - passed
  - `mise run check-boundaries` - passed
  - `mise run test-go-package -- ./backend/shared/http/support_http` - passed
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run test-go-package -- ./backend/shared/http/testing_stack` - passed
  - `mise run test-go-changed` - passed
  - `mise run lint-go` - 0 issues
  - `mise run verify` - WI-581 と WI-582 を含む HEAD に WI-580 の差分だけを適用した隔離作業ツリーで passed
  - 元作業ツリーの `mise run verify` は、並行して追加された未追跡の WI-583、WI-584、WI-585 にある不正な相対リンクだけを理由に `check` が failed
