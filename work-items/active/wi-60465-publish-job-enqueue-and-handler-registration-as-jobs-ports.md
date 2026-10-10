---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "ジョブの投入と処理の登録を、Jobs の公開パッケージの経由へ変えるだけである。重複排除、再試行、レーンの規則、永続状態、ドメインイベントは変えない。" }
---

# ジョブの投入と処理の登録を Jobs の公開契約にする

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 9 件は、DataKeys、IdGovernance、IdManagement、OAuth2、Provisioning から `backend/jobs/usecases` への依存である。
使っているのは、投入の `Enqueue` と `EnqueueDeps`、処理の登録の `Handler` と `HandlerRegistry` だけである。
6 個のモジュールが Jobs の非公開の実装に依存しているので、Jobs の usecases の変更がこれらへ波及する。

## 対象範囲

- ジョブの投入と処理の登録を、Jobs の公開パッケージの型と操作として公開する。
- 6 個のモジュールの呼び出しを公開契約の経由へ直し、組み立て地点で Jobs の実装を結ぶ。
- 解消した違反 ID を台帳から消す。

## 対象外

- 投入の規則（重複排除、上限、レーン）の変更。
- Jobs の残りのパッケージの `internal/` への移動。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D3：利用側が必要とするのは「ジョブを投入する」と「種類ごとの処理を登録する」の二つだけである。重複排除、クォータ、再試行の規則は Jobs に残す。
- D4：IdManagement と OAuth2 は Jobs との間に循環の辺を持つ。利用側が投入の契約だけに依存すれば、Jobs の usecases の具象への依存は消える。

| 案 | 判断 |
| --- | --- |
| `jobs/ports` に投入と処理の登録の契約を置き、組み立て地点で Jobs の実装を渡す | 採る。利用側は投入の契約だけを知る |
| `EnqueueDeps` を公開パッケージへ移す | 採らない。保存先とクォータの port の組を利用側へ漏らす |

## タスク

- [ ] T001 [Design] 投入と処理の登録の契約の型を決める。
- [ ] T002 [App] 6 個のモジュールの呼び出しと組み立て地点を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 組み立て地点で投入の実装を渡し忘れると、ジョブが投入されない。
  構築関数の必須引数にして、渡し忘れをコンパイルで拒否する。
