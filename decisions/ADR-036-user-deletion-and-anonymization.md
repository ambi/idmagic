---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-036: ユーザー削除と即時匿名化 (anonymize cascade)

## コンテキスト

ADR-031 で `/admin/users` の Disable / Enable は実装したが、削除経路が無い
ため次が成立しない。

- データ主体 (end user) の削除要求 (GDPR Art.17 right-to-erasure)。
- 退職処理。Disable のみだと audit / consent / refresh token / session が
  残り、攻撃時に「無効化された旧アカウントを起点に refresh token を再
  活性化される」リスクと運用衛生上の懸念が同居する。
- tenant 内のテスト用 user の本格的な掃除 (デモシードと衝突する)。

単純な hard delete は採用しない。

- `AdminAuditEvent` などの append-only ログが `sub` を参照しており、
  参照整合性 (概念上) を壊す。
- 削除と無効化の差を運用上見分けたい。
- GDPR 文脈でも "anonymize で sub + 一意化トークンを残す" 形が一般的。

## 決定

`User` aggregate に tombstone 状態を導入する。削除は物理削除ではなく即時匿名化とし、`sub` は audit
参照のため不変で残す。関連 aggregate (`Consent` / `RefreshTokenRecord` / `LoginSession` /
`PasswordHistory` / `MfaFactor` / `DeviceAuthorization`) は cascade で削除し、操作は冪等とする。
actor が自分自身の `admin` / `system_admin` アカウントを削除する操作は拒否する (自爆防止)。
`UserDeleted` を `AdminAuditEvent` として永続化する。

物理削除を採らないのは、`AdminAuditEvent` などの append-only ログが `sub` を参照しており hard
delete が参照整合性を壊すこと、削除と無効化を運用上見分けたいこと、GDPR 文脈でも「anonymize で
sub + 一意化トークンを残す」形が一般的であることによる。tombstone 置換アルゴリズム・cascade 対象の
一覧・`preferred_username` の再利用可否・削除済 user のトークン経路・自爆防止と冪等性の詳細は
[`backend/idmanagement/ARCHITECTURE.md`](../backend/idmanagement/ARCHITECTURE.md) に置く。

## 影響

- SCL に `UserLifecycle` 状態機械、`Delete` vocabulary、`Deleted` 終端
  状態、`DeleteUser` interface、`UserDeleted` event、`AdminUserDelete`
  permission が追加される。
- Authentication component の `owns_states` / `owns_events` /
  `owns_interfaces` / `owns_permissions` が更新される。
- `PiiPurgeAfterDeletion` objective は「削除時に即 PII 匿名化」を
  根拠に retention を `0s` に変更し、`+30日` の物理消去予定は撤回する
  (anonymize 自体が物理消去と同等の PII 排除を提供するため)。
- Go の repository port に `DeleteAllForSub(ctx, sub)` (cascade 対象) と
  `MarkDeleted(ctx, sub, now, tombstone)` (User) が増える。memory /
  postgres / valkey の各 adapter に新規メソッドが実装される。
- HTTP に `DELETE /api/admin/users/{sub}` が追加される。CSRF + Origin
  + `requireAdmin` の既存ガードを継承し、新規 RBAC 経路は作らない。
- UI に削除ダイアログ (preferred_username typing confirm + reason) が
  追加される。AdminUsersPage の `FindAll` は既に `deleted_at IS NULL` の
  user だけ返すため、削除後は一覧から自動的に消える。
- 既存 `disabled_at` 経路には影響なし。Disable と Delete は独立した
  終端で、`disabled_at != null` の user は restoration 可能なまま。
- Hard delete は debug 用 CLI も含めて提供しない。後の "30 日 grace
  期間 + 物理消去" の要件が来た場合は本 ADR を改定する。
