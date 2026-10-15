---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-16
priority: p2
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "作業の完了で作るコミットの分け方を開発手順の文書と実装の skill に書くだけである。HTTP の応答、永続状態、ドメインイベント、TypeSpec の契約、規範要素は変えない。" }
---

# 変更と縮めた記録を別のコミットにする

## 動機

[チェックポイントと最終コミット](../../docs/development/specification-first-workflow.md#チェックポイントと最終コミット)と `implement-work-item` の手順 7 は、`move-work-item` で記録を縮めてから、チェックポイントを一つの正式コミットへ折り畳む。
縮めた記録は状態、題名、要約だけを持ち、`affected_spec` を持たない。
`check-spec-impact` は縮めた記録の宣言を、[完了](../../docs/formats/work-item-format.md#完了)が定めるとおり全文を最後に含むコミットから読む。
一つのコミットに縮めると全文がどのコミットにも残らず、仕様を変える作業項目で `check-spec-impact` が「仕様を変えたのに宣言する作業項目がない」と失敗する。

wi-52235 と wi-93464 では、変更と全文の記録を一つ目のコミットに、縮めた記録を二つ目のコミットにして通した。
これは wi-38309 を縮めた既存のコミットと同じ形である。

## 対象範囲

- チェックポイントと最終コミットの節に、全文を含む正式コミットと縮めた記録のコミットを分ける順序と、縮めるコミットの件名を書く。
- `implement-work-item` の手順 7 を同じ順序にする。

## 対象外

- `move-work-item` と `check-spec-impact` の動作。
- 中止する記録の扱い。中止も同じ順序に従うが、文言は変えない。

## 設計

順序は次のとおりにする。

1. `## 要約` を書き、チェックポイントを折り畳んで、作業項目の全文を含む正式コミットを作る。
2. `move-work-item` で記録を縮め、`check-work-items` と `check-links` を通して、`docs(work-items): complete <work-item-id> by collapsing it to its summary` の件名で二つ目のコミットを作る。

仕様を変えない作業項目でも同じ順序にする。
作業項目の種類で手順を分けると、どちらに当たるかの判断を誤ったときにだけ検査が失敗し、原因が分かりにくいためである。

## タスク

- [x] T001 [Docs] チェックポイントと最終コミットの節を直す。
- [x] T002 [Docs] `implement-work-item` の手順 7 を直す。
- [x] T003 [Verify] 変更を検証する。
  `check-links`、`check-agent-guidance`、`check-work-items`、`check-repository` が成功した。

## 検証

- `mise run check-links`
- `mise run check-agent-guidance`
- `mise run check-work-items`
- `mise run check-repository`

## リスク

- 二つ目のコミットを忘れると、要約まで書いた全文が `in_progress` のまま `work-items/active/` に残る。
  この状態は未完了の記録と区別できず、検査は報告しない。
  手順は二つのコミットを一続きに書き、最終報告でブランチのコミットを示すときに縮めたコミットがあることを読み手が確かめられるようにする。

## 要約

`mise run spec-diff` は規範の差分なしを報告する。
作業を完了するときは、作業項目の全文を含む正式コミットを作ってから、`move-work-item` で縮めた記録を `docs(work-items): complete <id> by collapsing it to its summary` の別のコミットにする手順にした。
縮めた記録は `affected_spec` を持たず、`check-spec-impact` は宣言を全文を最後に含むコミットから読むためである。
チェックポイントと最終コミットの節と、`implement-work-item` の手順 7 を同じ順序にそろえた。
