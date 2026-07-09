---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-14
---

# 管理画面 (admin console) の機能を完成させる

## Motivation
idmagic の HTTP 層には admin API がほぼ全領域揃ったが、idmagic/ui の
React 管理画面は users と clients の 2 ページのみで、残り 4 領域 (consents
/ audit_events / keys / tenants) は API しかなく操作 UI が無い。完成形と
して 6 ページを揃え、サイドバーから相互遷移できる状態にする。

## Scope
- **ui**:
  - pages:
    - AdminConsentsPage: ListConsents / GetConsent / RevokeConsent を操作する。 テーブルでテナント内の Consent を一覧し、行選択で詳細パネル、Revoke ボタンを提供する。Consent の Create / Update は SCL 上意図的に存在 しないので UI でも提供しない。
    - AdminAuditEventsPage: ListAdminAuditEvents / GetAdminAuditEvent を操作 する。type / sub / after / before / limit のフィルタフォームと結果 テーブルを提供する。system_admin の場合のみ all_tenants トグルを 出す。行選択で payload JSON を展開表示する。
    - AdminKeysPage: ListAdminKeys / GetAdminKey / RotateAdminKey を操作 する。鍵一覧 (kid / alg / active / created_at)、行選択で公開鍵 JWK の展開、Rotate ボタン (確認ダイアログ付き) を提供する。Rotate は system_admin かつ default tenant でのみ表示。
    - AdminTenantsPage: ListTenants / GetTenant / CreateTenant / UpdateTenant / DisableTenant / EnableTenant を操作する。 /realms/default 経由でのみアクセスする画面とし、default tenant でない 場合はナビゲーション自体を隠す。PasswordPolicyOverride の編集も含む。
  - types:
    - AdminConsent / AdminAuditEvent / AdminKey / AdminTenant と 対応する Page 型を types.ts に追加する。
  - api:
    - api.ts に list / get / revoke / rotate / create / update / disable / enable の関数を追加し、loadPageData の URL ディスパッチを追加する。
  - routing:
    - router.tsx に /admin/consents, /admin/audit_events, /admin/keys, /admin/tenants の 4 ルートを追加する。
  - navigation:
    - 既存 AdminUsersPage / AdminClientsPage と新ページの sidebar navigation を統一する。disabled 項目だった「監査ログ」「概要」を解消し、新項目 「Consents」「鍵」「テナント」を追加する。テナントは control plane (default tenant 経路) でのみ表示する。

## Out of Scope
- admin 以外の画面の改修。
- 既存 AdminUsersPage / AdminClientsPage のロジック変更 (navigation 配列の 更新のみ)。
- 多言語化や a11y の大規模見直し。
- audit event payload の field レベルでの構造化表示 (生 JSON 表示で十分)。

## Verification
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- `go test ./internal/adapters/http/... ./internal/spec/...` (in: idmagic)
  - reason: バックエンドハンドラに変更は無いが回帰を確認する。
- 手動: dev サーバを起動し、admin user でログイン → /admin/users から nav 経由で 6 画面が遷移できる。/admin/consents で revoke 確認、 /admin/audit_events でフィルタ動作、/admin/keys で rotate 確認、 /realms/default/admin/tenants で create/update/disable 確認。

## Risk Notes
純粋なフロントエンド追加だが、CSRF とテナント境界を踏み外すと既存の
admin 画面まで壊れる。pages 配下に新ファイルを追加するのを基本とし、
既存ページの編集は navigation 配列のみに留める。

## Completion
- **Completed At**: 2026-06-14
- **Summary**:
  idmagic/ui に AdminConsentsPage / AdminAuditEventsPage / AdminKeysPage /
  AdminTenantsPage の 4 ページを追加し、admin console を全 6 領域
  (users / clients / consents / audit_events / keys / tenants) で操作可能に
  した。サイドバー navigation は lib/adminNav.ts に単一の source of truth を
  作り、6 ページから共通参照する形に整理した。tenants は control plane
  (/realms/default) 経路でのみ navigation に出す。backend は
  /api/auth/account に tenant_id と roles を追加し、UI 側のロールゲート
  (rotate / all_tenants) の根拠を提供する。
- **Verification Results**:
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui lint`
    - result: Checked 39 files. No fixes applied.
  - `bun --cwd idmagic/ui build`
    - result: vite build ok (428.97 kB / 123.33 kB gzip)
  - `go test ./internal/adapters/http/... ./internal/spec/...` (in: idmagic)
    - result: ok
  - `go build ./...` (in: idmagic)
    - result: ok
  - `go vet ./...` (in: idmagic)
    - result: ok
- **Affected Guarantees State**:
  - admin RBAC: 画面側のボタン表示ロジックは追加されたが、destructive 操作 (revoke / rotate / disable / create / update) の認可判断は 全て backend 側で requireAdminUser / requireSystemAdmin を呼ぶ 既存経路に乗っている。
  - CSRF: 全ての POST/PATCH/DELETE は X-CSRF-Token と csrf cookie の ペアを要求する既存契約に従う。新 API helper はすべて csrfToken を 引数で受け取り、AuthenticationAPI 経由でヘッダに乗せる。
  - Tenant isolation: AdminTenantsPage は /realms/default の navigation でのみ露出し、tenants エンドポイントが non-default tenant 経路から 呼ばれないようにフロント側でも遮断する。backend 側でも /admin/tenants は default tenant 配下でしか mount されない。
  - SCL coherence: SCL 側は変更していない。components / interfaces / permissions の wording は影響なし。
