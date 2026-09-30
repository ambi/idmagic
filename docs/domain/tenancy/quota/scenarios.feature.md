# Feature: リソース上限のシナリオ

## 入力

### Rule: REQ-TENANCY-012 System管理者はテナントのクォータ上限を調整できる

Primary actor: `SystemAdministrator`

#### Example: EX-TENANCY-012-01 通常経路

- Given system_admin ロールを持つ "sysadmin" が認証済みである
- When "sysadmin" が UpdateTenantQuota を呼び出しユーザー上限を 20000 に増やす
- Then 対象テナントの quota.users が 20000 になる

## 拒否

### Rule: REQ-TENANCY-013 Hard Quota を超過したリソース作成は拒否される

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-013-01 通常経路

- Given 対象テナントの groups 上限が 1000、利用量が 1000 である
- When テナント内管理者が新しい Group を作成しようとする
- Then QuotaExceededError で拒否され作成されない
