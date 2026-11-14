---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-013 }
---

# CSV の取り込みが Hard Quota で止まったときも QuotaExceeded を記録する

## 動機

REQ-TENANCY-013 は、作成を Hard Quota で拒否したときに `QuotaExceeded` を発行すると定める。
管理 API による User、Group、Agent の作成、アプリケーション、OAuth2Client の動的登録、同意、セッション、SSF ストリーム、ジョブの登録は、拒否のたびに `QuotaExceeded` を発行する。
User と Group の CSV の取り込みは、適用の途中で `users` または `groups` の上限に達すると、その行を失敗させる。
起票の後、Group の取り込みの PostgreSQL の実装（`group_import_committer.go`）は、行ごとに拒否の `QuotaExceeded` を発行するようになった。
User の取り込みの PostgreSQL の実装と、User と Group の取り込みのメモリーの実装は、まだ発行しない（2026-11-14 に確認）。
上限の逼迫を監査ログから読む運用者は、User の CSV の取り込みで止まった作成を見落とし、構成によって Group の結果も変わる。

wi-71372 で Tenancy の機能仕様を書き直したときに見つけ、仕様にない振る舞いの分類で (c) とした。

## 対象範囲

- User と Group の CSV の取り込みの適用で、Hard Quota による失敗ごとに、または取り込み一件につき一度、`QuotaExceeded` を発行する。Group の PostgreSQL の実装が行ごとに発行しているので、それに揃えるかを先に決める。
- PostgreSQL とメモリーの両方の実装で、同じ結果にする。

## 対象外

- 取り込みの行の失敗の表し方の変更。

## タスク

- [ ] T001 [Plan] 発行の単位を決める。
- [ ] T002 [App] 両方の実装で発行し、テストで固定する。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

- 行ごとに発行すると、大きな取り込みが監査ログを埋める。
  取り込み一件につき一度にする場合は、超過した行の数をイベントに載せるかを決める。
