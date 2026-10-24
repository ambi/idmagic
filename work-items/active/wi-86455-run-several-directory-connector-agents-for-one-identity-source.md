---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-24
priority: p3
depends_on: [wi-69206-run-a-directory-connector-that-reads-ad-and-ldap]
change_kind: feature
affected_spec:
  - { path: docs/modules/sourcing/features/directory.md, requirement: REQ-SOURCING-015 }
---

# 一つの取り込み元に複数の Directory Connector の Agent を置き、一台が止まっても同期を続ける

## 動機

今の Directory Connector は、一つの登録に一つの実行単位を前提とする。
新しい実行は同じ Connector の進行中の実行を打ち切る（REQ-SOURCING-015）ので、同じ登録で二台を動かすと互いの実行を打ち切り合う。
一台のホストが止まると、復旧まで同期も無効化の反映も止まる。

Okta は AD Agent を一つのドメインに 2 台以上置くことを推奨し、120 秒応答のない Agent を外して残りへ仕事を回す。
取り込みは優先の Agent に続けて割り当て、その Agent が使えなくなったときだけ別の Agent へ移す。

## 対象範囲

- 一つの Connector（取り込み元）に複数の Agent の鍵を登録できるようにする。
- 同時に一つの Agent だけが実行を持つように、サーバーが実行の担当を貸し出し（リース）、期限内の接続の報告で延ばす。
- 担当の Agent が期限までに報告しなければ、ほかの Agent が担当を引き継ぎ、完全な再同期から始める。
- 管理 API と管理コンソールに Agent ごとの最終接続を示す。

## 対象外

- Agent の間での差分の位置の共有。引き継いだ Agent は完全な再同期で回復する。
- Agent の自動更新の配布。

## 設計

### 未解決の問い

- Agent を Connector の下の別の記録にするか、Connector を複数作って取り込み元をまとめる記録を足すか。
- リースの期限と、引き継ぎで完全な再同期を強いるかどうか。

## 計画

1. 境界の判断と状態機械の案を設計に書き、利用者に確認する。
2. Sourcing の仕様と TypeSpec、DirectoryConnector の仕様を更新してから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いを決める。
- [ ] T002 [Spec] REQ-SOURCING-015 と関連する要件、Connector API を改める。
- [ ] T003 [App] リースと引き継ぎを実装する。
- [ ] T004 [Verify] 二つの Agent の E2E で引き継ぎを確かめる。

## 検証

- 二つの Agent を同じ取り込み元に登録し、担当の Agent を止めると、もう一方が期限の後に同期を引き継ぐことを E2E で確かめる。

## リスク

- リースの期限が短すぎると、遅いネットワークで担当が揺れ、完全な再同期が繰り返される。
