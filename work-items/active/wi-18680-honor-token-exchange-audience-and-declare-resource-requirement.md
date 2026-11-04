---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8707-MCP-RESOURCE-BINDING }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-SUBJECT-TOKEN }
---

# トークン交換の `audience` を捨てず、`resource` の必須を宣言と一致させる

## 動機

RFC 8693 §2.1 は、交換の要求の `resource` と `audience` をどちらも OPTIONAL とする。
`ExchangeToken` は `resource` を必須にし、`audience` を読まずに捨てる。
`audience` を付けたクライアントは、指定と異なる対象のトークンを受け取るか、`resource` の欠落で拒否される。

`resource` の必須は `token.md` の設計（「交換時には、結果を単一の対象へ狭める `resource` を必須とする」）が意図した選択である。
一方 `RFC8707-MCP-RESOURCE-BINDING` は、トークン交換を含む全経路で、未指定時の `audience` を `RFC9068-DEFAULT-AUDIENCE` に委ねると宣言している。
宣言どうしが食い違い、受け入れの行もない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `audience` を付けた交換の要求の扱いを決めて実装する。候補は、登録済みのリソースの名前として解決するか、対応しないことを `invalid_target` で返すかである。読まずに捨てる現在の振る舞いはどちらでもない。
- `resource` の必須を保つなら、`standards.md` に `partial` の行として宣言し、`RFC8707-MCP-RESOURCE-BINDING` から交換の未指定時の扱いを外す。必須をやめるなら、未指定の交換で既定の `audience` を使う。
- 選んだ形の要求を `/token` へ送る受け入れ試験を加える。

## 対象外

- ID-JAG の発行の `audience`。`IDJAG-ISSUANCE` が §4.3 の割り当てに従っている。

## タスク

- [ ] T001 [Spec] `resource` と `audience` の扱いを決め、宣言の行と要件を更新する。
- [ ] T002 [Acceptance] `audience` を付けた交換の要求で RED を確かめる。
- [ ] T003 [App] 決めた扱いを実装し、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `/token` のトークン交換の HTTP の受け入れ試験：応答の誤りか、発行したトークンの `aud` を観測するため。

## リスク

`audience` を解決する場合、`resource` との併用と複数の値の扱いを決めないと、対象を広げる経路になる。
