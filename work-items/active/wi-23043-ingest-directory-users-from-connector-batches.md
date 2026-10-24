---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-24
priority: p3
depends_on: [wi-46712-enroll-a-directory-connector-and-authenticate-its-calls]
change_kind: feature
affected_spec:
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-002 }
  - { path: docs/modules/identity-management/features/user.md, requirement: REQ-IDMANAGEMENT-001 }
---

# Directory Connector が送る差分から、外部の不変 ID で User を冪等に取り込む

## 動機

[[wi-46712-enroll-a-directory-connector-and-authenticate-its-calls]] で登録した Connector は、IdMagic を認証付きで呼べるが、まだ何も取り込めない。
AD と LDAP を正とするテナントでは、ユーザーの作成、属性の変更、無効化、削除を、Connector が送る差分から IdMagic の User へ反映する必要がある。
名前や DN が変わっても同じ User と扱うため、相関には `objectGUID` または `entryUUID` のような外部の不変 ID を使う。

## 対象範囲

- Connector 向けに、User の差分のバッチを受け取る操作を設ける。各項目は外部の不変 ID、正規化済みの属性、有効か無効か、削除かを持つ。
- `(Connector, 外部の不変 ID)` から内部の User への対応（SourceCorrelation）を Sourcing が持ち、作成と更新を `IdentityWrites` から IdManagement の操作へ渡す。
- 同じバッチの再送を冪等に扱う。バッチの ID と、項目ごとの結果を返す。
- 削除は SCIM と同じく論理削除へ統合する。
- 完全な再同期の完了を Connector が報告したとき、その同期で現れなかった対応済みの User を無効にする。
- 対応のない既存の User とユーザー名が衝突する項目は、自動で結び付けずに失敗として返す。
- 1 回の取り込みを IngestionRun として記録し、適用件数と失敗を照会できる。
- 仕様：`directory` 機能の取り込みの要件、TypeSpec、データ、用語集の IngestionRun の実体。

## 対象外

- グループとメンバーシップ（[[wi-29505-ingest-directory-groups-and-direct-memberships]]）。
- LDAP の属性から正規の属性への対応付け。Connector の実行環境（[[wi-69206-run-a-directory-connector-that-reads-ad-and-ldap]]）が行い、IdMagic は正規化済みの属性だけを受け取る。
- 衝突した項目を管理者が結び付ける操作。

## 設計

### 未解決の問い

着手時に決める。

- 差分を送る操作を、Connector API の独自の操作にするか、SCIM（`/scim/v2`）に拡張を足すか。SCIM は長命の Bearer トークンで認証し、`id` を IdMagic が採番し、完全な再同期の完了と差分の位置を表せない。独自の操作は SCIM と同じ `IdentityWrites` へ渡すので、書き込みの作用は変わらない。
- 差分の位置（SourceCursor）を Connector の手元だけに置くか、IdMagic にも確定済みの位置として保存するか。Connector の再導入時の再開と、完全な再同期の判断に影響する。
- バッチの大きさの上限と、1 回の要求で IdManagement へ渡す書き込みの単位。
- SCIM の `ScimUserRef` と同じテーブルの形を共有するか、取り込み元ごとに別にするか。Sourcing の設計の要点に従い、共通点が実在するかを比べる。

## 計画

1. 未解決の問いを決め、境界の判断を記録する。
2. 仕様を先に書き、受け入れテストで初回、差分、再送、削除、完全な再同期、衝突を RED にしてから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いを決め、境界の判断を記録する。
- [ ] T002 [Spec] 取り込みの要件、TypeSpec、データ、用語集を書き、`affected_spec` を新しい要件へ差し替える。
- [ ] T003 [Acceptance] 初回、差分、再送、削除、完全な再同期、衝突、越境を RED で確かめる。
- [ ] T004 [App] 対応、IngestionRun、取り込みのユースケース、Repository、HTTP を実装する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run check-boundaries`、`mise run check`。DB の契約テストはサンドボックスの外で実行する。

## リスク

- 誤った無効化は、多数の利用者のサインインを止める。完全な再同期の完了報告がない限り無効にせず、無効化の件数を IngestionRun に残す。
- 同期の量が多い。バッチの上限と索引を T001 で見積もる。
