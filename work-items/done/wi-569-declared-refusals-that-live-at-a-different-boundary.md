---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 製品の応答は変わらないが、OAuth2 管理 API のクライアント参照、更新、削除が返す 404 OAuth2ClientNotFoundError を公開契約へ新たに宣言するため、API 利用者へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-569.md }
initial_context:
  specification:
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-026
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-035
  typespec:
    - IdMagic.OAuth2.Operations.GetAdminOAuth2Client
    - IdMagic.OAuth2.Operations.UpdateAdminOAuth2Client
    - IdMagic.OAuth2.Operations.DeleteAdminOAuth2Client
    - IdMagic.OAuth2.Operations.Token
    - IdMagic.Contract.McpResourceServerNotFoundError
  source:
    - backend/oauth2/client/usecases/register_client.go
    - backend/oauth2/client/domain/client.go
    - backend/oauth2/handlers_http/admin_client_handler.go
  tests:
    - backend/oauth2/handlers_http/admin_client_handler_test.go
    - backend/shared/http/server_http/token_issuance_standards_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/oauth2/db_postgres
affected_spec:
  - { path: docs/domain/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-026 }
  - { path: docs/domain/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-035 }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.GetAdminOAuth2Client }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.UpdateAdminOAuth2Client }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.DeleteAdminOAuth2Client }
---

# 異なる境界で実施している OAuth2 の拒否を仕様と一致させる

## Motivation

`EX-OAUTH2-026-01` は、public クライアントへ `client_credentials` を指定した登録要求を `InvalidRequestError` で拒否すると宣言している。
現在の実装は登録を受理し、`/token` で `unauthorized_client` を返す。

`EX-OAUTH2-035-02` は、別テナントのクライアント参照を `InvalidRequestError` で拒否すると宣言している。
現在の実装は対象の存在を隠すため、404 Not Found を返す。

どちらも拒否は成立しているが、拒否する境界または利用者が観測するエラー種別が規範と一致しない。
WI-559 が測定結果を台帳へ残したため、本項目が判断と解消を引き取る。

## Scope

- `EX-OAUTH2-026-01` について、public クライアントを登録時に拒否するか、トークンエンドポイントで拒否する現在の境界を規範へ反映するかを決める。
- `EX-OAUTH2-035-02` について、テナントを越えた参照で対象の存在を隠すか、宣言済みの `InvalidRequestError` を返すかを決める。
- 決定に従って仕様を先に更新し、必要なら実装を変更する。
- 各具体例を名指しするテストで、エラー種別と拒否が防いだ効果を固定してから被覆台帳から外す。

## Out of Scope

- `client_credentials` グラントの成功経路。WI-559 のテストが固定している。
- OAuth2 管理 API 全体のテナント境界の再設計。
- 本項目が扱う二つ以外の被覆台帳項目。

## Design

`EX-OAUTH2-026-01` では、登録契約とトークンエンドポイントの責務を分けて判断する。
登録時の拒否を採る場合は、public クライアントが別のグラントを利用できる可能性を失わない入力条件を定める。
現在の境界を採る場合は、登録自体ではなくグラント利用を拒否することが読者へ伝わるように具体例を直す。

`EX-OAUTH2-035-02` では、対象の存在を別テナントへ漏らさない性質と、具体例が名指すエラー種別を同時に評価する。
404 Not Found を採る場合は非開示の理由を正準文書へ残し、`InvalidRequestError` を採る場合は情報開示が増えないことをテストで示す。

二つの判断を同じエラー名へ揃えることは目的にしない。
呼び出し境界と保護する情報が異なるため、それぞれの責務から結論を出す。

### 決定

| 具体例 | 採る境界とエラー | 理由 |
| --- | --- | --- |
| `EX-OAUTH2-026-01` | 現在の境界を規範へ反映する。public クライアントの登録は受理し、`/token` が `UnauthorizedClientError`（`unauthorized_client`）で拒否してトークンを発行しない | RFC 6749 §4.4 が制限するのはこのグラントの利用であり、`unauthorized_client` はクライアントがそのグラント種別を使えないことを表す登録済みの値である。`Token` の 400 union は `UnauthorizedClientError` を既に宣言している。登録時の拒否へ寄せると、管理 API の作成が返す `invalid_client_metadata` が TypeSpec の宣言する `InvalidRequestError` と一致しない別の食い違いに依存することになる |
| `EX-OAUTH2-035-02` | 対象の存在を明かさない 404 を正とし、`OAuth2ClientNotFoundError`（`urn:idmagic:error:client_not_found`）として `GetAdminOAuth2Client`、`UpdateAdminOAuth2Client`、`DeleteAdminOAuth2Client` に宣言する | `GetAdminOAuth2Client` の doc は別テナントのクライアントを存在しないものとして扱うと既に定めている。同じ Context の `McpResourceServerNotFoundError` と `AuthorizationDetailTypeNotFoundError` も同じ形で 404 を宣言している。実装は 404 `client_not_found` を返しているが TypeSpec が宣言していなかった |

実装コードは変えない。
変わるのは具体例の文言、TypeSpec の応答宣言、テスト、被覆台帳である。

`EX-OAUTH2-035-02` のテストは、別テナントの管理者による参照、更新、削除が、存在しない `client_id` を指定したときと同じ状態行と problem type を返し、応答本文に対象の設定を含まず、`acme` のクライアントが変更も削除もされないことを検査する。

`EX-OAUTH2-026-01` のテストは、`/token` の応答の `error` が `unauthorized_client` であり、トークンを 1 つも含まないことを検査する。

## Plan

1. TypeSpec、OAuth2 シナリオ、登録処理、管理 API のテナント解決を読み、現在の契約と非開示方針を確認する。
2. 各具体例について規範と実装のどちらを変更するか決め、仕様を先に更新する。
3. 観測可能な HTTP 境界で RED を確認し、必要な実装変更とテストを行う。
4. 被覆台帳から二件を外し、標準検証を通す。

## Tasks

- [x] T001 [Spec] `EX-OAUTH2-026-01` と `EX-OAUTH2-035-02` を決定した拒否境界とエラー種別へ書き換え、TypeSpec に `OAuth2ClientNotFoundError` を宣言する。
- [x] T002 [Acceptance] 二件を被覆台帳から外し、`mise run check-spec` がテストの欠落を RED として報告することを確かめる。
- [x] T003 [App] 決定は実装変更を求めないため、実装は変えない。
- [x] T004 [Tests] エラー種別と拒否が防いだ効果を検査するテストで二件を名指しし、故障注入で検出能力を確かめる。
- [x] T005 [Verify] `mise run verify` を通す。

## Verification

- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

- 登録時の拒否へ寄せると、public クライアントが利用できる別のグラントまで登録不能にするおそれがある。登録要求のどの組み合わせを拒否するかを先に仕様で限定する。
- テナントを越えた参照のエラーを変えると、対象の存在を推測できる差が生じるおそれがある。状態行だけでなく応答本文と対象テナントの保存状態も検査する。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` が示す規範の差分は、REQ-OAUTH2-026 と REQ-OAUTH2-035 の具体例、および TypeSpec に追加した
  `OAuth2ClientNotFoundError` である。
  `EX-OAUTH2-026-01` の最後の When と Then は、「public クライアントの登録を `InvalidRequestError` で拒否する」から、
  「`client_credentials` を宣言して登録済みの public クライアントがトークンを要求すると `UnauthorizedClientError` で拒否され、
  トークンは発行されない」へ変わった。RFC 6749 §4.4 が制限するのはグラントの利用であり、`Token` の 400 union は
  `UnauthorizedClientError` を既に宣言していた。
  `EX-OAUTH2-035-02` の Then は、「`InvalidRequestError` で拒否される」から、「別テナントの管理者による参照、更新、削除は
  どれも `OAuth2ClientNotFoundError` で拒否され、応答は存在しない `client_id` と同じであり、`acme` のクライアントは変更も
  削除もされない」へ変わった。TypeSpec は `GetAdminOAuth2Client`、`UpdateAdminOAuth2Client`、`DeleteAdminOAuth2Client` に
  404 `OAuth2ClientNotFoundError` を宣言し、更新と削除の doc にも別テナントのクライアントを存在しないものとして扱うことを書いた。
  製品のコードは変わっていない。二つの具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
  公開契約に 404 が加わるため、`docs/releases/changes/wi-569.md` にリリースノートを置いた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-OAUTH2-035
  - **Observed Failure**: 二件を台帳から外しテストを書く前の状態で exit 1。
    `EX-OAUTH2-026-01 is declared, but no test names it.` と `EX-OAUTH2-035-02 is declared, but no test names it.` の二件だけを名指しした。
  - **Detection Reason**: テストを書かずに台帳から外せば必ず落ちるため、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestClientCredentialsIsConfidentialOnlyAndPasswordGrantIsNotOffered`
    (`backend/shared/http/server_http/token_issuance_standards_test.go`)、
    `TestAdminOAuth2ClientOfAnotherTenantIsAnsweredAsNonexistent`
    (`backend/oauth2/handlers_http/admin_client_handler_test.go`)
  - **Requirement**: REQ-OAUTH2-026
  - **Observed Failure**: N/A: 製品の振る舞いを変えていないため、実装前に落ちる単体境界はない。
    代わりに故障を注入して落ちることを確かめた（下の Change-Resistance Results）。
  - **Detection Reason**: 前者は public クライアントの `client_credentials` 要求について、状態行 400、`error` が
    `unauthorized_client`、本文にトークンを含まないことを読む。後者は別テナントの `portal` への参照、更新、削除のそれぞれを
    存在しない `client_id` への同じ操作と比べ、どちらも 404 で、`instance` を除いた problem 本文が一致し、type が
    `urn:idmagic:error:client_not_found` であり、本文に対象の設定を含まないことを読む。さらに `acme` の `portal` が残り、
    `redirect_uris` が変わらず、Admin イベントが 1 件も発行されないことを読む。
- **Change-Resistance Results**:
  1. `token_handler.go` の public クライアント拒否の `error` を `invalid_request` へ差し替えると、前者が
     `error="invalid_request", want "unauthorized_client"` で失敗した。
  2. 同じ分岐の条件を常に偽にすると、前者が status=200 とトークンを観測して失敗した。
  3. `writeAdminOAuth2ClientError` の not-found 写像を 400 `invalid_request` へ差し替えると、後者が
     `GET: cross-tenant status=400 nonexistent status=400, want 404` で失敗した。
  4. `DeleteAdminOAuth2Client` の検索テナントを `acme` へ差し替える（作用の宛先を別テナントへ向ける）と、後者が
     `DELETE: cross-tenant status=204 nonexistent status=404` で失敗した。
  いずれの差し替えも復元済みである。この記録は production コードを変更していないため、`mise run test-go-mutation` で
  測れる変更はない。判断は上記 1〜4 の手動注入に拠る。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - passed
  - `mise run check-contract-drift` - passed
  - `mise run check-status-drift` - passed
  - `mise run lint-go` - 0 issues
  - `mise run test-go-package -- ./backend/oauth2/handlers_http` - passed
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run test-go-changed` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed (exit 0)
