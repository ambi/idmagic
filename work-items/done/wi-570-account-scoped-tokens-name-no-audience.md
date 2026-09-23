---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: resource を指定しない account スコープのアクセストークンの aud が client_id からレルムの発行者識別子へ変わり、account リソースサーバーはその audience を持たない account スコープのトークンを 401 で拒否するようになる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-570.md }
initial_context:
  specification:
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-001
    - docs/domain/oauth2/standards.md#RFC8707-MCP-RESOURCE-BINDING
    - docs/domain/api-tokens/standards.md#RFC9700-API-TOKEN-AUDIENCE
    - docs/design/security/authorization.md
  typespec: []
  source:
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/oauth2/ports/token_issuer.go
    - backend/oauth2/usecases/resource_indicator.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/shared/http/support_http/auth.go
    - backend/apitoken/usecases/usecases.go
    - backend/tenancy/context.go
  tests:
    - backend/oauth2/handlers_http/account_scope_examples_test.go
    - backend/shared/http/support_http/auth_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/oauth2/db_postgres
affected_spec:
  - { path: docs/domain/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-001 }
  - { path: docs/domain/oauth2/standards.md, requirement: RFC9068-DEFAULT-AUDIENCE }
  - { path: docs/domain/oauth2/standards.md, requirement: RFC8707-MCP-RESOURCE-BINDING }
primary_use_cases:
  - id: account-token-names-realm-api
    requirement: REQ-OAUTH2-001
    observable_result: account:read に同意した認可コードの交換が、sub が同意した User、aud がレルムの発行者識別子、scope が account:read のアクセストークンを返し、そのトークンで account リソースサーバーの参照だけが通る。
    unit_test: { path: backend/shared/security/tokens_jose/jwt_signer_audience_test.go, name: TestAccessTokenWithoutResourceInfersAudienceFromScope, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/account_scope_examples_test.go, name: TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts, task: test-go-race }
    unit_fault_model: 署名器が account スコープを持つ resource 未指定のトークンでも client_id を aud にする。
    e2e_fault_model: 発行経路が Audiences に client_id を明示して渡し、署名器の推定を迂回する。
  - id: account-api-refuses-foreign-audience
    requirement: REQ-OAUTH2-001
    observable_result: aud がリクエスト先レルムの発行者識別子を含まない account スコープのトークンを、account リソースサーバーは 401 invalid_token で拒否し、ポータル境界のスコープだけを持つトークンは従来どおり受ける。
    unit_test: { path: backend/shared/http/support_http/auth_test.go, name: TestAccountScopedTokenMustNameTheRealmApi, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/account_scope_examples_test.go, name: TestAccountApiRefusesAccountScopedTokenForAnotherAudience, task: test-go-race }
    unit_fault_model: Authenticator が account スコープのトークンの aud を読まずにスコープだけで受理する。
    e2e_fault_model: 製品の組み立てで account API の Bearer 検証が audience の検査を通らず、スコープだけで受理する。
---

# account スコープのトークンが、宛先のリソースサーバーを名乗らない

## Motivation

[[wi-559-back-oauth2-remaining-examples-with-tests]] が `EX-OAUTH2-001-01` を消化する過程で、実装が具体例の `Then` を 1 つ満たしていないことを測った。

具体例は「アクセストークンの `sub` は同意した User、**audience はレルムの IdMagic API**、スコープは `account:read` になる」と言う。実測では `sub` と `scope` は宣言どおりだが、`aud` は `client_id` だった。

**レルムの IdMagic API を audience として名乗る経路が存在しない。** 理由は 3 つある。

1. `tokens_jose.JWTSigner.SignAccessToken` は、`Audiences` が空なら `aud` を `client_id` にする。
2. `Audiences` を埋めるのは resource indicator の解決だけで、`ResolveResourceIndicator` は登録済みの `McpResourceServer` しか受け付けない。レルム自身の API はそこに登録されていない。`/authorize` へ `resource=<レルムの発行者>` を付けると、認可リクエストごと拒否される。
3. account リソースサーバー側 (`support_http.Authenticator`) は audience を 1 度も読まない。判定に使うのはスコープだけである。

つまり「名乗っていない」だけでなく「名乗っても誰も読まない」状態で、直すには発行と検証の両方に判断が要る。

`sub`、`scope`、そして account リソースサーバーが参照だけを許すことは wi-559 が `TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts` で固定した。残るのは audience だけである。

## Scope

- レルムの IdMagic API を audience として名乗る経路を決める。resource indicator の解決先にレルム自身を加えるのか、account スコープの発行だけが特別に audience を差し替えるのかを Design に書く。
- account リソースサーバーが audience を検証するかどうかを決める。検証しないなら、具体例から audience の `Then` を落とす規範の変更として扱う。
- 決めた側に合わせて、`EX-OAUTH2-001-01` を名指すテストを `TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts` へ足し、`tools/check/example-coverage-debt.json` から当該行を外す。

## Out of Scope

- `McpResourceServer` に対する resource indicator の既存の振る舞い。RFC 8707 の行は別の記録が持つ。
- REQ-OAUTH2-001 のほかの具体例。`EX-OAUTH2-001-02` と `EX-OAUTH2-001-03` は消化済みである。
- ほかの Context のアクセストークンの audience。
- ポータル境界のスコープ（`idmagic.account`、`idmagic.admin`）だけを持つトークンの audience と、admin API での audience の検証。

## Design

RFC 9068 §3 は、`resource` を含まないトークン要求にもデフォルトの resource indicator を `aud` へ入れること（MUST）、その値を `scope` から推定すること（SHOULD）を求める。
本項目はこの推定を採り、`account:` で始まるスコープをレルムの IdMagic API の資源と見なす。
レルムの IdMagic API の識別子は、そのレルムの発行者識別子（`tenancy.Issuer`）とする。
管理発行の API トークンは、既に同じ値を `aud` に固定している（`RFC9700-API-TOKEN-AUDIENCE`）。

### 発行

| 項目 | 内容 |
| --- | --- |
| 判定 | `spec.IsAccountScope(scope string) bool` が、スコープが account リソースサーバーの操作スコープ（`account:` 接頭辞）であるかを返す。`token_handler.go` の `containsAccountScope` もこれへ寄せる |
| デフォルトの audience | `ports.AccessTokenInput.Audiences` が空のとき、`tokens_jose.JWTSigner.SignAccessToken` は `Scopes` に account スコープがあればレルムの発行者識別子を、なければ `client_id` を `aud` にする |
| 置き場所の理由 | デフォルトの audience を決めているのは署名器の 1 か所だけであり、認可コード、リフレッシュ、デバイス認可、CIBA の各発行経路は `Audiences` を空で渡す。各経路へ推定を足すと、1 経路の配線漏れがそのまま観測できない欠陥になる |
| `resource` 指定時 | `Audiences` が埋まるので推定は働かない。`ResolveResourceIndicator` の登録済みかつ Active の検査は変えず、レルム自身を resource indicator の解決先に加えない。未登録の `resource` を通す経路は生じない |

### 検証

`support_http.Authenticator.resolveAuthnContext` は、Bearer のスコープに account スコープがあるとき、`aud` がリクエスト先レルムの発行者識別子を含まなければ `InvalidTokenError`（401 `invalid_token`）で拒否する。
レルムの発行者識別子を解決できない場合も拒否する。
ポータル境界のスコープ（`idmagic.account`、`idmagic.admin`）だけを持つトークンは対象にしない。
ポータルのトークンの audience は別の判断であり、Out of Scope に置く。

`aud` の不一致は、RFC 6750 §3.1 の `invalid_token`（そのほかの理由で無効）に当たる。
スコープ不足ではないため、403 `insufficient_scope` にはしない。

### 規範

- `EX-OAUTH2-001-01` の audience を「レルムの IdMagic API（レルムの発行者識別子）」と具体化する。
- `EX-OAUTH2-001-04` を追加する。audience がレルムの IdMagic API を含まない account スコープのトークンを、account リソースサーバーが `InvalidAccessTokenError` で拒否する。
- `standards.md` へ `RFC9068-DEFAULT-AUDIENCE` を追加し、`RFC8707-MCP-RESOURCE-BINDING` の「`resource` が未指定であれば `client_id` を `audience` とする」をこの行への参照へ置き換える。
- 用語集の Audience と `docs/design/security/authorization.md` に、account スコープの audience の規則を書く。

### 障害モデル

| 障害 | 検出するテスト |
| --- | --- |
| 署名器が account スコープでも `client_id` を `aud` にする | `TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts`（`aud` の値を読む） |
| 検証を外し、`aud=client_id` の account スコープのトークンを受ける | `TestAccountApiRefusesAccountScopedTokenForAnotherAudience`（HTTP 境界で 401 を読む） |
| 検証を account 以外のトークンへ広げ、ポータルのトークンを拒否する | `support_http` の単体テスト（`idmagic.account` だけのトークンが通ること） |
| 検証が別レルムの発行者識別子を受ける | `support_http` の単体テスト（別レルムの `aud` を拒否すること） |

## Tasks

- [x] T001 [Refactor] account スコープの判定を `spec.IsAccountScope` へ抽出し、`token_handler.go` の判定を置き換える（構造変更として先にコミットする）。
- [x] T002 [Spec] `EX-OAUTH2-001-01` と `EX-OAUTH2-001-04`、`RFC9068-DEFAULT-AUDIENCE`、`RFC8707-MCP-RESOURCE-BINDING`、用語集、`authorization.md` を更新し、`mise run check-spec` を通す。
- [x] T003 [Acceptance] 台帳から `EX-OAUTH2-001-01` を外し、`check-spec` の RED と、`aud` を読む HTTP テストの RED を確かめる。
- [x] T004 [Infrastructure] 署名器のデフォルトの audience を scope から推定する。
- [x] T005 [Adapters] Authenticator が account スコープのトークンの audience を検証する。単体 RED から実装する。
- [x] T006 [Verify] 故障注入、`mise run verify`、`mise run test-ui-e2e` を通す。

実行する検査の手順は次のとおりである。

- RED、GREEN、故障注入: `mise run test-go-test -- <package> <test>`
- 振る舞いが GREEN になった時点: `mise run lint-go`、`mise run test-go-changed`

## Verification

- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run check-spec` が `EX-OAUTH2-001-01` を台帳から外した状態で通る。
- `mise run verify`

## Risk Notes

- **発行だけを直して、検証を足さない。** audience を名乗らせても誰も読まなければ、宣言が 1 つ増えるだけで境界は動かない。発行と提示の両方を観測する。
- **audience の検証を足して、既存のトークンを一斉に拒否する。** account リソースサーバーは現在 `aud=client_id` のトークンを受けている。検証を入れるなら、どの値を受けるかを先に決める。
- **resource indicator の解決先にレルムを足すと、既存の拒否が緩む。** `ResolveResourceIndicator` は登録済みかつ Active の資源だけを通す設計で、そこへ例外を作ると未登録の resource を通す経路になりうる。例外の形を Design に書く。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` が示す規範の差分は、REQ-OAUTH2-001 の具体例、追加した `RFC9068-DEFAULT-AUDIENCE`、
  変更した `RFC8707-MCP-RESOURCE-BINDING` である。
  RFC 9068 §3 に従い、`resource` を含まない要求では `scope` からデフォルトの資源を推定して `aud` に入れることにした。
  `account:` で始まるスコープを含むトークンは、レルムの IdMagic API（レルムの発行者識別子）を `aud` に持つ。
  含まないトークンは従来どおり `client_id` を持つ。
  推定は、デフォルトの audience を決める唯一の場所である `tokens_jose.JWTSigner.SignAccessToken` に置いた。
  認可コード、リフレッシュ、デバイス認可、CIBA の各発行経路は変えていない。
  account リソースサーバー（`support_http.Authenticator`）は、account スコープを持つトークンの `aud` がリクエスト先
  レルムの発行者識別子を含まなければ 401 `invalid_token` で拒否する。ポータル境界のスコープだけを持つトークンは検査しない。
  `EX-OAUTH2-001-01` の audience を「レルムの IdMagic API（レルムの発行者識別子）」と具体化し、検証の振る舞いを
  `EX-OAUTH2-001-04` として追加した。`RFC8707-MCP-RESOURCE-BINDING` の「`resource` が未指定であれば `client_id` を
  `audience` とする」は、`RFC9068-DEFAULT-AUDIENCE` への参照に置き換えた。用語集の Audience と
  `docs/design/security/authorization.md` にも同じ規則を書いた。
  構造変更として、account スコープの判定を `spec.IsAccountScope` へ抽出し、`token_handler.go` の
  `containsAccountScope` を置き換えた。
  `EX-OAUTH2-001-01` は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-OAUTH2-001
  - **Observed Failure**: 仕様を更新し、`EX-OAUTH2-001-01` を台帳から外した状態で exit 1。
    `RFC9068-DEFAULT-AUDIENCE`、`EX-OAUTH2-001-01`、`EX-OAUTH2-001-04` の 3 件について `is declared, but no test names it.` を報告した。
  - **Detection Reason**: 宣言した id を名指すテストがない限り必ず落ちるため、テストを書かずに規範だけを進められない。
- **E2E RED Evidence**:
  - **Test**: `TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts`、
    `TestAccountApiRefusesAccountScopedTokenForAnotherAudience`（`backend/oauth2/handlers_http/account_scope_examples_test.go`）
  - **Requirement**: REQ-OAUTH2-001
  - **Observed Failure**: 実装前に、前者は `aud="web-app", want "https://idp.example/realms/default"` で、後者は
    `aud が client_id: status=200, want 401` と `aud が別レルムの発行者識別子: status=200, want 401` で失敗した。
    後者の対照（レルムの IdMagic API を名乗るトークンと `idmagic.account` だけのトークンが 200）は実装前から通っていた。
  - **Detection Reason**: 認可コード + PKCE の正式な入口で得たトークンの `aud` を読み、同じ署名器とレルムの文脈で
    作った別 audience のトークンを account API へ提示して状態行を読む。
- **Unit RED Evidence**:
  - **Test**: `TestAccessTokenWithoutResourceInfersAudienceFromScope`
    (`backend/shared/security/tokens_jose/jwt_signer_audience_test.go`)、
    `TestAccountScopedTokenMustNameTheRealmApi` (`backend/shared/http/support_http/auth_test.go`)
  - **Requirement**: REQ-OAUTH2-001
  - **Observed Failure**: 前者は account スコープの 2 例で `aud="c1", want "https://idp.test/realms/acme"`。後者は
    `client_id`、別レルム、audience なし、レルム未解決の 4 例で `want InvalidTokenError` に対し受理された。
  - **Detection Reason**: 前者は署名器が scope から推定した `aud` を、`resource` 指定時に推定が働かないことと合わせて読む。
    後者は Authenticator の判定を、受理する 3 例（レルムの IdMagic API、複数 audience の 1 つ、ポータル境界のスコープだけ）
    と拒否する 4 例で読む。
- **Primary Use Case Evidence**:
  - id: account-token-names-realm-api
    unit_red: '実装前の `TestAccessTokenWithoutResourceInfersAudienceFromScope` は、account スコープの 2 例で `aud="c1", want "https://idp.test/realms/acme"` として失敗した。'
    e2e_red: '実装前の `TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts` は、認可コードの交換で得たトークンについて `aud="web-app", want "https://idp.example/realms/default"` として失敗した。'
    unit_fault_injection: '署名器の推定の case を無効にすると、単体テストが `aud="c1"` を観測して失敗した。'
    e2e_fault_injection: '`exchange_code.go` が `Audiences` に `client_id` を明示して推定を迂回すると、E2E テストが `aud="web-app"` を観測して失敗した。'
  - id: account-api-refuses-foreign-audience
    unit_red: '実装前の `TestAccountScopedTokenMustNameTheRealmApi` は、`client_id`、別レルム、audience なし、レルム未解決の 4 例で InvalidTokenError ではなく受理を観測して失敗した。'
    e2e_red: '実装前の `TestAccountApiRefusesAccountScopedTokenForAnotherAudience` は、`aud` が `client_id` と別レルムの発行者識別子のトークンについて 401 ではなく 200 を観測して失敗した。'
    unit_fault_injection: 'レルムを解決できないときに受理するよう判定を差し替えると、単体テストの「リクエスト先レルムを解決できない」が失敗した。'
    e2e_fault_injection: 'Authenticator から検証の呼び出しを外すと、E2E テストが `aud が client_id: status=200, want 401` で失敗した。'
- **Change-Resistance Results**:
  1. 署名器の推定の case を無効にすると、単体テストと E2E テストの両方が `aud="c1"` と `aud="web-app"` を観測して失敗した。
  2. `exchange_code.go` が `Audiences` に `client_id` を明示して推定を迂回すると、E2E テストが `aud="web-app"` で失敗した。
  3. Authenticator の検証の呼び出しを外すと、単体テストの 4 例と E2E テストの 2 例が受理を観測して失敗した。
  4. レルムを解決できないときに受理すると、単体テストの「リクエスト先レルムを解決できない」が失敗した。
  5. 検証を account スコープ以外のトークンへ広げると、単体テストの「ポータル境界のスコープだけ」と E2E テストの対照が
     401 を観測して失敗した。
  いずれの差し替えも復元済みである。
  `mise run test-go-mutation` は `backend/shared/security/tokens_jose`（Killed 169、Lived 31）と
  `backend/shared/http/support_http`（Killed 303、Lived 65）で実行した。今回変更した行の変異は、`jwt_signer.go` 81 行目の
  `len(in.Audiences) == 0` と `auth.go` 218 行目の `realmAPI != ""` がどちらも殺された。`jwt_signer.go` 85 行目の
  `> 1` から `>= 1` への変異は生き残ったが、既存の行であり、直前の `== 1` の case が先に当たるため振る舞いが変わらない
  等価変異である。`!` の反転は変異器の演算子に含まれないため、上記 3 と 5 の手動注入で確かめた。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-repository` - passed
  - `mise run lint-go` - 0 issues
  - `mise run test-go-changed` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed (exit 0)
  - `mise run test-ui-e2e` - passed
