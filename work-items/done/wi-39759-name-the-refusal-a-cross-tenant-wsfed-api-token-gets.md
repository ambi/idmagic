---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-25
priority: p3
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 具体例の文言を、既に契約が宣言し製品が返している 401 に合わせる変更であり、製品の振る舞いも公開契約も変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/ws-federation/scenarios.feature.md#REQ-WSFEDERATION-001
    - docs/domain/saml/scenarios.feature.md#REQ-SAML-005
  typespec: []
  source: []
  tests:
    - backend/wsfederation/handlers_http/scope_examples_test.go
    - backend/shared/http/server_http/saml_scope_examples_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/apitoken
    - backend/wsfederation/db_postgres
    - tools/coverage-debt-report/src
affected_spec:
  - { path: docs/domain/ws-federation/scenarios.feature.md, requirement: REQ-WSFEDERATION-001 }
---

# テナントの一致しない API アクセストークンが WS-Federation の管理 API で受ける拒否を、具体例に書く

## 動機

`docs/domain/ws-federation/scenarios.feature.md` の `EX-WSFEDERATION-001-03` は、トークンのテナントとリクエスト先のテナントが一致しないとき「操作を `AccessDeniedError` で拒否する」と宣言している。

[[wi-554-back-ws-federation-examples-with-tests]] の実測では、製品が返すのは 401 `invalid_token` である。
`acme` レルムで発行した `wsfed:read` と `wsfed:write` のトークンを `default` レルムの `/api/admin/v1/wsfed/relying-parties`（一覧、登録、削除）と `/api/admin/v1/wsfed/entra-federation` へ提示すると、8 通りすべてが 401 になり、保存先の RP は変わらない。

これは `EX-SAML-005-03` と同じ経路の食い違いである。
[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] は SAML について、RFC 7662 の非開示に従う 401 を正とし、具体例を `InvalidAccessTokenError` へ合わせた。
同項目は `EX-WSFEDERATION-001-03` を Out of Scope とし「個別の記録で扱う」と書いたが、その記録は起票されていなかった。
この具体例は `tools/check/example-coverage-debt.json` に `blocked_by: wi-558` のまま残り、[[wi-496-burn-down-the-example-coverage-debt]] が台帳を空にできない。

## 対象範囲

- `EX-WSFEDERATION-001-03` の `Then` を、実装が返す拒否（401 の `InvalidAccessTokenError`）へ合わせる。
- `EX-WSFEDERATION-001-03` を名指しするテストを書き、`tools/check/example-coverage-debt.json` から外す。テストは RP の一覧、登録、削除と Entra フェデレーションの操作を列挙し、拒否応答と保存先の RP が変わらないことの双方を観測する。

## 対象外

- 管理発行トークンの照合そのものの設計変更。
- 403 `AccessDeniedError` へ実装を寄せる変更。wi-558 が却下した理由（トークン自体は有効だという事実を別テナントの提示者へ伝える）がそのまま当てはまる。

## 設計

wi-558 の結論を踏襲する。
判断の材料は RFC 7662 の非開示と、`apitoken/usecases` の `AuthenticateClaims` がリクエスト先テナントで jti を照合し、見つからなければ `active: false` を返す既存の設計である。
TypeSpec はすでに 401 の `IdMagic.Contract.AuthenticationRequiredResponse` を宣言しているので、TypeSpec と製品のコードは変わらない見込みである。

## 計画

1. `spec-change` で `EX-WSFEDERATION-001-03` の `Then` を書き換え、`mise run check-spec` を通す。
2. 台帳から外した状態で `mise run check-spec` が落ちることを確かめ、テストを書いて通す。

## タスク

- [x] T001 [Spec] `EX-WSFEDERATION-001-03` の `Then` を 401 の `InvalidAccessTokenError` へ書き換える。
- [x] T002 [Acceptance] 台帳から外した状態で `mise run check-spec` の RED を確認する。
- [x] T003 [Test] 8 通りの提示を列挙するテストを書き、`//spec:covers EX-WSFEDERATION-001-03` を付ける。
- [x] T004 [Verify] `mise run verify`。

各 RED、GREEN、故障注入には `mise run test-go-test -- ./backend/wsfederation/handlers_http TestForeignTenantApiTokenCannotOperateWsFedTrustSettings` を使い、振る舞いが緑になった時点で `mise run test-go-package -- ./backend/wsfederation/handlers_http` と `mise run lint-go` を 1 回ずつ実行する。

## 検証

- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## リスク

- **実装を具体例に寄せるほうが速いので、そちらへ流れる。** 401 を 403 に変えることは非開示を崩す変更であり、速さを理由に選ばない。

## Completion

- **Completed At**: 2026-09-26
- **Summary**:
  `mise run spec-diff` が示す規範の差分は REQ-WSFEDERATION-001 だけであり、`EX-WSFEDERATION-001-03` の `Then` が
  「操作を AccessDeniedError で拒否する」から「操作を 401 の InvalidAccessTokenError で拒否する」へ変わったことである。
  `Given` と `When` も `EX-SAML-005-03` と同じ形へ揃え、トークンが発行元テナントでは有効であることと、提示先が別テナントの 4 つの操作であることを明示した。
  具体例が、TypeSpec が既に宣言していた 401 の `IdMagic.Contract.AuthenticationRequiredResponse` と、共通認証境界が実際に返す拒否に一致した。
  製品のコードと TypeSpec は変わっていない。
  この具体例は名指すテストを得て `tools/check/example-coverage-debt.json` から外れ、台帳の `untested` は空になった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-WSFEDERATION-001
  - **Observed Failure**: `EX-WSFEDERATION-001-03` を台帳から外した状態で exit 1。
    `EX-WSFEDERATION-001-03 is declared, but no test names it.` がその 1 件だけを名指しした。
  - **Detection Reason**: テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestForeignTenantApiTokenCannotOperateWsFedTrustSettings`
    (`backend/wsfederation/handlers_http/scope_examples_test.go`)
  - **Requirement**: REQ-WSFEDERATION-001
  - **Observed Failure**: N/A: 製品の振る舞いは変えていないので、実装前に落ちる単体境界はない。
    代わりに故障を注入して落ちることを確かめた（下の Change-Resistance Results）。
  - **Detection Reason**: `acme` レルムで発行した `wsfed:read` と `wsfed:write` のトークンを `default` レルムの RP の一覧、登録、削除と
    Entra フェデレーションの構成へ提示し、8 通りすべてで 401 の状態コード、problem type `urn:idmagic:error:invalid_token`、
    `WWW-Authenticate: Bearer error="invalid_token"` を読む。
    提示の後に保存先の RP を読み直して増減がないことを確かめ、同じ削除が `default` 発行の `wsfed:write` なら通ることを対照にする。
- **Change-Resistance Results**:
  1. 共通認証境界 `WriteAccessTokenError` の `InvalidTokenError` 分岐を 403 `access_denied` へ差し替えると、
     `TestForeignTenantApiTokenCannotOperateWsFedTrustSettings` が最初の提示（`wsfed:read` の RP の一覧）で status=403 を観測して失敗した。差し替えは復元済み。
  2. `mise run test-go-mutation -- backend/wsfederation/handlers_http` を実行した（efficacy 90.67%、68 killed／7 lived、10 not covered）。
     生存した 7 件は `admin_entra_handler.go:74`（reply URL の既定値）、`admin_relying_party_handler.go:116`（アプリケーション所有 RP の削除拒否）、
     `wsfed_handler.go:85`、`wsfed_handler.go:122`、`wstrust_handler.go:82` であり、いずれもテナントの照合経路の外にある既存の分岐である。
     この記録は production コードを変更していないので、判断は上記 1 の手動注入に拠る。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run lint-go` - 0 issues
  - `mise run test-go-package -- ./backend/wsfederation/handlers_http` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
