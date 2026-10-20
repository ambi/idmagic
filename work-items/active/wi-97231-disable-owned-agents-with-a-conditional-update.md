---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-081, impact: conforms }
---

# 所有者の停止に伴う Agent の無効化を条件付きの更新で確定する

## 動機

REQ-IDMANAGEMENT-081 は、所有者の User が止まると、その User が所有する Agent を無効化すると定める。
`backend/idmanagement/internal/agent/usecases/owner_offboarding.go` は Agent を読み、状態を変えて丸ごと保存する。
管理者の Agent の編集（`admin_agents.go`）も Agent を読んで丸ごと保存する。

管理者が Agent を編集している間に所有者が停止すると、編集の保存が読んだ時点の `Active` を書き戻し、所有者の止まった Agent が有効のまま残り得る。
無効化は自動の書き手なので、管理者は書き戻しに気づけない。

## 対象範囲

- 所有者の停止に伴う無効化を、`Active` の Agent だけを `Disabled` にする条件付きの更新へ置き換える。メモリと PostgreSQL の実装と共有の契約テストを揃える。
- 管理者の Agent の編集が、編集の間に行われた無効化を書き戻さないことを確かめ、書き戻す場合は直す。
- 修正後に、[同時の更新と排他](../../docs/design/data/database.md#同時の更新と排他)の適用状況と、IdManagement のデータとリスクの文書から、この無効化を除く。

## 対象外

- 管理者どうしの Agent の編集の競合（[wi-605](wi-605-detect-lost-updates-with-etag-and-if-match.md)）。
- 集約の一部だけを書く処理の設計規則（[wi-77130](../done/wi-77130-record-external-fetch-results-without-overwriting-configuration.md)）。

## 設計

無効化は `UPDATE ... SET status = 'disabled' ... WHERE owner = $1 AND status = 'active'` の形にし、無効化した Agent ごとのイベントは更新した行から発行する。
管理者の編集の保存は状態の列を書かないか、状態を書く操作（無効化、再有効化、停止）と分ける。
どちらにするかは、着手時に管理者の編集が変える列を列挙して決める。

## 計画

1. 管理者の編集の読み取りと保存の間に所有者の停止を挟むテストで、故障を RED にする。
2. 条件付きの更新と、管理者の編集の保存の修正を実装し、GREEN にする。

## タスク

- [ ] T001 [Acceptance] 管理者の編集が、間に行われた所有者の停止による無効化を書き戻す故障を RED にする。
- [ ] T002 [App] 条件付きの無効化と、管理者の編集の保存を直す。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- メモリと PostgreSQL の共有の契約テストで、条件付きの更新が `Active` の Agent だけを変えることを確かめる。PostgreSQL の検査はサンドボックスの外で実行する。
- `mise run test-go-changed`、`mise run lint-go`。

## リスク

無効化のイベントを更新した行から作るため、更新と発行の間の失敗の扱いを既存の無効化と揃える。
