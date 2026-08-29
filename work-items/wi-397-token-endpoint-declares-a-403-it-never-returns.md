---
depends_on: []
status: in_progress
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/contexts/oauth2/standards.md]
  typespec:
    - IdMagic.OAuth2.Operations.Token
    - IdMagic.OAuth2.Operations.Authorize
  source:
    - backend/oauth2/handlers_http/errors.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/handlers_http/authorize_handler.go
    - backend/oauth2/handlers_http/authorize_login.go
  tests:
    - backend/oauth2/handlers_http
  stop_before_reading: [frontend, backend/sourcing, backend/idmanagement]
affected_spec:
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.Token }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.Authorize }
---

# Token endpoint が返さない 403 を契約が宣言している

## Motivation

[[wi-391-refusal-declaration-floor-and-reinventory]] の R4 が、契約とシナリオを突き合わせて見つけた食い違いである。

`spec/contexts/oauth2/main.tsp` の `Token` は `TokenError403` として `OAuthAccessDeniedError` を宣言している。しかし実装の `writeOAuthError` (`backend/oauth2/handlers_http/errors.go`) が状態コードを変えるのは `invalid_client` → 401 と `server_error` → 500 だけで、`access_denied` を含む残りはすべて 400 で返る。承認要求を利用者が拒否した場合 (CIBA) も、デバイス認可を拒否した場合も、返るのは 400 の `{"error":"access_denied"}` である。**`Token` が 403 を返す経路は存在しない。**

RFC 6749 §5.2 はトークンエンドポイントのエラー応答を 400 と定めており (`invalid_client` のみ 401 を許す)、RFC 8628 §3.5 の `access_denied` もこれに従う。**実装が標準どおりで、契約が誤っている**というのが現時点の判断である。

契約が返らない応答を宣言していると、クライアント実装は起こらない分岐を書き、生成された OpenAPI を読む相手は誤った期待を持つ。

## Scope

- `Token` の 403 宣言を実装と標準に合わせる。`OAuthAccessDeniedError` を 400 の本文へ移すか、`Token` から取り除くかを決める。
- 同じ宣言を持つ `Authorize` の 403 も、実際に返る経路があるかを確かめる。
- OpenAPI ベースラインの更新と、互換性検査の判断を記録する。

## Out of Scope

- `access_denied` を実際に 403 で返すよう実装を変えること。標準に反する。
- 管理 API の `AccessDeniedError` (Problem Details) の扱い。こちらは 403 で正しい。

## Design

未定。着手時に `Authorize` の実測と、ベースライン更新の影響を確かめてから決める。

## Plan

- 実装が返す状態コードを経路ごとに実測してから契約を直す。契約を先に直すと、別の経路が 403 を返していた場合に取り違える。

## Tasks

- [ ] T001 [Survey] `Token` と `Authorize` の各エラー経路が実際に返す状態コードを実測する。
- [ ] T002 [Spec] 契約の 403 宣言を実測に合わせる。
- [ ] T003 [Verify] `mise run check-api-compat` とベースラインの更新を確認する。

## Verification

- `mise run verify`
- `mise run check-api-compat`

## Risk Notes

リスクは low。契約の宣言を実測に合わせるだけで、実装の振る舞いは変えない。ただし公開済みの OpenAPI から応答が 1 つ消えるため、互換性検査が破壊的変更として扱う可能性がある。返らない応答の削除をどう扱うかを、ベースライン更新の判断として記録する。
