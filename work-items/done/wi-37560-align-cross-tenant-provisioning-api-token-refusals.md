---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-21
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 具体例の文言を、既に契約が宣言し製品が返している 401 に合わせる変更であり、製品の振る舞いも公開契約も変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-001
    - docs/domain/saml/scenarios.feature.md#REQ-SAML-005
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-003
  typespec:
    - IdMagic.Contract.AuthenticationRequiredResponse
    - IdMagic.Contract.InvalidAccessTokenError
  source:
    - backend/shared/http/support_http/auth.go
    - backend/apitoken/usecases/usecases.go
  tests:
    - backend/shared/http/server_http/provisioning_api_token_scope_test.go
    - backend/shared/http/server_http/saml_scope_examples_test.go
    - backend/oauth2/handlers_http/oauth2_scope_examples_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/provisioning
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-001 }
---

# テナントの一致しない Provisioning API トークンが実際に受ける拒否を、具体例に書く

## 動機

`EX-PROVISIONING-001-03` は、トークンのテナントとリクエスト先のテナントが異なる場合に `AccessDeniedError` を返すと宣言していた。
製品は、他テナントで発行した有効な `provisioning:write` トークンを default tenant の入口へ提示されると、状態を変えずに `401 invalid_token` を返す。

この記録は当初、実装を 403 `AccessDeniedError` へ寄せる bugfix として起票した。
準備段階の確認で、同じ食い違いについて Application、SAML、OAuth2 の各 Context が既に規範の側を 401 `InvalidAccessTokenError` へ合わせていることがわかった。
そのため方針を改め、規範を実装へ合わせる。

## 対象範囲

- `EX-PROVISIONING-001-03` の `Then` を、実装が返す拒否（401 の `InvalidAccessTokenError`）へ合わせる。
- `EX-PROVISIONING-001-03` を名指しするテストで、接続と配信の参照と変更が 401 `invalid_token` で拒否され、接続が作られないことを固定する。
- `tools/check/example-coverage-debt.json` から `EX-PROVISIONING-001-03` を外す。

## 対象外

- 他 Context のクロステナント拒否形式の変更。`EX-WSFEDERATION-001-03` は同じ経路の食い違いだが、個別の記録で扱う。
- API トークンの署名方式、トークン形式、管理発行トークンの照合の設計変更。
- 製品コードと TypeSpec の変更。

## 設計

`EX-PROVISIONING-001-03` を実装へ合わせ、実装は変えない。
判断の材料は次のとおりである。

| 観点 | 内容 |
| --- | --- |
| 既存の判断との一致 | Application の account API と admin API、SAML（`EX-SAML-005-03`）、OAuth2（`EX-OAUTH2-003-04`）は、発行元と異なるテナントへの提示を 401 `InvalidAccessTokenError` と宣言している。Provisioning の管理 API も同じ共通認証境界 `backend/shared/http/support_http/auth.go` を通る。 |
| 非開示 | 管理発行トークンの照合は `apitoken/usecases` の `AuthenticateClaims` がリクエスト先テナントで jti を探し、見つからなければ `active: false` を返す。403 にすると「このトークン自体は有効だが、このテナントでは使えない」ことを提示者へ伝える。401 は `RFC7662-API-TOKEN-INACTIVE` の非開示と一致する。 |
| 公開契約 | Provisioning の各操作は、TypeSpec で 401 の `IdMagic.Contract.AuthenticationRequiredResponse`（`AuthenticationRequiredError` と `InvalidAccessTokenError` の union）を既に宣言している。TypeSpec は変更しない。 |

同一テナント内でスコープが不足する場合の 403（`EX-PROVISIONING-001-02`）は変えない。
テナント越境は認証失敗、スコープ不足は認可失敗という区分を保つ。

## 計画

1. 規範の `Then` を 401 `InvalidAccessTokenError` へ書き換え、`mise run check-spec` を通す。
2. 被覆台帳から外し、`check-spec` の RED を確認する。
3. `EX-PROVISIONING-001-03` を名指しするテストを書き、GREEN にする。

## タスク

- [x] T001 [Readiness] 共通認証境界と既存 Context の判断を確かめ、規範を実装へ合わせる方針に決める。
- [x] T002 [Spec] `EX-PROVISIONING-001-03` の `Then` を 401 の `InvalidAccessTokenError` へ書き換える。
- [x] T003 [Acceptance] 台帳から外した状態で `mise run check-spec` の RED を確認する。
- [x] T004 [Adapters] `TestForeignTenantProvisioningTokenIsRejectedAsInvalid` に `//spec:covers EX-PROVISIONING-001-03` を付け、4 操作の拒否と接続の不変を固定する。
- [x] T005 [Verify] `mise run verify` を実行する。

## 検証

- `mise run check-spec`
- `mise run test-go-test -- ./backend/shared/http/server_http TestForeignTenantProvisioningTokenIsRejectedAsInvalid`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run check-work-items`
- `mise run verify`

## リスク

- **実装を具体例に寄せるほうが速いので、そちらへ流れる。** 401 を 403 に変えると、別テナントの提示者へトークンが有効であることを伝える。速さを理由に選ばない。
- **拒否が効いていることと、型が合っていないことを混同する。** 拒否された登録と全件再同期の後に保存先を読み、接続が作られていないことを同じテストで確かめる。

## 完了

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` が示す規範の差分は REQ-PROVISIONING-001 だけであり、`EX-PROVISIONING-001-03` の `Then` が
  「操作は `AccessDeniedError` で拒否される」から「操作は 401 の `InvalidAccessTokenError` で拒否される」へ変わったことである。
  具体例が、TypeSpec が既に宣言していた `IdMagic.Contract.AuthenticationRequiredResponse` と、
  共通認証境界 `backend/shared/http/support_http/auth.go` が実際に返す拒否に一致した。
  製品のコードと TypeSpec は変わっていない。この具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
  記録は当初 403 へ寄せる bugfix として起票したが、Application、SAML、OAuth2 の既存の判断と RFC 7662 の非開示に合わせて、規範を実装へ寄せる方針へ改めた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-PROVISIONING-001
  - **Observed Failure**: `EX-PROVISIONING-001-03` を台帳から外した状態で exit 1。
    `EX-PROVISIONING-001-03 is declared, but no test names it.` がその 1 件だけを名指しした。
  - **Detection Reason**: テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestForeignTenantProvisioningTokenIsRejectedAsInvalid`
    (`backend/shared/http/server_http/provisioning_api_token_scope_test.go`)
  - **Requirement**: REQ-PROVISIONING-001
  - **Observed Failure**: N/A: 製品の振る舞いは変えていないので、実装前に落ちる単体境界はない。
    代わりに故障を注入して落ちることを確かめた（下の Change-Resistance Results）。
  - **Detection Reason**: `acme` レルムで発行した `provisioning:read` と `provisioning:write` のトークンを `default` レルムへ提示し、
    テナントの接続一覧、配信一覧、接続の登録、全件再同期の 4 操作で 401、problem type `urn:idmagic:error:invalid_token`、
    `WWW-Authenticate: Bearer error="invalid_token"` を読む。拒否の後に保存先を読み、接続が作られていないことも確かめる。
    同一テナントのトークンで一覧が 200 になる対照を置き、拒否の原因がテナントの食い違いであることを示す。
- **Change-Resistance Results**:
  1. 共通認証境界 `WriteAccessTokenError` の `InvalidTokenError` 分岐を 403 `access_denied` へ差し替えると、
     `TestForeignTenantProvisioningTokenIsRejectedAsInvalid` が status=403 を観測して失敗した。差し替えは復元済み。
  2. `mise run test-go-mutation` は実行していない。この記録は production コードを変更しておらず、変異器で測れる対象がない。判断は上記 1 の手動注入に拠る。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run lint-go` - 0 issues
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
