---
status: completed
archived_from: f3d58c5966dd744ef545123541072e551b535c13
---

# IdMagic の基盤・identity 系 context を SCL 3.0 に移行する

- System / Tenancy / IdentityManagement / Authentication / Audit / Jobs の6 context を SCL 3.0
(ADR-103) へ意味移行した。旧 `invariants` / `permissions` / `user_experience` を廃止し、
field/model constraints、interface `requires`/`ensures`、`states` transition guard/effect、
`authorization.principals`/`policies` + 全 interface 明示 `access`、`scenarios`
(`actor`/`given`/`main_success`/`extensions`)、`flows` (navigation only) へ再分類した。
- 反復する admin/system_admin/self-service 条件を各 context 内で `TenantAdministrator` /
`SystemAdministrator` / `AuthenticatedSelf` principal + policy へ集約した。Authentication は
旧 `permissions` が存在しなかったため、全25 interface の access を新規に設計した
(public: login/MFA/password-reset 系、AuthenticatedSelf: self-service 系、
TenantAdministrator: admin 系)。
- retention/TTL/password policy/runtime hardening/i18n tooling など SLO でない旧 `objectives` を
ADR-105/106/107 に保存し、値そのものは変更していない。Tenancy に誤配置されていた監査ログ
scenario は Audit へ、Authentication に混入していた IdentityManagement/Tenancy 所有の
self-service・RBAC・password-policy scenario 6件は IdentityManagement/Tenancy へ移し替えた
(挙動は変更せず、所有 context だけを訂正)。
- 全 88 interface (System 1 / Tenancy 16 / IdentityManagement 36 / Authentication 25 / Audit 4 /
Jobs 6) に明示的 `access` を付与した。

全文は `git show f3d58c5966dd744ef545123541072e551b535c13:<このファイルのパス>` で読む。
