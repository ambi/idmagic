---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/signing-keys/features/lifecycle.md, requirement: REQ-SIGNINGKEYS-003 }
  - { path: docs/modules/signing-keys/features/lifecycle.md, requirement: REQ-SIGNINGKEYS-010 }
---

# 検証用の署名鍵の無効化を、構成によらず同じ結果にし、監査へ残す

## 動機

管理者が検証用の署名鍵を無効化すると、PostgreSQL の構成では鍵を `Archived` にし、メモリの構成では鍵を削除する。
どちらの構成も、無効化をドメインイベントとして発行しない。
署名鍵の失効は、署名済みの監査トークンの検証に関わるので、構成で結果が変わり、監査に残らないことは欠陥である。

## 対象範囲

- 両方の構成で、無効化した鍵を `Archived` にし、鍵素材を保持する。
- 無効化をドメインイベントとして発行し、要件と状態遷移の節に書く。

## 対象外

- ローテーションと退役の振る舞いの変更。

## 設計

### 負債の棚卸しでの判断

2026-10-18 時点のライフサイクル仕様は、重複期間の経過と管理者の無効化ではドメインイベントを発行しないと明記している。
`SigningKeyRetired` の未使用の宣言の削除は wi-96851 で行い、この項目の監査イベントの実装済みとは扱わない。
退役は時刻による `Retired` への判定、無効化は `Archived` への遷移であるため、同じイベントとして実装しない。
この負債の整理では監査イベントの新設を見送る。
着手時には、既存の無効化の要件を先に変え、メモリ保存先での鍵素材の保持と、両保存先に共通するイベントの契約を一緒に確定する。

ストアの契約テスト（`testing_contract`）に無効化の結果を加え、両方の実装を同じテストで確かめる。
イベントの名前は既存の `SigningKey*` の命名に合わせて決め、TypeSpec に宣言する。

## タスク

- [ ] T001 [Spec] 要件と状態遷移の節に無効化のイベントを書く。
- [ ] T002 [Acceptance] 契約テストでメモリの構成の RED を確認する。
- [ ] T003 [App] 両方の構成を GREEN にし、イベントを発行する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run verify`

## リスク

- メモリの構成に依存するテストが削除の結果を前提にしている場合がある。失敗したテストを一つずつ読み、前提を直す。
