---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-25
priority: p2
group: development-process
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "作業項目の形式の文書だけを変え、製品の規範仕様、TypeSpec、検査器の判定は変えない。" }
---

# 主要ユースケースの証拠を、形式の文書だけで検査を通るように書けるようにする

## 動機

`docs/formats/work-item-format.md` の「基本形式」のテンプレートは `primary_use_cases` の例を持つが、検査器が要求する次の二点を示さない（2026-11-25 に確認）。

- `setup` の `state`、`entry_points`、`fixtures`。テンプレートの例はこれらを持たず、`tools/check/src/primary-use-case-evidence.ts` は一つでも空なら拒否する。「主要ユースケースの証拠」の表はフィールド名を挙げるが、YAML での書き方は示さない。
- `test.task` に書ける値。表は「必要な `mise` または CI の `task`」とだけ書く。検査器（`tools/check/src/check-work-items.ts` の `requiredVerificationTasks`）が受け入れるのは、`mise.toml` の `verify` の依存の閉包に入るタスクか、`.github/workflows` が `mise run <task>` で呼ぶタスクだけである。`test-go-one` や `test-go-package` のような局所検査のタスクは拒否される。

「主要ユースケースの証拠」の節からテンプレートの例への案内もない。
[[wi-62564-accept-upstream-jwks-without-alg]] の着手時に、この文書だけでは検査を通る記入を決められず、検査器のテストの fixture（`tools/check/src/repository-checks.acceptance.test.ts`）を探した。

## 対象範囲

- 「基本形式」のテンプレートの `primary_use_cases` の例に `setup` を足す。
- 「主要ユースケースの証拠」の節に、`test.task` が受け入れられる条件と、Go の受け入れ試験と単体試験では `test-go` を書くことを一文で明記する。
- 同じ節から、テンプレートの例へリンクする。

## 対象外

- 検査器の判定の変更。受け入れるタスクの集合は今のままにする。
- `maturity_evidence` など、ほかのフィールドの例の見直し。

## 設計

例は「基本形式」のテンプレートの一か所だけに置き、「主要ユースケースの証拠」の節からはリンクする。
同じ例を二か所に書くと、片方だけが古くなるためである。

採らない案：検査器の失敗の出力に記入例を含める。
記入する前に読む文書で分かるほうが往復が少なく、出力はすでに欠けたフィールドを名指している。

## 計画

1. テンプレートの例に `setup` を足す。
2. 「主要ユースケースの証拠」の節に `task` の条件とテンプレートへのリンクを足す。
3. テンプレートの例を写した作業項目が `check-work-items` を通ることを確かめる。

## タスク

- [ ] T001 [Docs] テンプレートの `primary_use_cases` の例に `setup` を足す。
- [ ] T002 [Docs] 「主要ユースケースの証拠」の節に `test.task` の条件とテンプレートへのリンクを足す。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run check-links`：節へのリンクが解決することを確かめるため。
- `mise run check-work-items`：テンプレートの例を写した作業項目が形式の検査を通ることを確かめるため。

## リスク

文書が検査器の判定と再びずれる懸念がある。
受け入れるタスクの条件は検査器の実装を名指して書き、変更時にどちらを直すかを分かるようにする。
