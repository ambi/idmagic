---
id: idp-wi-35-audit-event-tenant-scoping
title: "監査イベントに emit 時点で tenant_id を載せ、テナント所属 admin の監査ビューに出るようにする"
created_at: 2026-06-20
authors: ["tn"]
status: completed
risk: medium
---
# Motivation
ログイン (UserAuthenticated) をはじめとする認証・ID 系のドメインイベント
は emit され、`AuditEventRepository` にも append されている。しかし
**テナント所属の admin の監査画面には出てこない**。原因は以下の連鎖:

1. `UserAuthenticated` / `AuthenticationFailed` / `LoginThrottled` /
   `PasswordChanged` / `PasswordResetRequested` および user/group
   ライフサイクルイベントは payload に `tenantId` を持たない
   (spec/scl.yaml の event 定義、internal/spec/events.go の struct とも)。
2. その結果 `newAuditEventRecord` (internal/bootstrap/audit_event_record.go)
   は `payload["tenantId"]` を拾えず、監査レコードの `tenant_id` を
   **空文字**で保存する。現状 tenant_id を持つのは Tenant ライフサイクル系
   (TenantCreated/Updated/Disabled/Enabled) のみ。
3. 一覧フィルタ `auditEventMatches`
   (internal/adapters/persistence/memory/audit_event_store.go) は
   `!AllTenants && q.TenantID != "" && rec.TenantID != q.TenantID` で、
   tenant_id 空のレコードをテナント絞り込みクエリから除外する。
4. ListAdminAuditEvents は actor の所属テナントで絞る
   (admin_audit_event_handler.go: parseAuditEventQuery)。`all_tenants=true`
   の横断ビューは `system_admin` かつ default テナント在籍のときだけ。

よって seed される admin alice (role `admin`・tenant `default`・
internal/bootstrap/seed.go) は、ログインイベントを画面で一切確認できない。
確認できるのは「default テナント在籍の `system_admin` が all_tenants を
ON にした横断ビュー」のときだけ、という実装と意図のズレがある
(type フィルタの placeholder が "UserAuthenticated" なのに、通常の
テナント admin ビューでは決して引っかからない)。

本 WI は「emit 時点で tenant_id を載せる」方針でこのズレを解消する。
フィルタ側を「空 tenant_id = グローバルとして所属テナント admin にも
見せる」に変える対案 (out_of_scope) は採らない。イベントは必ず
あるテナントの文脈で発生しており、その帰属を payload に明示するのが
ID プロバイダとして自然で、cross-tenant 漏洩の懸念も小さいため。

# Scope
- **scl**: spec/scl.yaml の event 定義に `tenantId: { type: String }` を追加する: UserAuthenticated / AuthenticationFailed / LoginThrottled / PasswordChanged / PasswordResetRequested / UserCreated / UserUpdated / UserDisabled / UserEnabled / UserDeleted / GroupCreated / GroupUpdated / GroupDeleted / GroupMemberAdded / GroupMemberRemoved。 Tenant ライフサイクル系は既に tenantId を持つので変更しない。, これらは Authentication / identity コンポーネントが owns する event。 payload のフィールド順は既存の occurredAt を先頭に保ち、tenantId は 意味的に近い位置 (sub / actorSub の隣) に置く。AuditEventResponse の tenant_id 定義 (既存) は変更不要。
- **go**: internal/spec/events.go の対象 event struct に `TenantID string \`json:"tenantId"\`` を追加する。SCL の twin 定義と一致させる (events_test.go の wire 名規約に従い json タグは tenantId)。, emit 各所で tenant_id を渡す。tenant コンテキストの取得元: - authorize_handler.go: UserAuthenticated は `user.TenantID`。
  AuthenticationFailed (writeAuthenticationFailed 経路) と
  LoginThrottled は user 未解決でも `requestTenantID(c)`
  (tenant_middleware.go) でリクエストパスから取得。
- change_password.go / reset_password_with_token.go /
  request_password_reset.go: 対象 user の TenantID。リセット要求の
  anti-enumeration を壊さないよう、user 不在時も requestTenantID 相当の
  テナントを載せる (PII を増やさない範囲で)。
- admin_users.go / admin_groups.go: actor もしくは target の
  TenantID を載せる (同一テナント内操作のため一致する)。, newAuditEventRecord は変更不要。既存の `payload["tenantId"]` 抽出が そのまま効くようになる (この WI の狙いはまさにそこ)。, AuthenticationFailed / LoginThrottled は username/IP を PII/ハッシュ で扱う既存方針を維持する。tenant_id は PII ではないのでそのまま平文。
- **ui**: 変更なし。AdminAuditEventsPage / AdminDashboardPage は既存のまま、 テナント絞り込みクエリで自然にイベントが返るようになる。 ダッシュボードの "24h audit events" カードも、これまで空だった テナント admin ビューで実数が出るようになる (副次的改善)。
- **documentation**: README には新規記述を増やさない。挙動の意味的差分は本 WI の completion.semantic_diff に記録する。

# Out of Scope
- フィルタ semantics の変更 (空 tenant_id を「グローバル」として所属 テナント admin にも見せる対案)。本 WI は emit 時 tenant_id 付与で 解決し、フィルタ (auditEventMatches / parseAuditEventQuery) は触らない。
- oauth2 / token フロー系イベントへの tenant_id 付与: AccessTokenIssued / RefreshTokenIssued/Rotated/ReuseDetected / AuthorizationCodeIssued/Redeemed / TokenRevoked / TokenIntrospected / DeviceAuthorization* / PARStored / SigningKeyRotated / ClientRegistered / AdminClientCreated/Updated/Deleted / ConsentGranted/Revoked。同じギャップだが tenant を usecase 層へ plumbing する範囲が広く、高頻度イベントの量も論点になるため別 WI に 切り出す。本 WI は Authentication + identity (user/group) に限定する。
- 既存 (legacy) の tenant_id 空レコードの遡及補正 / バックフィル。 監査ログは append-only の履歴であり、過去レコードは書き換えない。
- ログイン履歴の属性拡張 (IP / UA / session_id / GeoIP / MFA 段階別)。 これは [[wi-20-authentication-event-history]] の領分。本 WI は既存 イベントの「テナント帰属の可視化」だけに絞る。
- end user 向けサインイン履歴画面。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動 1 (login 可視化): seed の alice (admin / tenant default) で ログイン → `/admin/audit_events` を type=UserAuthenticated で絞ると 自分のログインイベントが出ること。これまで空だったこととの差分を確認。
- 手動 2 (失敗ログイン): 存在しない username / 誤パスワードでログイン 失敗 → AuthenticationFailed が同テナントの監査ビューに出ること。 閾値到達で LoginThrottled も出ること。
- 手動 3 (cross-tenant 隔離): 2 テナント構成で、テナント A の admin が テナント B のログインイベントを見られないこと (all_tenants OFF)。
- 手動 4 (dashboard): `/admin` の "24h audit events" カードが、テナント admin ビューで実数を表示すること (従来 0 になりがちだった)。

# Risk Notes
event struct と scl.yaml を twin で触るため、片方だけ更新すると
spec↔impl drift になる。events_test.go の wire 名検証と coherence_test を
両方緑にして担保する。

emit 各所で「正しいテナント」を渡すのが肝。特に認証失敗・スロットリングは
user が未解決のケースがあり、requestTenantID(c) (リクエストパス由来) に
フォールバックする。誤って空や default 固定を載せると、別テナントの失敗が
default に紛れ込む / 自テナントで見えない、という新たなズレを生む。
マルチテナント環境を想定した手動 3 のクロステナント隔離確認を必須にする。

本 WI は Authentication + identity に絞るため、完了後も oauth2 / token /
consent / client 系イベントは依然テナント admin ビューに出ない。これは
意図した境界であり、follow-up WI で同じ方針 (emit 時 tenant_id) を適用する。
境界を completion で明示し、誤って「全イベントが見えるようになった」と
誤認されないようにする。

# Completion
- **Completed At**: 2026-06-20
- **Summary**:
  認証・ID 系の 15 ドメインイベントに tenantId を追加し、emit 時点で
  所属テナントを載せるようにした。これにより、これまで tenant_id 空で
  記録されてテナント所属 admin の監査ビューから除外されていたログイン等の
  イベントが、自テナントの `/admin/audit_events` に出るようになった。
  フィルタ (auditEventMatches / parseAuditEventQuery) は方針どおり一切
  変更していない。newAuditEventRecord の既存の payload["tenantId"] 抽出が
  そのまま効くようになっただけ。

  対象イベント: UserAuthenticated / AuthenticationFailed / LoginThrottled /
  PasswordChanged / PasswordResetRequested / UserCreated / UserUpdated /
  UserDisabled / UserEnabled / UserDeleted / GroupCreated / GroupUpdated /
  GroupDeleted / GroupMemberAdded / GroupMemberRemoved。

  tenant の取得元: ログイン成功は user.TenantID、認証失敗・スロットリング・
  リセット要求は user 未解決でも requestTenantID(c) / tenancy.TenantID(ctx)
  (リクエストパス由来)、パスワード変更は対象 user.TenantID、admin の
  user/group 操作は対象 aggregate の TenantID (actor と一致)。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
