---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-05
priority: p3
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "外部 OIDC の受け入れの制限を標準要件として宣言する作業であり、製品の振る舞いと既存の規範の結果は変えない。" }
---

# 外部 OIDC の ID Token とトークン応答の受け入れの制限を宣言する

## 動機

IdMagic は外部 OIDC の RP として、上流の ID Token を RS256 でだけ検証し、トークン応答からは `id_token` だけを読む。
OpenID Connect Core は RS256 を既定とするが、OP は登録した `id_token_signed_response_alg` で ES256 などを使える。
`OIDC-CORE-CODE-FLOW` は検証の項目を並べるが、アルゴリズムの制限も受け入れる応答の形も書いていない。
制限が宣言にないため、ES256 の OP と接続できないことを利用者が事前に知る手段がない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- RS256 だけを受け入れる制限を `partial` の標準要件として宣言する。
- トークン応答の `token_type`、`expires_in`、`refresh_token` を読まずに受け入れることを、受け入れの標準要件として宣言する。
- 宣言した各標準要件を参照する試験を確かめ、なければ加える。

## 対象外

- RS256 以外のアルゴリズムへの対応。必要になれば feature として起票する。
- `kid` の省略。[[wi-41129-accept-upstream-id-tokens-without-kid-for-a-single-key]] が扱う。

## タスク

- [ ] T001 [Spec] 標準要件を加える。
- [ ] T002 [Test] 標準要件を参照する試験を確かめ、なければ加える。
- [ ] T003 [Verify] `mise run check-spec` と変更した試験を通す。

## 検証

- `mise run check-spec`：加えた標準要件の形式と参照を確かめるため。

## リスク

小さい。
