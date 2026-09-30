# Feature: テナントのライフサイクルのシナリオ

## 生成

### Rule: REQ-TENANCY-003 default テナントは起動時に自動作成され削除も無効化もできない

Primary actor: `System`

#### Example: EX-TENANCY-003-01 通常経路

- When IdP を起動する
- Then テナント "default" が status=Active で存在する

#### Example: EX-TENANCY-003-02 default テナントの削除を試みる

- When IdP を起動する
- Then default テナントの削除を試みる
- Then default テナントを削除する API は提供されない

#### Example: EX-TENANCY-003-03 default テナントの無効化を試みる

- When IdP を起動する
- Then default テナントの無効化を試みる
- Then default テナントの disable は InvalidRequestError で拒否される

## 利用

### Rule: REQ-TENANCY-014 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-014-01 通常経路

- Given "operator" は admin ロールのみを持ち system_admin ロールを持たない
- When "operator" が ListTenants を呼び出す
- Then AccessDeniedError で拒否される
