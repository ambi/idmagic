---
depends_on: []
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
    - docs/domain/saml/scenarios.feature.md#REQ-SAML-005
    - docs/domain/api-tokens/standards.md#RFC7662-API-TOKEN-INACTIVE
  typespec:
    - IdMagic.Contract.AuthenticationRequiredResponse
    - IdMagic.Contract.InvalidAccessTokenError
  source:
    - backend/apitoken/usecases/usecases.go
    - backend/shared/http/support_http/auth.go
  tests:
    - backend/shared/http/server_http/saml_scope_examples_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/saml/db_postgres
affected_spec:
  - { path: docs/domain/saml/scenarios.feature.md, requirement: REQ-SAML-005 }
---

# テナントの一致しない API アクセストークンが実際に受ける拒否を、具体例に書く

## Motivation

`docs/domain/saml/scenarios.feature.md` の `EX-SAML-005-03` は、トークンのテナントとリクエスト先のテナントが一致しないとき「操作を `AccessDeniedError` で拒否する」と宣言している。

[[wi-550-back-saml-examples-with-tests]] が測ったところ、製品が返すのは 401 `invalid_token` である。`acme` レルムで発行した `saml:read` と `saml:write` のトークンを `default` レルムの `/api/admin/v1/saml/service-providers` へ提示すると、参照も登録も 401 になり、保存先には何も残らない。

拒否そのものは効いている。食い違っているのは拒否の型だけである。

そして 401 の側には理由がある。管理発行トークンの照合は `apitoken/usecases` の `AuthenticateClaims` が `FindByJTI(ctx, リクエスト先テナント, jti)` で行い、見つからなければ RFC 7662 に従って検証の内訳を漏らさず `active: false` を返す。`usecases.go` にはその旨の `//nolint:nilerr` 注記が置いてある。403 `AccessDeniedError` へ変えることは、「このトークン自体は有効だが、このテナントでは使えない」という事実を提示者へ伝えることになる。

つまりこれは写像の欠落ではなく、規範と実装のどちらを正とするかの判断である。

## Scope

- `EX-SAML-005-03` の `Then` を、実装が返す拒否（401 の `InvalidAccessTokenError`）へ合わせる。
- `EX-SAML-005-03` を名指しするテストを書き、`tools/check/example-coverage-debt.json` から外す。テストを置く場所は `backend/shared/http/server_http/saml_scope_examples_test.go` で、同じレルム越えの提示を組み立てる材料はそこに揃っている。

## Out of Scope

- SAML 以外の Context が同じ形で宣言している具体例。`EX-WSFEDERATION-001-03` は同じ経路の食い違いだが、この項目の結論を参照して個別の記録で扱う。
- 管理発行トークンの照合そのものの設計変更。

## Design

`EX-SAML-005-03` を実装へ合わせ、実装は変えない。

判断の材料は次の三つである。

- [[wi-580-align-cross-tenant-application-api-token-refusals]] が Application account API と admin API について同じ判断（発行元と異なるレルムへの提示は 401 `invalid_token`）を既に下し、共通認証境界 `backend/shared/http/support_http/auth.go` の `InvalidTokenError` 判定として実装済みである。SAML admin API も同じ共通境界を通るため、同じ結論を別々に下す理由がない。
- 401 `invalid_token` は RFC 7662 の `RFC7662-API-TOKEN-INACTIVE`（`docs/domain/api-tokens/standards.md`）が要求する非開示と一致する。403 `AccessDeniedError` へ変えることは、「このトークン自体は有効だが、このテナントでは使えない」という事実を提示者へ伝える情報漏えいになる。
- TypeSpec の `RegisterSamlServiceProvider`、`ListSamlServiceProviders`、`DeleteSamlServiceProvider` は既に `IdMagic.Contract.AuthenticationRequiredResponse`（`AuthenticationRequiredError` と `InvalidAccessTokenError` の union）を 401 の応答として宣言している。TypeSpec は変更しない。規範のプローズだけが実装と食い違っていた。

同一テナント内でスコープが不足する場合の 403（`EX-SAML-005-02`）は変えない。テナント越境は認証失敗、スコープ不足は認可失敗という区分を保つ。

## Plan

1. 決めた側（実装）へ規範を寄せ、`spec-change` を通す。
2. `EX-SAML-005-03` を名指しするテストを書き、台帳から外す。

## Tasks

- [x] T001 [Spec] `EX-SAML-005-03` の `Then` を実装が返す 401 `InvalidAccessTokenError` へ書き換え、`mise run check-spec` を通す。
- [x] T002 [Adapters] `EX-SAML-005-03` を名指しするテストを `saml_scope_examples_test.go` に書き、台帳から外す。

## Verification

- `mise run check-spec`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

- **実装を具体例に寄せるほうが速いので、そちらへ流れる。** 401 を 403 に変えるのは 1 行で済むが、それは「このトークン自体は有効である」ことを別テナントの提示者へ伝える変更になる。判断の材料は RFC 7662 の非開示と、そこへ寄せた既存の設計である。速さを理由に選ばない。
- **拒否が効いていることを、型が合っていないことと混同する。** 測定では参照も登録も 401 で止まり、保存先には何も残っていない。この項目が扱うのは型の食い違いだけであり、境界そのものは壊れていない。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` が示す規範の差分は REQ-SAML-005 だけであり、`EX-SAML-005-03` の `Then` が
  「操作を AccessDeniedError で拒否する」から「操作を 401 の InvalidAccessTokenError で拒否する」へ変わったことである。
  具体例が、TypeSpec が既に宣言していた `IdMagic.Contract.AuthenticationRequiredResponse`（401、`InvalidAccessTokenError` を含む union）と、
  共通認証境界 `backend/shared/http/support_http/auth.go` が実際に返す拒否に一致した。
  製品のコードと TypeSpec は変わっていない。この具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
  `EX-WSFEDERATION-001-03` は同じ経路の食い違いを台帳へ残したままとし、Out of Scope のとおり個別の記録で扱う。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-SAML-005
  - **Observed Failure**: `EX-SAML-005-03` を台帳から外した状態で exit 1。
    `EX-SAML-005-03 is declared, but no test names it.` がその 1 件だけを名指しした。
  - **Detection Reason**: テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestForeignTenantApiTokenCannotOperateSamlServiceProviders`
    (`backend/shared/http/server_http/saml_scope_examples_test.go`)
  - **Requirement**: REQ-SAML-005
  - **Observed Failure**: N/A: 製品の振る舞いは変えていないので、実装前に落ちる単体境界は無い。
    代わりに故障を注入して落ちることを確かめた（下の Change-Resistance Results）。
  - **Detection Reason**: `acme` レルムで発行した `saml:read`／`saml:write` トークンを `default` レルムの
    サービスプロバイダー API へ提示し、参照・登録・削除の 3 通りで 401 の状態コード、
    problem type `urn:idmagic:error:invalid_token`、`WWW-Authenticate: Bearer error="invalid_token"` を読む。
    登録と削除については拒否のたびに保存先を読み直し、拒否が保存を防いだ効果まで確かめる。
- **Change-Resistance Results**:
  1. 共通認証境界 `WriteAccessTokenError` の `InvalidTokenError` 分岐を 403 `access_denied` へ差し替えると、
     `TestForeignTenantApiTokenCannotOperateSamlServiceProviders` が status=403 を観測して失敗した。差し替えは復元済み。
  2. `mise run test-go-mutation -- backend/shared/http/server_http` を実行した（efficacy 66.67%、60 killed／30 lived）。
     この記録が変更した拒否の型判定は `backend/shared/http/support_http/auth.go`（別パッケージ）にあり、
     変異器は `./backend/shared/http/server_http` 直下の production ファイル（`gateway_allowlist.go`、
     `health_handler.go`、`priority_class.go`、`priority_class_reference.go`、`routes.go`）だけを対象にした。
     生存した 30 件はいずれもこの記録が触れていない既存コードの分岐であり、本項目の診断能力を測っていない。
     この記録は production コードを変更していないので、変異器で測れる対象そのものが無い。判断は上記 1 の手動注入に拠る。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run lint-go` - 0 issues
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
