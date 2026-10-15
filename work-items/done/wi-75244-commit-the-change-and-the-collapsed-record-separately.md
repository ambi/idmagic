---
status: completed
---

# 変更と縮めた記録を別のコミットにする

`mise run spec-diff` は規範の差分なしを報告する。
作業を完了するときは、作業項目の全文を含む正式コミットを作ってから、`move-work-item` で縮めた記録を `docs(work-items): complete <id> by collapsing it to its summary` の別のコミットにする手順にした。
縮めた記録は `affected_spec` を持たず、`check-spec-impact` は宣言を全文を最後に含むコミットから読むためである。
チェックポイントと最終コミットの節と、`implement-work-item` の手順 7 を同じ順序にそろえた。

全文は `git log -1 -- work-items/active/wi-75244-commit-the-change-and-the-collapsed-record-separately.md` が示すコミットの親、またはそのコミットで読む。
