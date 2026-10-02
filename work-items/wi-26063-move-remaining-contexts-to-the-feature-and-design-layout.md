---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-21670-move-identity-management-to-the-feature-and-design-layout, wi-39141-move-tenancy-to-the-feature-and-design-layout]
change_kind: docs
spec_impact:
  kind: none
  reason: "残りの Context の文書の置き場所と書式だけを変える。各 Context の REQ と EX の ID、例のステップ、規則が定める値、状態遷移の表、標準仕様の表は変えない。"
---

# 残りの Context の仕様と設計を、機能仕様と内部設計の構造へ移す

## 動機

wi-43431 は、新旧の二つの形式を Context ごとに切り替える仕組みを入れる。
IdManagement（wi-21670）と Tenancy（wi-39141）を移した後も、19 の Context が旧形式のまま `tools/check/legacy-spec-layout.json` に残る。
二つの形式が並ぶ間は、読み手は Context ごとに読み方を変えなければならない。
検査と生成器も、二つの経路を保守し続けることになる。

## 対象範囲

- `legacy-spec-layout.json` に残る Context を、wi-21670 と wi-39141 で確定した手順で移す。
- すべての Context を移した後、旧形式の経路を検査と生成器から取り除き、`legacy-spec-layout.json` を削除する。

## 対象外

- 規則の内容の変更。

## 設計

移行の手順は、wi-21670 と wi-39141 の完了時点の手順に従う。
一つの作業で移す Context の数は、着手時に決める。
Context の数が多いので、この項目を複数の work item に分けて進めてもよい。

## 計画

着手時に、残った Context の一覧と規則の数から、分割と順序を決める。

## タスク

- [ ] T001 [Spec] 残りの Context を移す。
- [ ] T002 [App] 旧形式の経路と `legacy-spec-layout.json` を取り除く。
- [ ] T003 [Verify] 検査と spec-diff で、規則が変わっていないことを確かめる。

## 検証

- `mise run check`
- `mise run spec-diff` で、REQ と EX の削除と変更が 0 件であることを確かめる。

## リスク

旧形式の経路を取り除く前に移行し漏れた Context があると、検査が失敗する。
経路を取り除くのは、`legacy-spec-layout.json` が空になった後に限る。
