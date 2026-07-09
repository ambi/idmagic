---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-15
---

# 管理画面からのユーザ削除 (anonymize cascade) を導入する

## Motivation
現状 `/admin/users` には Disable / Enable しか無く、ユーザを「もう存在
しないもの」として扱う経路が無い。これにより次が成立しない:
  - データ主体 (end user) の削除要求 (GDPR Art.17 right-to-erasure)。
  - 退職処理。Disable のみだと audit / consent / refresh token / session
    が残り、攻撃時に「無効化された旧アカウントを起点に refresh token を
    再活性化される」リスクと運用衛生上の懸念が同居する。
  - tenant 内のテスト用 user の本格的な掃除 (デモシードと衝突するため)。

単純な hard delete は次の問題で採用しない:
  - `AdminAuditEvent` などの append-only ログが `sub` を参照しており、
    参照整合性 (実装上は柔らかいが概念的に) を壊す。
  - 削除と無効化の差を運用上見分けたい。
  - GDPR 文脈でも "anonymize で sub + 一意化トークンを残す" 形が一般的。

本 WI では `User` aggregate に `deleted_at` と tombstone 状態を導入し、
関連 aggregate (Consent / RefreshTokenFamily / Session / PasswordHistory /
Device 関連) を cascade 失効させる。`sub` 自体は再利用しない方針とし、
audit trace を保つ。

## Scope
- **decision**:
  - 新規 ADR (`idmagic/decisions/ADR-036-user-deletion-and-anonymization.md`): hard delete ではなく anonymize を既定とする方針、tombstone のフィールド 置換規約 (`preferred_username = "deleted:<sub>"`、`name = ""`、 `email = nil`、`password_hash = nil`、`mfa_*` をクリア)、cascade 対象 aggregate の列挙、`deleted_at` 設定後の認証・introspection・userinfo の挙動 (login 不可 / introspect は active=false / userinfo は 401)、 `sub` の再利用禁止、deletion 自体の audit event 規約。
- **scl**:
  - `User` model に `deleted_at: timestamp | null` を追加。
  - `state_machines.User` に `Deleted` 終端状態を追加 (`Active` / `Disabled` → `Deleted`、`Deleted` から戻る遷移は持たない)。
  - `interfaces` に `DeleteUser` (admin) と `user.deleted` event を追加。
  - `permissions` の `admin.users.write` に DeleteUser を含める。
  - `objectives` の認証・トークン保証義務に「`deleted_at != nil` の user で 認証成立せず」を明記。
- **go**:
  - domain:
    - internal/spec/ の User struct に `DeletedAt *time.Time` を追加し、 `IsDeleted()` helper を提供。`FindBySub` 系で見つかった場合でも login / consent issuance / token issuance は拒否する。
  - usecases:
    - authusecases.DeleteUser を新設。actor RBAC 確認 → user lookup → tombstone 置換 → 関連 aggregate cascade → `user.deleted` event を emit する。冪等 (既に deleted の場合は no-op + 200)。
    - 「最後の admin を自分で削除する」を拒否する pre-check を入れる (actor.Sub == target.Sub && admin role の場合 reject)。
  - persistence:
    - UserRepo に `MarkDeleted(ctx, sub, now, tombstone)` を追加 (memory / postgres)。
    - 既存 ConsentRepo / RefreshTokenFamilyRepo / SessionRepo / PasswordHistoryRepo / DeviceAuthRepo に `DeleteAllForSub(ctx, sub)` を追加する。Lua / トランザクションを使い同一バッチで実行する。
    - PostgreSQL migration を 1 本追加 (`users.deleted_at` カラム、必要な index)。AUTO_MIGRATE で前方互換に流れる。
  - http:
    - DELETE `/admin/users/:sub` を追加。既存 PATCH / POST disable と 同様の verifyBrowserRequest + requireAdmin の上に乗せる。
    - 削除済 user を再度 DELETE しても 200 / no-op (idempotent)。
  - introspection_userinfo:
    - `/introspect` は `deleted_at != nil` の subject を `active=false` として返す。
    - `/userinfo` は deleted user では 401 `invalid_token`。
  - audit:
    - AdminAuditEvent に `user.deleted` を追加し、actor_sub / target_sub / reason (任意 free-text) を残す。
- **ui**:
  - AdminUsersPage の詳細パネル下部 (`onDisabled` セクションの下) に 「アカウントを削除」危険ボタンを追加する。Disable と別行にする。
  - 削除ダイアログでは対象 user の `preferred_username` を入力させ、 一致した場合のみ実行ボタンを有効化する (ミス防止)。任意の `reason` フィールドを 1 つ用意し、audit event に同梱する。
  - 削除済 user は一覧に出さない既定 (status filter `deleted` を持つ `system_admin` 限定の表示は本 WI では作らない)。
  - StatusBadge は「削除済」を扱わない (詳細パネル側にだけ表示するため)。
- **documentation**:
  - idmagic/README.md の Phase 4 「ユーザ削除」行は completion で 除去を記録する。
  - decisions/CONCEPTION.md / CONCEPTION_BASELINE.md は変更しない (model の追加のみで base concept は不変)。

## Out of Scope
- 完全 hard delete (debug 用 CLI も含めて提供しない)。
- "削除予約 + 30 日 grace 期間" のような soft 状態。将来別 WI で 積み増せる設計にしておくに留める。
- データ主体への export (DSAR の export 側は Phase 5)。
- `sub` を生成する際の予測困難化 / opaque 化 (現状は authusecases 側の 既存生成器をそのまま使う)。
- SCIM 経由の deprovisioning (Phase 6)。
- 監査ログ自身の削除・改竄防止強化 (Phase 8 のスコープ)。

## Verification
- `go test ./...` (in: idmagic)
  - reason: DeleteUser use case + cascade テスト + introspect/userinfo の deleted user 経路 + idempotency。
- `go vet ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動: dev seed の `bob` を作成して active token を発行 → AdminUsersPage で削除 → introspect が active=false / userinfo が 401 / refresh が `invalid_grant` を返す / 再 login 不可。
- 手動: 自分自身を消そうとして失敗することを確認 (最終 admin 自爆防止)。

## Risk Notes
cascade を取りこぼすと「user は消えたが consent / refresh family が
生き残る」状態になる。新規 DeleteAllForSub を全 repository に強制し、
use case 側で 1 トランザクションで束ねる。PostgreSQL adapter は
pgx.BeginTx を使い、Valkey 側は Lua スクリプトでまとめて削除する。
cascade テストは memory / postgres 両 adapter で同じ table-driven テスト
を回し、回帰防止する。

## Completion
- **Completed At**: 2026-06-16
- **Summary**:
  `/admin/users` に DELETE 経路を追加し、削除を「即時 anonymize cascade」
  として実装した。ADR-036 を新規に採用し、SCL では User 状態機械
  `UserLifecycle` の追加・終端状態 `Deleted`・`DeleteUser` interface・
  `UserDeleted` event・`AdminUserDelete` permission を入れた。
  バックエンドは新規 use case `authusecases.DeleteUser` で tombstone 化と
  関連 aggregate (Consent / RefreshToken / Session / PasswordHistory /
  MfaFactor / DeviceAuthorization) の cascade 削除を一括で行う。HTTP は
  `DELETE /api/admin/users/:sub` を既存 `verifyBrowserRequest` +
  `requireAdmin` の上に乗せ、UI は AdminUsersPage に "アカウントを削除"
  危険ボタンと preferred_username typing 確認ダイアログを追加した。
- **Verification Results**:
  - `go test ./...` (in: idmagic)
    - result: ok (全パッケージ pass)
  - `go vet ./...` (in: idmagic)
    - result: ok
  - `go build ./...` (in: idmagic)
    - result: ok
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui lint`
    - result: ok (40 files、警告無し)
  - `bun --cwd idmagic/ui build`
    - result: ok (vite build pass、6284 modules)
  - 手動確認 (residual): dev サーバ上での "AdminUsersPage で削除 → introspect が active=false / userinfo が 401 / refresh が invalid_grant / 再 login 不可" の e2e は本セッションでは未実施。memory adapter で cascade と tombstone は単体テストで検証済。
- **Affected Guarantees State**:
  - tenant isolation: cross-tenant delete は禁止のまま。`requireAdmin` が actor.TenantID == request.TenantID を要求し、DeleteUser use case でも target.TenantID != tenancy.TenantID(ctx) のとき ErrUserNotFound を 返す。
  - 自爆防止: actor.Sub == target.Sub かつ target.Roles に admin / system_admin が含まれる場合は ErrSelfDeleteForbidden で reject。HTTP は 400 self_delete_forbidden として返す。
  - 既発行 access token: short-lived JWT は expiry まで RS で検証は通る (本 WI スコープ外)。本 WI では `/introspect` (refresh token 経由) と `/userinfo` (deleted user に対して 401 invalid_token) を整合させ、 refresh token は cascade で消去されるため `/token` (refresh) でも invalid_grant が返るようにした。
  - SCL coherence: UserLifecycle と Delete vocabulary、UserDeleted event、 AdminUserDelete permission を追加。SCLPermissionsCoverage 系の coherence test と admin policy test、state machine validation を pass。
  - 監査保全: UserDeleted event は actor_sub / target_sub / reason を持ち、 AdminAuditEvent として永続化される (既存 audit emit 経路に乗る)。
  - backend 既存挙動: FindBySub のセマンティクスを memory / postgres 両 adapter で揃え (deleted を除外)、FindBySubIncludingDeleted を 別途追加。これにより既存の oauth2/refresh/exchange/device/userinfo callers は意図せず deleted user を見ない。
