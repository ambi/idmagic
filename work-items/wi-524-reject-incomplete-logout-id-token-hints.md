---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-09
priority: p1
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-024 }
  - { path: docs/contexts/oauth2/standards.md, requirement: OIDC-LOGOUT-ID-TOKEN-HINT }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.Contract.EndSession1 }
---

# 不完全な `id_token_hint` によるログアウト対象の誤解決を拒否する

## Motivation

`VerifyIDTokenHint` は署名と発行者を検証して `aud`、`sub`、`sid` を抽出するが、これらが空でも成功を返す。
`ResolveEndSession` も空の `sub` と `sid` を拒否せず、`sub` と `sid` が示す LoginSession の主体を照合しない。

このため、署名済みでも必須 claim が欠けた ID Token をログアウト対象の根拠として受理する。
`sid` が空なら HTTP 層はブラウザー Cookie のセッションへフォールバックするため、ヒントが示す対象とは別のローカルセッションを失効しうる。

## Scope

- `id_token_hint` の署名、発行者、audience、subject、`sid` を一組の検証結果として扱う。
- `aud`、`sub`、`sid` のいずれかが空なら `invalid_request` として fail-closed で拒否する。
- `sid` が示す LoginSession の User ID と `sub` が一致する場合だけローカル失効へ進む。
- 拒否時に LoginSession と RefreshTokenRecord が失効しないことを HTTP 経路から観測する。

## Out of Scope

- 期限切れの ID Token を `id_token_hint` として許容する既存方針の変更。
- front-channel と back-channel の通知。
- CIBA が使う `id_token_hint` の有効期限規則。

## Design

`IDTokenHintClaims` は検証済み claim の値を運ぶだけであり、空値を有効とする型ではない。
暗号アダプターは署名と発行者の検証に加えて `aud`、`sub`、`sid` の存在と非空を検証し、不完全な token を use case へ渡さない。

HTTP 層は hint から得た `sid` の LoginSession をローカル失効前に読み、`sub` と User ID を照合する。
一致しない場合は応答だけを拒否して処理を続ける形を防ぐため、セッションと同じ `sid` の RefreshTokenRecord が残っていることもテストする。

## Plan

1. claim 欠落と主体不一致を `/end_session` から送るテストを先に追加し、誤った失効または受理を観測する。
2. 暗号アダプターで必須 claim を検証し、ローカル失効前に主体を照合する。
3. 狭いパッケージテストから変更パッケージ、最終検証へ広げる。

## Tasks

- [ ] T001 [Spec] 現行仕様が必須 claim と対象主体の照合を十分に表しているか確認し、不足があれば仕様を先に更新する。
- [ ] T002 [Acceptance] `/end_session` へ不完全または主体が矛盾する `id_token_hint` を送り、拒否と効果の不在について RED を確認する。
- [ ] T003 [Adapter] `VerifyIDTokenHint` が空の `aud`、`sub`、`sid` を拒否する Unit RED を GREEN にする。
- [ ] T004 [Use Case] hint の `sub` と LoginSession の User ID を照合し、不一致を fail-closed で拒否する。
- [ ] T005 [Resistance] 必須 claim の検査と主体照合をそれぞれ外し、対応するテストが落ちることを確認する。
- [ ] T006 [Verify] `mise run verify`。

## Verification

- `mise run test-go-package -- ./backend/shared/security/tokens_jose`
- `mise run test-go-package -- ./backend/oauth2/token/usecases`
- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run test-go-changed`
- `mise run verify`

## Risk Notes

- 署名検証だけでは token 内の claim 同士と保存済みセッションの整合性を保証しない。
- 拒否応答だけを検証すると、拒否後にローカル失効を続ける欠陥を見逃す。
- CIBA とログアウトは同じ verifier を使うため、共通層の厳格化が CIBA の既存入力へ及ぼす影響を狭いテストで確認する。
