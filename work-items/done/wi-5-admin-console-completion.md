---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面 (admin console) の機能を完成させる

idmagic/ui に AdminConsentsPage / AdminAuditEventsPage / AdminKeysPage /
AdminTenantsPage の 4 ページを追加し、admin console を全 6 領域
(users / clients / consents / audit_events / keys / tenants) で操作可能に
した。サイドバー navigation は lib/adminNav.ts に単一の source of truth を
作り、6 ページから共通参照する形に整理した。tenants は control plane
(/realms/default) 経路でのみ navigation に出す。backend は
/api/auth/account に tenant_id と roles を追加し、UI 側のロールゲート
(rotate / all_tenants) の根拠を提供する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
