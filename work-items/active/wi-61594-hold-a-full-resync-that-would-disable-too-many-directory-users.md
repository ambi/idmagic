---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/sourcing/features/directory.md, requirement: REQ-SOURCING-018 }
---

# 完全な再同期が多すぎる User を無効にするとき、無効化を保留して管理者の承認を待つ

## 動機

完全な再同期の完了は、実行に現れなかった対応済みの `Active` の User をすべて無効にする（REQ-SOURCING-018）。
Connector の `base_dns` や `user_filter` の誤り、AD 側の OU の改名で検索の範囲が外れると、一度の完了でテナントの多数の User がサインインできなくなる。
同種の製品は、この事故を件数のしきい値で止める。

| 製品 | 機能 |
| --- | --- |
| Microsoft Entra Connect Sync | Prevent accidental deletes。デフォルトで有効で、500 件を超える削除の書き出しを止める |
| Microsoft Entra Cloud Sync | Accidental delete prevention。しきい値を超える削除で同期を止め、通知先へ知らせる |
| Google Cloud Directory Sync | Delete limits。件数か割合の上限を超えると、その回の変更をすべて適用しない |

Connector 側の `check` コマンドは送る前に件数を示せるが、無効にする件数はサーバーだけが知っている。

## 対象範囲

- テナントの設定に、完全な再同期の完了で無効にしてよい件数のしきい値（件数と割合）を設ける。デフォルト値は着手時に決める。
- しきい値を超える完了は、実行を `completed` にせず保留の状態にし、無効にする候補の件数を記録し、管理者へ通知する。
- 管理者は保留した実行の候補を確かめ、承認して無効化を適用するか、破棄する。
- 保留の間に始まった同じ Connector の新しい実行の扱いを状態機械に足す。

## 対象外

- Connector 側のしきい値。無効にする件数を Connector は知らない。
- 削除の予約（`deleted` の項目）のしきい値。Connector は今は削除の項目を送らない。

## 設計

### 未解決の問い

- しきい値のデフォルト値と、件数と割合のどちらを使うか。
- 保留を IngestionRunLifecycle の新しい状態にするか、完了とは別の承認の記録にするか。
- 通知の経路（監査イベント、管理コンソール、メール）。

## 計画

1. 状態機械と要件の差分を設計に書き、利用者に確認する。
2. 仕様を更新し、ドメインから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いを決める。
- [ ] T002 [Spec] REQ-SOURCING-018 と IngestionRunLifecycle を改める。
- [ ] T003 [App] 保留、承認、破棄を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- 取り込みの受け入れテストで、しきい値の境界の前後の件数と、承認と破棄の後の User の状態を確かめる。

## リスク

- しきい値が小さすぎると、正当な大量の退職の反映が止まる。承認の操作を管理者が迷わず行えるようにする。
