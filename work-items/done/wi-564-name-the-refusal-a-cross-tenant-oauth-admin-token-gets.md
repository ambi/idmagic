---
depends_on: [wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 具体例の文言を、既に契約が宣言し製品が返している 401 に合わせる変更であり、製品の振る舞いも公開契約も変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-003
    - docs/domain/api-tokens/standards.md#RFC7662-API-TOKEN-INACTIVE
  typespec:
    - IdMagic.Contract.AuthenticationRequiredResponse
    - IdMagic.Contract.InvalidAccessTokenError
  source:
    - backend/apitoken/usecases/usecases.go
    - backend/shared/http/support_http/auth.go
  tests:
    - backend/oauth2/handlers_http/oauth2_scope_examples_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/oauth2/db_postgres
affected_spec:
  - { path: docs/domain/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-003 }
---

# テナントの一致しない OAuth2 管理 API トークンが実際に受ける拒否を、具体例に書く

## Motivation

`docs/domain/oauth2/scenarios.feature.md` の `EX-OAUTH2-003-04` は、トークンのテナントとリクエスト先のテナントが一致しないとき「操作を `AccessDeniedError` で拒否する」と宣言している。

[[wi-538-back-oauth2-examples-with-tests]] が測ったところ、製品が返すのは 401 `invalid_token` である。`acme` レルムで発行した `oauth-clients:read` と `oauth-clients:write` のトークンを `default` レルムの `/api/admin/v1/clients` へ提示すると、参照も登録も 401 になり、`default` テナントのクライアント一覧は 1 件も動かない。`WWW-Authenticate` は `Bearer error="invalid_token"` を返す。

拒否そのものは効いている。食い違っているのは拒否の型だけである。

これは [[wi-550-back-saml-examples-with-tests]] が `EX-SAML-005-03` について測ったものと同じ食い違いであり、原因も同じである。管理発行トークンの照合は `apitoken/usecases` の `AuthenticateClaims` が `FindByJTI(ctx, リクエスト先テナント, jti)` で行い、見つからなければ RFC 7662 に従って検証の内訳を漏らさず扱う。403 `AccessDeniedError` へ変えることは、「このトークン自体は有効だが、このテナントでは使えない」という事実を提示者へ伝えることになる。

したがって判断は 1 つで、対象の具体例が 2 つある。[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] が SAML 側で判断を下し、本項目はその結論を OAuth2 側へ適用する。

## Scope

- wi-558 の結論に従って `EX-OAUTH2-003-04` を解決する。規範を直すなら `spec-change` を通し、実装を直すなら RFC 7662 の非開示との折り合いを記録に残す。
- 決着後、`EX-OAUTH2-003-04` を名指しするテストを書き、`tools/check/example-coverage-debt.json` から外す。テストを置く場所は `backend/shared/http/server_http/oauth2_scope_examples_test.go` で、レルム越えの提示を組み立てる材料 (`apiTokenStack` の `issue` と `oauthAdminRequest`) はそこに揃っている。

## Out of Scope

- wi-558 が下す判断そのもののやり直し。本項目は結論を適用する側である。
- 管理発行トークンの照合そのものの設計変更。
- REQ-OAUTH2-003 のほかの具体例。[[wi-538-back-oauth2-examples-with-tests]] が消化済みである。

## Design

[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] が SAML 側で下した判断（401
`InvalidAccessTokenError` へ規範を寄せ、実装は変えない）をそのまま適用する。OAuth2 admin API も
同じ共通認証境界 `backend/shared/http/support_http/auth.go` の `WriteAccessTokenError` を通るため、
別の結論を出す理由がない。TypeSpec の `CreateAdminOAuth2Client` などは既に
`IdMagic.Contract.AuthenticationRequiredResponse`（401 の union）を宣言しているので、TypeSpec も
変更しない。規範のプローズだけが実装と食い違っていた。

## Plan

1. `EX-OAUTH2-003-04` の `Then` を実装が返す 401 `InvalidAccessTokenError` へ書き換え、
   `mise run check-spec` を通す。
2. `EX-OAUTH2-003-04` を名指しするテストを `backend/oauth2/handlers_http/oauth2_scope_examples_test.go`
   に書き、台帳から外す。

## Tasks

- [x] T001 [Spec] `EX-OAUTH2-003-04` の `Then` を実装が返す 401 `InvalidAccessTokenError` へ書き換え、
  `mise run check-spec` を通す。
- [x] T002 [Adapters] `EX-OAUTH2-003-04` を名指しするテストを `oauth2_scope_examples_test.go` に書き、
  台帳から外す。

## Verification

- `mise run check-spec` が、`EX-OAUTH2-003-04` を台帳から外した状態で通る。
- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run verify`

## Risk Notes

- **SAML 側と別の結論を出す。** 同じ照合、同じ理由、同じ食い違いなので、2 つの具体例が別々の拒否の型を名指す状態は、規範の読み手に対して嘘になる。wi-558 の結論からずれるなら、ずれる理由を記録に書く。
- **拒否の型だけを直して、拒否が防いだ効果を観測しない。** 型を 403 へ変える実装も 401 のまま残す実装も、保存先を動かさないことまで読まなければ区別できない。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` が示す規範の差分は REQ-OAUTH2-003 だけであり、`EX-OAUTH2-003-04` の `Then` が
  「操作は `AccessDeniedError` で拒否される」から「操作を 401 の InvalidAccessTokenError で拒否する」へ変わったことである。
  [[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] が SAML 側で下した判断（管理発行トークンの
  照合は共通境界 `backend/shared/http/support_http/auth.go` を通り、RFC 7662 の非開示に従って 401 で拒否する）を
  OAuth2 admin API へそのまま適用した。TypeSpec が既に宣言していた `IdMagic.Contract.AuthenticationRequiredResponse`
  （401、`InvalidAccessTokenError` を含む union）と、実際に返る拒否が一致した。製品のコードと TypeSpec は変わっていない。
  この具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
  テストの置き場所は、work item が起票時に見込んでいた `backend/shared/http/server_http/oauth2_scope_examples_test.go`
  ではなく、`backend/oauth2/handlers_http/oauth2_scope_examples_test.go`（`EX-OAUTH2-003-01`〜`03` が既にある場所）
  になった。同じ `oauthAdminRequest` と `stack.OtherRealm` が揃っており、具体例を分割する理由がなかった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-OAUTH2-003
  - **Observed Failure**: `EX-OAUTH2-003-04` を台帳から外した状態で exit 1。
    `EX-OAUTH2-003-04 is declared, but no test names it.` がその 1 件だけを名指しした。
  - **Detection Reason**: テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestForeignTenantApiTokenCannotOperateOAuth2AdminResources`
    (`backend/oauth2/handlers_http/oauth2_scope_examples_test.go`)
  - **Requirement**: REQ-OAUTH2-003
  - **Observed Failure**: N/A: 製品の振る舞いは変えていないので、実装前に落ちる単体境界は無い。
    代わりに故障を注入して落ちることを確かめた（下の Change-Resistance Results）。
  - **Detection Reason**: `acme` レルムで発行した OAuth2 クライアント・認可詳細タイプ・MCP リソースサーバーの
    各スコープトークンを `default` レルムの管理 API へ提示し、参照・登録の 3 resource で 401 の状態コード、
    problem type `urn:idmagic:error:invalid_token`、`WWW-Authenticate: Bearer error="invalid_token"` を読む。
    登録については拒否のたびに保存先を読み直し、拒否が保存を防いだ効果まで確かめる。対照として、同一テナントの
    トークンなら OAuth2 クライアントの参照が通ることも読み、拒否がテナントの食い違いによるものだと示す。
- **Change-Resistance Results**:
  1. 共通認証境界 `WriteAccessTokenError` の `InvalidTokenError` 分岐を 403 `access_denied` へ差し替えると、
     `TestForeignTenantApiTokenCannotOperateOAuth2AdminResources` が status=403 を観測して失敗した。差し替えは復元済み。
  2. `mise run test-go-mutation -- backend/oauth2/handlers_http` は開始したが、対象パッケージの本番コードが
     4,613 行あり、gomutants は変異ごとにパッケージのテスト（約 7 秒）を再実行するため長時間を要すると判断し中止した。
     この記録はテストファイルと仕様文書だけを変更し、production コードを変更していないので、変異器で測れる対象
     そのものが無い（wi-558 の完了記録と同じ判断）。判断は上記 1 の手動注入に拠る。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run lint-go` - 0 issues
  - `mise run test-go-package -- ./backend/oauth2/handlers_http` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed (exit 0)
