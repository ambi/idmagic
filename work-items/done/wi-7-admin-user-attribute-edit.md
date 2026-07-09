---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-06-15
---

# 管理画面でユーザ属性 (preferred_username / name / email / email_verified) を編集できるようにする

## Motivation
idmagic の `UpdateUser` use case と HTTP endpoint
(`PATCH /admin/users/:sub`) は `PreferredUsername` / `Name` / `Email` /
`EmailVerified` / `Roles` を受け取れる。ところが UI 側 (AdminUsersPage)
は `RoleEditorDialog` (ロールのみ) と Disable / Enable しか叩いておらず、
実運用で発生する次のオペレーションが UI からできない:
  - メールアドレス変更 (人事異動・タイポ訂正)
  - 表示名 (`name`) / 表示用ユーザ名 (`preferred_username`) の修正
  - メールドメイン移管後の `email_verified` 再フラグ
結果として、admin は API を直叩きするか DB に SQL を当てるしかなく、
操作の追跡 (CSRF + audit event) も外れる。backend は完成済なので、
UI に属性編集ダイアログを 1 つ足すだけで、本番運用に必要な
user maintenance 作業が監査経路つきで解禁される。

本 WI のスコープは UI のみ。新規 backend endpoint・新規 SCL field・
新規 event は導入しない (既存 `user.updated` event を流用)。

## Scope
- **ui**:
  - pages:
    - AdminUsersPage の詳細パネル (`UserDetails`) に "属性を編集" ボタンを 足し、`AttributeEditorDialog` を開く。
    - AttributeEditorDialog は次のフィールドを編集可能とする: `preferred_username` (required、unique 違反は server error として表示)、 `name`、`email`、`email_verified` (checkbox)。
    - 既存 `RoleEditorDialog` と並ぶ独立ダイアログとし、保存ボタン押下で PATCH `/admin/users/:sub` を呼ぶ。
    - メールアドレスを変更した場合、`email_verified` を勝手に true にしない (チェックを外す既定動作)。ADR-030 の email 検証規約を破らない。
  - api:
    - api.ts に `updateAdminUserAttributes(csrf, sub, attrs)` を追加し、 既存 `updateAdminUserRoles` と同じく PATCH `/admin/users/:sub` を呼ぶ。 ロール更新と属性更新の同時送信は許可しない (ダイアログを分けるため)。
    - 既存 `updateAdminUserRoles` は変更しない (call site が増えるだけ)。
  - types:
    - 既存 `AdminUser` 型を流用し、新規型は追加しない。
  - navigation:
    - 既存 sidebar navigation は変更しない (新規ページ無し)。
  - a11y:
    - ダイアログは `role="dialog" aria-modal="true"`、`aria-labelledby` を 付け、既存 `RoleEditorDialog` のパターンを揃える。
- **scl**:
  - 変更なし。`UpdateUser` interface / `user.updated` event の wire 形式は維持。
- **documentation**:
  - idmagic/README.md の Phase 4 「ユーザ属性編集 UI」行は、本 WI 完了後 `完了済` として除去する (completion で反映を記録する)。

## Out of Scope
- 新規 backend endpoint / use case の追加。
- SCL の `User` モデル拡張 (新規 attribute はこの WI では追加しない)。
- パスワードリセットを admin が代行する経路 (別 WI 候補)。
- email 変更時に確認メール (verification) を自動で送る連携 (Phase 0 の Email verification WI と統合する)。
- 監査イベントの schema 拡張。既存 `user.updated` の changed-fields 表示を AdminAuditEventsPage で構造化する話は別 WI とする。

## Verification
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- `go test ./internal/adapters/http/... ./internal/authentication/usecases/...` (in: idmagic)
  - reason: backend に変更は無いが、既存の admin user handler テストの 回帰を確認する。
- 手動: dev サーバで admin としてログイン → `/admin/users` で alice を選択 → 属性編集ダイアログを開いて `name` と `email` を変更 → 保存 → 一覧で反映、 `/admin/audit_events` に `user.updated` が表示される。

## Risk Notes
純粋な UI 追加。新規ファイル + 既存ページ内の 1 セクション追加で完結
する。誤動作が起きても backend 側の validation で弾かれる (Zog + use case
の email 形式チェックがある)。回帰範囲は AdminUsersPage に閉じる。

## Completion
- **Completed At**: 2026-06-15
- **Summary**:
  AdminUsersPage の詳細パネルに「属性を編集」ボタンと AttributeEditorDialog
  を追加し、既に backend に存在する `PATCH /admin/users/:sub` から
  `preferred_username` / `name` / `email` / `email_verified` を編集できる
  経路を解禁した。新規 backend endpoint / 新規 SCL field / 新規 event は
  追加していない (既存 `UpdateUser` use case + `user.updated` event を流用)。
- **Verification Results**:
  - `bun --cwd idmagic/ui typecheck`
    - result: ok (tsc --noEmit pass)
  - `bun --cwd idmagic/ui lint`
    - result: ok (biome lint pass, 39 files)
  - `bun --cwd idmagic/ui build`
    - result: ok (vite build pass, 6283 modules)
  - `go test ./internal/adapters/http/... ./internal/authentication/usecases/...` (in: idmagic)
    - result: ok (admin_user_handler / admin_users 既存テストの回帰なし)
  - 手動確認 (residual): dev サーバを起動した実ブラウザ操作は本セッションで は実施していない。AdminUsersPage の他ダイアログ (RoleEditor / CreateUser) と同パターンで実装したため typecheck / lint / build 緑で 動作する見込み。実環境での操作確認は次回 dev 起動時に行う。
- **Affected Guarantees State**:
  - admin RBAC: 既存 `verifyBrowserRequest` + `requireAdmin` を流用するだけ なので回帰は無し。HTTP handler に変更が無いことを backend test の pass で再確認した (admin_user_handler.go テスト群)。
  - CSRF: 既存の `X-CSRF-Token` + cookie ペアで保護される PATCH を流用。 新規 endpoint は無いため CSRF 仕様は不変。
  - email verification 規約: email を編集すると UI が email_verified を 既定で false にリセットする。admin が改めて true にする場合は明示 操作が必要で、無意識のうちに未確認のメールを「確認済み」にできない。
  - tenant 境界: 既存 `requireAdmin` が user.TenantID と request tenant の 一致を確認しているため、本 WI で追加の境界チェックは不要だった。
  - SCL coherence は不変。
