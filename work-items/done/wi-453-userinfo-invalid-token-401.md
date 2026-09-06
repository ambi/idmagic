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
  reason: "`/userinfo` の invalid_token 応答から誤った 400 を除去し、規格どおりの 401 と Bearer challenge に変えるため、旧ステータスへ依存する呼び出し側には移行が必要になる。"
  references:
    - { kind: release_note, path: docs/releases/changes/wi-453-userinfo-invalid-token-401.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-453-userinfo-invalid-token-401.md }
initial_context:
  specification:
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-020
    - docs/contexts/oauth2/standards.md#RFC6750-INVALID-TOKEN
    - docs/api-rules.md#http-error-responses
  typespec:
    - IdMagic.OAuth2.Operations.UserInfo
    - IdMagic.OAuth2.Operations.PostUserInfo
  source:
    - backend/oauth2/handlers_http/userinfo_handler.go
    - backend/oauth2/handlers_http/errors.go
    - backend/oauth2/token/usecases/userinfo.go
    - backend/shared/http/support_http/auth.go
  tests:
    - backend/oauth2/handlers_http/errors_test.go
    - backend/oauth2/handlers_http/userinfo_handler_test.go
  stop_before_reading:
    - frontend
    - backend/saml
primary_use_cases:
  - id: userinfo-invalid-token
    requirement: RFC6750-INVALID-TOKEN
    observable_result: GET と POST の UserInfo が invalid_token を 401 と Bearer challenge で返し、ユーザークレームを返さない。
    unit_test: { path: backend/oauth2/handlers_http/errors_test.go, name: TestWriteUserInfoErrorMapsInvalidTokenToBearerChallenge, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/userinfo_handler_test.go, name: TestUserInfoRejectsRevokedAccessTokenWithBearerChallenge, task: test-go-race }
    unit_fault_model: UserInfo 固有の写像が invalid_token を OAuth 認可サーバー用の汎用 400 写像へ渡す。
    e2e_fault_model: UserInfo ハンドラーが固有の写像を迂回して writeOAuthError を直接呼ぶ。
affected_spec:
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-020 }
  - { path: docs/contexts/oauth2/standards.md, requirement: RFC6750-INVALID-TOKEN }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.UserInfo }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.PostUserInfo }
---

# `/userinfo` の `invalid_token` を RFC 6750 のとおり 401 と `WWW-Authenticate` で返す

## Motivation

`wi-386` の総なめで、契約と実装のどちらが正しいかが実装側にある例が 1 件残った。

`/userinfo` は `UserInfoError401` を宣言し、その本体は `invalid_token` である。しかし handler は失敗をすべて `writeOAuthError` に渡し、`writeOAuthError` は `invalid_client` と `server_error` 以外をすべて 400 にする。したがって `/userinfo` に無効な token を出すと 400 が返り、`WWW-Authenticate` も付かない。

RFC 6750 §3.1 は、保護資源が `invalid_token` を返すときは 401 と `WWW-Authenticate` challenge を要求する。契約が正しく、サーバーが規格に従っていない。同じ判断は `PostUserInfo` にも及ぶ。

`wi-386` はこれを検出できていない。`writeOAuthError` はエラー値から応答を決める写像なので、`wi-386` の検査は辿らず、`/userinfo` を「読み残しあり」として数えている。人手で handler を読んで確かめた 1 件である。

## Scope

- `/userinfo` と `/userinfo` (POST) が `invalid_token` を 401 で返し、`support_http.SetBearerChallenge` と同じ形の `WWW-Authenticate` を付ける。
- 同じ判断が及ぶ他の保護資源接点 (Bearer token を検証して `invalid_token` を返す経路) を数え上げ、同じ形に揃える。
- 拒否が「何を残さなかったか」まで見る検査を置く。無効な token で `/userinfo` を叩いたとき、claim が 1 つも漏れていないことを確かめる。

## Out of Scope

- `writeOAuthError` の写像表そのものの作り直し。`/authorize` や `/token` は RFC 6749 §5.2 の側にあり、400 が正しい。接点ごとに規格が違うので、共通関数を一律に変えない。

## Design

`writeUserInfoError(c *echo.Context, err error) error` を UserInfo の保護リソース境界に置き、`OAuthError.Code` が `invalid_token` なら `support_http.SetBearerChallenge` と 401 の OAuth 本文を返し、`insufficient_scope` なら同じ challenge と 403 を返す。それ以外だけを既存の `writeOAuthError` へ委ねる。時刻、永続化、乱数、通知の効果は変更せず、入力はエラー値、出力は HTTP 応答への効果として境界に現れる。

## Plan

1. RFC 6750 の採用行、UserInfo シナリオ、401 response header を仕様へ先に追加する。
2. 専用写像の Unit RED と、実ルートを通る GET/POST の E2E RED を観測する。
3. ハンドラーの全失敗経路を専用写像へ集約し、狭いテストから全体検証へ広げる。

## Tasks

- [x] T001 [Spec] RFC6750-INVALID-TOKEN、REQ-OAUTH2-020、UserInfo の 401 header 契約を更新し、`mise run check-spec` を通す。
- [x] T002 [Acceptance] `TestUserInfoRejectsRevokedAccessTokenWithBearerChallenge` が GET/POST とも status=400 で落ちる E2E RED を確認する (RFC6750-INVALID-TOKEN / REQ-OAUTH2-020)。
- [x] T003 [Adapter] `TestWriteUserInfoErrorMapsInvalidTokenToBearerChallenge` が `writeUserInfoError` 未定義で落ちる Unit RED を確認する (RFC6750-INVALID-TOKEN / REQ-OAUTH2-020)。
- [x] T004 [Adapter] UserInfo 固有のエラー写像を実装し、全失敗経路から使う。
- [x] T005 [Verify] 狭いテスト、障害注入、`mise run verify` を通す。

## Verification

- `mise run check-spec`
- `mise run verify`
- `mise run check-status-drift`

## Risk Notes

400 から 401 への変更は、`/userinfo` の失敗を分岐している既存クライアントを壊しうる。壊れる側は規格に反した実装に依存していたことになるが、変更は 1 接点ずつ行い、監査ログで実際の呼び出し側を確かめてから広げる。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `RFC6750-INVALID-TOKEN` の追加と `REQ-OAUTH2-020` の変更を報告した。GET/POST の UserInfo は、失効済みアクセストークンを `401 invalid_token` と Bearer challenge で拒否し、クレームを返さない。`insufficient_scope` も同じ保護資源境界で 403 と challenge に写像し、認可サーバー用の汎用 400 写像とは分離した。
- **Primary Use Case Evidence**:
  - id: userinfo-invalid-token
    unit_red: >-
      `TestWriteUserInfoErrorMapsInvalidTokenToBearerChallenge` は専用写像の実装前に `writeUserInfoError` 未定義で失敗した。
    e2e_red: >-
      `TestUserInfoRejectsRevokedAccessTokenWithBearerChallenge` は GET/POST とも実応答が 400 だったため、期待する 401 と一致せず失敗した。
    unit_fault_injection: >-
      `invalid_token` の写像を 400 に戻すと単体テストが `status=400, want 401` で失敗した。
    e2e_fault_injection: >-
      失効トークン経路を `writeOAuthError` へ迂回させると GET/POST の受け入れテストがともに 400 を検出した。
- **Change-Resistance Results**:
  専用写像の状態コードを 400 に戻す故障と、ハンドラーから専用写像を迂回する故障を別々に注入し、単体境界と実ルート境界がそれぞれ検出することを確認した。いずれも復元後に狭いテストを再実行して成功した。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-status-drift` - passed (0 findings)
  - `mise run verify` - passed
