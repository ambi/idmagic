---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-23
priority: p3
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 具体例の文言を、既に契約が宣言し製品が返している 401 に合わせる変更であり、製品の振る舞いも公開契約も変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/ws-federation/scenarios.feature.md#REQ-WSFEDERATION-004
  typespec:
    - IdMagic.WsFederation.Operations.WsTrustIssue
  source:
    - backend/wsfederation/handlers_http/wstrust_handler.go
  tests:
    - backend/wsfederation/handlers_http/wstrust_examples_test.go
    - backend/wsfederation/handlers_http/wsfed_handler_test.go
  stop_before_reading: []
affected_spec:
  - { path: docs/domain/ws-federation/scenarios.feature.md, requirement: REQ-WSFEDERATION-004 }
---

# WS-Trust の資格情報の誤りが実際に受ける拒否を、具体例に書く

## Motivation

`docs/domain/ws-federation/scenarios.feature.md` の `EX-WSFEDERATION-004-03` は、UsernameToken の資格情報が不正なとき「`AccessDeniedError` を返しトークンを発行しない」と宣言している。

[[wi-554-back-ws-federation-examples-with-tests]] が測ったところ、製品の `/trust/usernamemixed` は誤ったパスワードにも未知の username にも 401 の平文 `invalid credentials` を返し、`WsTrustTokenRejected` を発行する。応答は RSTR も assertion も運ばず、`WsTrustTokenIssued` も出ない。拒否そのものは効いている。食い違っているのは拒否の型だけである。

契約 `WsTrustIssue` はこの 401 を `WsTrustIssueError401`（`text/plain`）として宣言している。403 `AccessDeniedError` も宣言しているが、どの経路もそれを返さない。

401 の側には意味の理由がある。資格情報の誤りは認証の失敗であり、`AccessDeniedError` が表す「主体が操作のポリシーを満たさない」とは別の状態である。したがってこれは写像の欠落ではなく、規範と実装のどちらを正とするかの判断である。[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] と同じ型の食い違いだが、原因となる設計は別である。

## Scope

- `EX-WSFEDERATION-004-03` の `Then` を実装の拒否と一致させるか、実装を具体例に合わせるかを決める。
- 決めた側へ寄せる。規範を直すなら `spec-change` を通す。
- 契約 `WsTrustIssue` の 403 `AccessDeniedError` を、どの経路も返さない宣言として残すかを同じ判断のなかで決める。
- 決着後、`EX-WSFEDERATION-004-03` を名指しするテストを書き、`tools/check/example-coverage-debt.json` から外す。材料は `backend/wsfederation/handlers_http/wsfed_handler_test.go` の `TestWsTrustUsernameTokenPassword_AuthenticatesTheSuppliedCredential` に揃っている。

## Out of Scope

- WS-Trust の拒否を SOAP fault（`wst:FailedAuthentication`）で返す変更。応答形の全面的な見直しになるので、必要なら別項目とする。
- 400 の応答が契約の `application/problem+json` ではなく平文で返っている件。拒否の型ではなく本文の形の食い違いであり、本項目の判断に依存しない。
- 資格情報の検証そのもの（ログイン失敗の計数、sentinel ハッシュによる時間の均し）。

## Design

選択肢は 2 つある。

| 選択肢 | 得るもの | 失うもの |
| --- | --- | --- |
| 具体例を 401 の平文に合わせる | 認証の失敗と認可の拒否の区別が規範に残る。既存の契約と実装をそのまま使える | 具体例が名指す型が `AccessDeniedError` から変わる |
| 実装を 403 `AccessDeniedError` に合わせる | 具体例の文言を保てる | 認証の失敗を認可の拒否として返すことになる。既存の WSS-UsernameTokenPassword の観測（401）も書き換えが要る |

前者を採る。具体例の `Then` を「401 で拒否しトークンを発行しない」に書き換える。拒否の型は `WsTrustIssueError401` として契約が既に宣言しているので、TypeSpec は変えない。

契約 `WsTrustIssue` の 403 `AccessDeniedError` は、どの経路も返さない宣言だが残す。外すと `mise run check-api-compat` が `POST /trust/usernamemixed 403: response status removed` を破壊的変更として拒否し、通すにはパスの版上げか基準線の書き換えが要る。宣言だけが残っていても利用者の実装は壊れないので、この判断を本項目の規模で覆す理由が無い。

## Plan

1. 選択肢を決め、Design に記録する。
2. 決めた側へ寄せる。規範を変えるなら `spec-change` を通す。
3. `EX-WSFEDERATION-004-03` を名指しするテストを書き、台帳から外す。

## Tasks

- [x] T001 [Docs] 選択肢を決める。
- [x] T002 [Spec] 決めた側へ規範または実装を寄せる。
- [x] T003 [Adapters] `EX-WSFEDERATION-004-03` を名指しするテストを書き、台帳から外す。

## Verification

- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **実装を具体例に寄せるほうが速いので、そちらへ流れる。** 401 を 403 に変えるのは 1 行で済むが、それは認証の失敗を認可の拒否として外へ伝える変更になる。速さを理由に選ばない。
- **拒否が効いていることを、型が合っていないことと混同する。** 測定ではトークンは出ておらず、拒否イベントも出ている。この項目が扱うのは型の食い違いだけである。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` が示す規範の差分は、`EX-WSFEDERATION-004-03` の `Then` が
  「AccessDeniedError を返しトークンを発行しない」から「401 で拒否しトークンを発行しない」へ変わったことだけである。
  具体例が、契約 `WsTrustIssue` の `WsTrustIssueError401` と製品が実際に返す拒否に一致した。
  製品のコードと TypeSpec は変わっていない。この具体例は名指すテストを得て、被覆台帳から外れた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-WSFEDERATION-004
  - **Observed Failure**: `EX-WSFEDERATION-004-03` を台帳から外した状態で exit 1。
    `EX-WSFEDERATION-004-03 is declared, but no test names it.` がその 1 件だけを名指しした。
  - **Detection Reason**: テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestWsTrustIssue_RefusesAWrongCredentialWith401AndNoToken`
    (`backend/wsfederation/handlers_http/wstrust_examples_test.go`)
  - **Requirement**: REQ-WSFEDERATION-004
  - **Observed Failure**: N/A: 製品の振る舞いは変えていないので、実装前に落ちる単体境界は無い。
    代わりに故障を注入して落ちることを確かめた（下の Change-Resistance Results）。
  - **Detection Reason**: 401 の状態コード、応答が RSTR も assertion も運ばず `WsTrustTokenIssued` も出ないこと、
    `WsTrustTokenRejected` が 1 件出ることの 3 つを、誤ったパスワードと未知の username の 2 通りで読む。
    正しい資格情報が同じ入口で発行される対照を置く。
- **Change-Resistance Results**:
  1. 資格情報の誤りへの応答を 401 から 403 に替える → 上のテストが落ちる。
  2. 資格情報の誤りで `WsTrustTokenRejected` を発行しない → 上のテストが落ちる。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - passed（403 を外す案は破壊的変更として落ちたので採らなかった）
  - `mise run verify` - passed
