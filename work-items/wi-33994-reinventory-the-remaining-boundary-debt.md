---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: [wi-17193-move-context-specific-code-out-of-support-http, wi-39119-publish-the-resolved-tenant-as-tenancy-public-language, wi-65906-inject-time-randomness-and-network-into-domain, wi-92970-adopt-information-hiding-modular-design]
change_kind: tooling
spec_impact: { kind: none, reason: "境界の負債の台帳の理由を書き直し、残りの解消を work item として起票するだけである。製品のコード、HTTP の応答、ドメインイベント、永続状態は変えない。" }
---

# 残った境界の負債をモジュールの組ごとに棚卸しし、理由を書き直す

## 動機

`tools/check/boundary-debt.json` の `private-import`、`module-cycle`、`shared-dependency` の項目の理由は、旧台帳から引き継いだ定型文か、移行のときに機械的に作った文である。
定型文の理由からは、どの依存をどう直すのかが読み取れず、解消の work item も起票されていない。

wi-17193、wi-39119、wi-65906 で、原因が一か所に集まった負債（`support_http` を経由した依存、Tenancy への依存、`domain` の作用）が消える。
残るのは、モジュールの組ごとに原因の異なる依存である。
現時点で多いのは、`private-import` の参照先の `backend/claimmapping/usecases`（11）と `backend/jobs/usecases`（9）、および 57 件の `module-cycle` である。

`table-write` の 7 件は、原子性の理由と参照を持つ具体的な理由を移行の時点で書いてある。
ただし、公開操作を別のトランザクションで呼ぶ形へ変えるだけでは原子性を保てないので解消にならない。

## 対象範囲

- 前提の 3 項目が完了した後の `mise run check-boundaries` の結果から、残った違反をモジュールの組ごとにまとめる。
- 組ごとに、[境界を選ぶ判断手順](../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D1〜D8 で直す方法を決める。
  非公開の処理への依存は D3、循環は D4、共有ライブラリからの依存は D5、変換の配置は D6、所有者外の書き込みは D2 と D7 を適用する。
- 直す方法と、それが必要な理由を、台帳の各項目の理由として書き直す。
- 直すための work item を、まとめて直せる単位で起票する。
- `table-write` の 7 件は、呼び出し側のトランザクションに参加する公開操作を所有者が公開する案（D2、D7）を比べ、採るなら解消の work item を起票する。

## 対象外

- 個々の依存と書き込みの解消そのもの。起票した work item で行う。

## タスク

- [ ] T001 [Inventory] 残った違反をモジュールの組ごとにまとめ、D1〜D8 で直す方法を決める。
- [ ] T002 [Tooling] 台帳の理由を書き直す。
- [ ] T003 [Plan] 解消の work item を起票する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-work-items`

## リスク

- 理由を書き直すだけでは負債は減らない。
  この項目の完了の条件を、すべての残りの項目に解消の work item が対応していることにする。
