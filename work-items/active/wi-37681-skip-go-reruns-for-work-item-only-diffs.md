---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-14
priority: p2
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: "完了処理での Go 検査の再実行の判断を文書へ記す。製品の HTTP 応答、認証と認可、永続状態、イベント、外部呼び出しは変更しない。"
---

# 検証済みの revision からの差分が作業記録だけなら、Go 検査を再実行しない判断を示す

## 動機

[wi-98584 の観察](wi-98584-reduce-follow-up-refactor-and-bugfix-items-at-first-implementation.md#観察の記録)には、完了記録の移動と最終コミットで Go 検証レポートが `stale` になり、製品コードを変えていない段階でも全 Go 検査を再実行した実例がある。
ツールで実行の結果を再利用する案は[wi-31542](../done/wi-31542-reuse-go-execution-evidence-for-unchanged-inputs.md)で試作し、保守の費用と誤った受理の危険から中止した。

[最終検査の選択](../../docs/development/specification-first-workflow.md#最終検査の選択)は、入力が変わらない検査を繰り返さないと定めている。
一方、[結果の参照](../../docs/development/testing.md#結果の参照)の `stale` は全追跡ファイルを入力とするため、作業記録だけの変化でも再実行が必要に見える。
この二つの関係を、確認の手順とともに文書で示したい。

## 対象範囲

- 検証時の作業ツリーがクリーンで、そのレポートの revision から現在の作業ツリーまでの差分が `work-items/` だけなら、Go の検査を再実行しなくてよいことを[最終検査の選択](../../docs/development/specification-first-workflow.md#最終検査の選択)へ記す。
- 差分の確かめ方（レポートの `input.revision` と `dirty`、`git diff --name-only <revision>`、未追跡ファイル）を示す。
- `report-verification` の `stale` 表示がこの判断を妨げないことを[結果の参照](../../docs/development/testing.md#結果の参照)へ記す。

## 対象外

- 検証ツールのコード、レポート形式、`mise` タスクの変更。
- `work-items/` 以外の文書だけの差分を、同じ判断の対象にすること。
- CI の必須検査の変更。

## 設計

判断はツールでなく手順で行う。
条件は、レポートの `input.dirty` が `false` であり、`git diff --name-only <input.revision>` と `git ls-files --others --exclude-standard` が `work-items/` 以外のパスを含まないことである。
条件を満たさなければ、従来どおり該当する Go 検査を再実行する。
完了記録の検証欄には、参照したレポートの revision と、差分が作業記録だけであることを確かめた結果を記す。

## タスク

- [ ] T001 [Docs] 最終検査の選択と結果の参照へ、条件と確かめ方を記す。
- [ ] T002 [Verify] `check-links`、`check-work-items`、`check-agent-guidance` で文書を確かめる。

## 検証

文書だけの変更であり、[振る舞いを保つ変更](../../docs/development/specification-first-workflow.md#振る舞いを保つ変更)の文書の変更に当たる。
`check-links`、`check-work-items`、`check-agent-guidance` の成功を記す。

## リスク

- 検証時に未コミットの変更があったレポートを基準にすると、差分の確認が検証した木を表さない。
  `input.dirty` が `true` のレポートは条件から外す。
