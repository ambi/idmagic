---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-24
priority: p3
depends_on: [wi-23043-ingest-directory-users-from-connector-batches]
change_kind: feature
affected_spec:
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-002 }
---

# AD と LDAP を読み、差分を IdMagic へ送る Directory Connector の実行環境を作る

## 動機

[[wi-46712-enroll-a-directory-connector-and-authenticate-its-calls]] と [[wi-23043-ingest-directory-users-from-connector-batches]] で、IdMagic は Connector を受け入れ、差分を取り込めるようになる。
残るのは、AD と同じ閉域に置いて LDAP を読み、外向きの接続だけで差分を送る実行環境である。

## 対象範囲

- 新しい実行単位 `idmagic-directory-connector` を追加する。
- 登録のコマンドで鍵対を手元に作り、登録トークンで公開鍵を登録する。
- LDAPS または StartTLS で、最小権限のサービスアカウントとして bind する。bind の資格情報は Connector の実行環境だけに置く。
- ページングした検索で初回の完全な同期を行い、以後は差分を読む。AD では `uSNChanged` または DirSync、OpenLDAP では `modifyTimestamp` または `entryCSN` を使う。どれを使うかは T001 で決める。
- `objectGUID` または `entryUUID` を外部の不変 ID とし、LDAP の属性を正規の属性へ対応付けて送る。
- 送信の失敗を再試行し、確定した位置から再開する。
- テスト用の OpenLDAP、可能なら Samba AD をコンテナで用意し、統合テストで初回、差分、名前の変更、削除、TLS の失敗、再開を確かめる。
- 実行時の構成の文書に、配置、導入、登録、最小権限のサービスアカウント、鍵の更新、障害時の運用を書く。

## 対象外

- グループの読み取り（[[wi-29505-ingest-directory-groups-and-direct-memberships]]）とコンピューターアカウント（[[wi-43000-sync-active-directory-computer-accounts-through-the-directory-connector]]）。
- AD への書き戻しと、パスワードの取り込み、LDAP の bind による認証の委譲。
- Connector の自動更新の配布。

## 設計

### 未解決の問い

着手時に決める。

- AD の差分取得に `uSNChanged` と DirSync のどちらを使うか。DirSync は削除を返せるが、権限の要件が強い。
- LDAP のクライアントのライブラリ。依存の追加として選定を記録する。
- Connector の配布形態（単体のバイナリ、コンテナ、またはその両方）。

## 計画

1. 未解決の問いを決め、実行時の構成と依存の追加を設計へ記録する。
2. 統合テストの fixture を先に用意し、初回の同期を RED にしてから実装する。

## タスク

- [ ] T001 [Plan] 差分取得の方式、ライブラリ、配布形態を決める。
- [ ] T002 [Spec] Connector の振る舞いの要件と、実行時の構成の文書を書く。
- [ ] T003 [Acceptance] OpenLDAP の fixture で、初回の同期と差分の取り込みを RED で確かめる。
- [ ] T004 [App] 登録、読み取り、対応付け、送信、再開を実装する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run check`、LDAP の統合テスト。統合テストはコンテナを使うのでサンドボックスの外で実行する。

## リスク

- AD の実装差で差分を取りこぼすと、無効化が反映されない。定期的な完全な再同期で回収し、その間隔を設定できるようにする。
- bind の資格情報がログや送信内容に出ると、閉域の外へ秘密が漏れる。ログと送信内容に出ないことをテストで確かめる。
