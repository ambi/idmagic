---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/authorization/README.md, requirement: REQ-OAUTH2-005 }
---

# PKCE をすべてのクライアントに求めるか、クライアントごとに定めるかを決めて揃える

## 動機

認可のモデルと判断（PKCE の必須化を公開クライアントと FAPI のクライアントに限る）は、PKCE の要否をクライアントの `require_pkce` で定めるとしている。
実装の `/authorize` は、すべてのクライアントに `S256` の `code_challenge` を求める。
REQ-OAUTH2-005 は実装に合わせてすべてのクライアントに求めると書いたので、モデルと判断が要件と食い違っている。

## 対象範囲

- すべてのクライアントに求めるか、`require_pkce` で定めるかを決める。
- 決めた方に、モデル、判断、要件、実装のどれかを揃える。

## 対象外

- `code_challenge_method` に `S256` 以外を許すこと。

## 設計

すべてに求める方は、RFC 9700 の推奨に沿い、実装を変えずに済む。その場合は判断の文書とモデルを改める。
クライアントごとに定める方は、confidential のクライアントの互換を保てるが、要件と実装の両方を変える。
推奨は、すべてに求める方である。

## タスク

- [ ] T001 [Spec] 方針を決め、判断とモデルと要件を揃える。
- [ ] T002 [App] 実装を変える方を選んだ場合は RED と GREEN を確認する。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run verify`

## リスク

- confidential のクライアントで PKCE を使っていない既存の連携がある場合、すべてに求める方は互換を壊す。導入状況を確かめてから決める。
