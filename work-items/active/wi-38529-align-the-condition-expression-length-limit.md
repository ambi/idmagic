---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-14
priority: p3
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-064 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-090 }
  - { path: spec/modules/oauth2/models.tsp, symbol: ResourceAccessCondition }
---

# 条件の式の長さの上限を、契約、サーバー、画面で一つの数え方にそろえる

## 動機

規則の条件の式（`ResourceAccessCondition` の `deny_when` と `require_approval_when`）の長さの上限は、三か所で数え方が異なる（2026-11-14 に確認）。

| 箇所 | 上限と数え方 |
| --- | --- |
| TypeSpec（`spec/modules/oauth2/models.tsp`） | `@maxLength(1024)`。REQ-SYSTEM-028 に従えばコードポイントで数える |
| REQ-OAUTH2-064 | 「式が 1 KiB を超える」場合に 400 と `invalid_resource_server` |
| サーバー（`backend/oauth2/domain/resource_access_condition.go` の `compileCondition`） | `MaxConditionExpressionBytes = 1024`。UTF-8 のバイト数で数える |

日本語の文字列リテラルを含む式は、契約上は受け付けられる長さでもサーバーに拒否される。
画面（[[wi-45648-edit-resource-access-rules-with-a-form]]）は受理範囲を一つに決められず、式の長さを送信前に検査していない。

## 対象範囲

- 上限の数え方を一つに決め、TypeSpec、REQ-OAUTH2-064、サーバーの検証をそろえる。
- 画面の条件の欄に、同じ数え方の送信前検査を加える（REQ-OAUTH2-090 の送信前の検査）。
- 上限の直前、その値、直後を、ASCII と基本多言語面の外の文字の両方で検査する。

## 対象外

- 上限の値（1024）の変更。
- CEL のコンパイルの費用の上限など、長さ以外の制約。

## 設計

### 候補

| 候補 | 内容 | 評価 |
| --- | --- | --- |
| A：コードポイントにそろえる | サーバーを `utf8.RuneCountInString` で 1024 コードポイントに変え、REQ-OAUTH2-064 の「1 KiB」を「1024 文字」に直す | REQ-SYSTEM-028 と同じ数え方であり、TypeSpec と画面の共通の欄の規則（`codePointLength`）をそのまま使える。最大のバイト数は約 4 KiB になる |
| B：バイトにそろえる | TypeSpec の `@maxLength` を外し、`@doc` でバイトの上限を述べる | 汎用 API の長さの数え方の例外になり、画面に専用の数え方が要る |

A を推奨する。
バイトの上限を残す理由（CEL の解析の費用）は、4 KiB でも既存の内包の入れ子の制限と比べて小さいと見込むが、T001 で確かめる。

### 確かめる問い

- 汎用 API の長さの検証（REQ-SYSTEM-028）が、`access_rules` の入れ子の `condition` にも 422 と `field_length_exceeded` で働くか。働くなら、超過の応答は 400 ではなく 422 になり、REQ-OAUTH2-064 の記述と食い違う。どちらの応答にするかを T001 で決め、要件に書く。

## 計画

T001 で数え方と応答を決めて仕様を直し、サーバーのドメインのテストで境界を RED にしてから実装し、画面の検査を加える。

## タスク

- [ ] T001 [Spec] 数え方と超過時の応答を決め、REQ-OAUTH2-064 と TypeSpec を直す。汎用 API の長さの検証が入れ子へ届くかを確かめる。
- [ ] T002 [Domain] `compileCondition` の上限を決めた数え方にし、境界の両側を ASCII と多バイトの文字で検査する。
- [ ] T003 [UI] 条件の欄に送信前の長さの検査を加え、`accessRuleProblems` の単体テストで境界を確かめる。
- [ ] T004 [Acceptance] 管理 API で上限ちょうどの多バイトの式を保存でき、超えた式を拒否することを確かめる。

## 検証

- `mise run test-go-package -- ./backend/oauth2/domain`
- `mise run test-ui-unit-file -- src/features/admin-mcp-resource-servers/accessRuleProblems.test.ts`
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

A を選ぶと、これまで拒否していた多バイトの長い式を受け付けるようになる。
既存の保存値は以前の上限内なので、読み込みで拒否される規則は生じない。
