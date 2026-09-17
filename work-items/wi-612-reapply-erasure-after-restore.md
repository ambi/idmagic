---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/standards.md, requirement: GDPR-ERASURE }
---

# バックアップから復元すると、消去した利用者と破棄した DEK が戻る

## Motivation

バックアップは取得した時点のデータを保持する。
取得より後に利用者を消去したり DEK を破棄したりしても、そのバックアップから復元すると、消去前の行と `wrapped_dek` が戻る。
マスター鍵が残っている限り、戻った DEK は再びアンラップでき、暗号学的に消去した秘密情報も復号できる。

これを防ぐ手段は二つの組み合わせになる。

- バックアップの保持期間を、`GDPR-ERASURE` の消去の期限の内側に収める。
- 復元した後に、復元時点より後に行った消去を再適用する。

現在はどちらもない。
バックアップの保持期間は未確定であり、`infra/backup/restore-postgres.sh` は短命な表を空にするが、消去の再適用は行わない。
復元時点より後に発行された `UserDeleted` などの監査イベントは同じデータベースにあるので、復元すると消去の記録そのものも失われる。

## Scope

- 消去の記録（利用者の消去、DEK の破棄）を、PostgreSQL のバックアップとは別の場所に、復元時点より新しい状態で残す仕組みを決める。
- 復元の手順に、その記録にもとづく消去の再適用を加える。
- バックアップの保持期間と消去の期限の関係を、品質要求または標準仕様に記録する。

## Out of Scope

- バックアップの頻度、保管先、RPO と RTO。[[wi-588-reliability-and-performance-design]] が扱う。
- テナントの物理削除。[[wi-494-system-console-tenant-management-and-resource-inspection]] が扱う。

## Risk Notes

消去の記録を別の場所に置くと、その記録自体が個人識別情報を含まないように設計する必要がある。記録は利用者の識別子と時刻に限る。

## Verification

- `mise run restore-drill`
- `mise run verify`
