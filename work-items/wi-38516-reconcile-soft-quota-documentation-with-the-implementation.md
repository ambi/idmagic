---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-01
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/tenancy/quota/scenarios.feature.md, requirement: REQ-TENANCY-036 }
---

# Soft Quota の文書と実装の食い違いを解消する

## 動機

リソース上限の内部設計は、Soft Quota（`audit_events_retained`、`export_artifacts_bytes`）について、操作を成功させたうえで非同期の警告と監査イベントを発行すると書く。
しかし、この二つのリソースの使用量を加算する呼び出しも、警告を発行する処理も、リポジトリにない。
上限の値は保存できるが、何にも使われていない。

バックエンドの間でも扱いが食い違う。
PostgreSQL の実装はこの二つのリソースの加算を受け付け、メモリーの実装は未知のリソースとして拒否する。
Tenancy の既存コードを書き起こしたとき、Hard Quota だけを規則（REQ-TENANCY-036）にし、Soft Quota は書き起こさなかった。

## 対象範囲

- Soft Quota を実装するか、文書と保存できる項目から外すかを決める。
- 決めた結果に合わせて、内部設計、規則、TypeSpec、両方のバックエンドをそろえる。

## 対象外

- Hard Quota の変更。

## 計画

1. Soft Quota を必要とする利用者の要件を確かめる。
2. 実装するか外すかを決め、仕様を先に変える。
3. 両方のバックエンドを同じ契約にそろえ、共通の契約テストで固定する。

## タスク

- [ ] T001 [Plan] Soft Quota を実装するか外すかを決める。
- [ ] T002 [Spec] 仕様を更新する。
- [ ] T003 [App] 両方のバックエンドを同じ契約にそろえる。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- 保存できる項目を外すと、上限を設定済みのテナントの値が失われる。
