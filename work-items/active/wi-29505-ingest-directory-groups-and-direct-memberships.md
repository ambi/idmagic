---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-24
priority: p3
depends_on: [wi-23043-ingest-directory-users-from-connector-batches, wi-69206-run-a-directory-connector-that-reads-ad-and-ldap]
change_kind: feature
affected_spec:
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-005 }
---

# Directory Connector で、選んだグループと直接のメンバーシップを取り込む

## 動機

AD のグループは、アプリケーションへの割り当てとクレームの出どころになる。
[[wi-23043-ingest-directory-users-from-connector-batches]] は User だけを取り込むため、AD を正とするテナントでもグループを手で作り直すことになる。

## 対象範囲

- Connector の設定で選んだグループを、外部の不変 ID で Group へ取り込む。
- グループの直接のメンバーのうち、取り込み済みの User だけをメンバーシップとして反映する。
- グループの削除と、メンバーの追加と除去を差分で反映する。
- Connector の実行環境に、グループの読み取りを足す。

## 対象外

- 入れ子のグループの展開と階層の取り込み（[[wi-289-nested-group-hierarchy-and-inherited-membership]]）。
- 取り込んでいない User やコンピューターアカウントのメンバーシップ。

## 設計

User の取り込みの対応と IngestionRun をそのまま使い、取り込みの種類に Group を足す。
メンバーシップの書き込みは、SCIM の Group の同期と同じく `IdentityWrites` から IdManagement の操作へ渡す。

## タスク

- [ ] T001 [Spec] グループの取り込みの要件と TypeSpec を書き、`affected_spec` を新しい要件へ差し替える。
- [ ] T002 [Acceptance] グループの作成、メンバーの追加と除去、削除を RED で確かめる。
- [ ] T003 [App] IdMagic 側の取り込みと Connector の読み取りを実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run check`。DB の契約テストと LDAP の統合テストはサンドボックスの外で実行する。

## リスク

- 大きなグループのメンバーシップは量が多い。差分だけを送り、範囲付きの属性の取得（`member;range=`）に対応する。
