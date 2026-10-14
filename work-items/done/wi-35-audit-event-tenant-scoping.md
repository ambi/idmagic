---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 監査イベントに emit 時点で tenant_id を載せ、テナント所属 admin の監査ビューに出るようにする

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
