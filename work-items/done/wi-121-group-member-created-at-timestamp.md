---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-04
---

# GroupMember の所属追加時刻を created_at に統合する

## Motivation
GroupMember は User と Group の所属関係そのものを表す insert/delete-only の永続化状態モデルであり、
現状の要件では「行が作成された時刻」と「所属が追加された時刻」は同じ意味を持つ。
`created_at` と `added_at` を併存させると、同じ時刻概念を二重管理することになり、
PersistedModelTimestamps invariant の「永続化される状態モデルは作成時刻を持つ」という
横断方針とも読み取りづらくなる。

外部システム由来の過去の所属開始日を別管理する要件はまだ無いため、GroupMember の
所属追加時刻は `created_at` に統合し、API/UI も同じモデル属性を返す。

## Scope
- SCL: `IdentityManagement.models.GroupMember` と `GroupMemberResponse`。
- Schema: `deploy/schema/postgres.sql` の `group_members`。
- Implementation: `internal/shared/spec/groups.go`、Postgres/memory repository、identity management、SCIM、UI 型。

## Out of Scope
- 外部システム由来の過去の所属開始日を別属性として扱う機能追加。
- GroupMember の更新可能化。
- 既存データ migration/backfill の実装。

## Initial Context
- `group_members` に `created_at` と `added_at` が併存している。
- 現状 `added_at` は `created_at` と独立した domain timestamp ではない。
- GroupMember は insert/delete-only の状態モデルとして `created_at` を持ち、`updated_at` と `added_at` は持たない。

## Affected Guarantees
- PersistedModelTimestamps invariant と GroupMember モデルの整合性。
- Group membership API/UI の時刻属性の一貫性。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`

## Risk Notes
API response の `added_at` が `created_at` に変わるため、UI 型と利用箇所を同時に更新する必要がある。
既存データ migration はこの repository の現行 declarative schema 変更範囲外とする。

## Completion
- **Completed At**: 2026-07-04
- **Summary**:
  GroupMember の所属追加時刻を `created_at` に統合し、`added_at` を SCL model、Postgres schema、
  Go model/repository/usecase、SCIM 同期経路、admin HTTP response、UI 型から削除した。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just scl-render` - passed
  - `just verify-go` - passed
  - `just verify-ui` - passed
- **Affected Guarantees State**:
  GroupMember は insert/delete-only の永続化状態モデルとして `created_at` だけを持ち、
  `updated_at` と独立 domain timestamp の `added_at` は持たない状態になった。
