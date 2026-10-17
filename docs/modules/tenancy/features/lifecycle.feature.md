# Feature: テナントのライフサイクルの例

## Rule: REQ-TENANCY-003 default テナントは起動時に自動作成され削除も無効化もできない

### Example: EX-TENANCY-003-01 通常経路

- When IdP を起動する
- Then テナント "default" が status=Active で存在する

### Example: EX-TENANCY-003-02 default テナントの削除を試みる

- When IdP を起動する
- Then default テナントの削除を試みる
- Then default テナントを削除する API は提供されない

### Example: EX-TENANCY-003-03 default テナントの無効化を試みる

- When IdP を起動する
- Then default テナントの無効化を試みる
- Then default テナントの disable は InvalidRequestError で拒否される

## Rule: REQ-TENANCY-025 System 管理者が作成するテナントの realm は、予約されていない単一の DNS ラベルである

### Background:

- Given system_admin ロールを持つ "sysadmin" がデフォルトテナントで認証済みである

### Example: EX-TENANCY-025-01 通常経路

- When "sysadmin" が realm " acme " と表示名 " Acme " でテナントを作成する
- Then realm "acme"、表示名 "Acme"、status=active、endpoint_style=path のテナントが 201 で返る
- And "TenantCreated" が発行される

### Example: EX-TENANCY-025-02 予約された realm を指定する

- When "sysadmin" が realm "login" でテナントを作成する
- Then `invalid_request` の 400 で拒否され、テナントは作成されず、イベントも発行されない

### Example: EX-TENANCY-025-03 ハイフンで終わる realm を指定する

- When "sysadmin" が realm "acme-" でテナントを作成する
- Then `invalid_request` の 400 で拒否され、テナントは作成されない

### Example: EX-TENANCY-025-04 `xn--` で始まる realm を指定する

- When "sysadmin" が realm "xn--acme" でテナントを作成する
- Then `invalid_request` の 400 で拒否され、テナントは作成されない

### Example: EX-TENANCY-025-05 既存のテナントと同じ realm を指定する

- Given realm "acme" のテナントが存在する
- When "sysadmin" が realm "acme" でテナントを作成する
- Then `tenant_conflict` の 409 で拒否され、テナントは増えず、イベントも発行されない

## Rule: REQ-TENANCY-014 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない

### Example: EX-TENANCY-014-01 通常経路

- Given "operator" は admin ロールのみを持ち system_admin ロールを持たない
- When "operator" が ListTenants を呼び出す
- Then AccessDeniedError で拒否される

## Rule: REQ-TENANCY-026 テナント一覧は id の昇順で、各テナントの上限と使用量を添えて返す

### Example: EX-TENANCY-026-01 通常経路

- Given system_admin ロールを持つ "sysadmin" がデフォルトテナントで認証済みである
- And デフォルトテナントのほかに二つのテナントが存在する
- When "sysadmin" が ListTenants を呼び出す
- Then 三つのテナントが `id` の昇順で返り、それぞれが `quota` と `usage` を持つ

### Example: EX-TENANCY-026-02 上限を読み取れないテナントがある

- Given system_admin ロールを持つ "sysadmin" がデフォルトテナントで認証済みである
- And 上限の読み取りが失敗する
- When "sysadmin" が ListTenants を呼び出す
- Then 一覧は 200 で返り、各テナントは `quota` を持たない

## Rule: REQ-TENANCY-027 無効化と再開は、すでにその状態にあるテナントにも成功する

### Example: EX-TENANCY-027-01 無効なテナントをもう一度無効化する

- Given system_admin ロールを持つ "sysadmin" がデフォルトテナントで認証済みである
- And テナント "acme" は無効化されている
- When "sysadmin" がテナント "acme" を無効化する
- Then 204 が返り、`disabled_at` は一度目の要求の時刻のまま変わらず、"TenantDisabled" がもう一度発行される

### Example: EX-TENANCY-027-02 存在しない realm を無効化する

- Given system_admin ロールを持つ "sysadmin" がデフォルトテナントで認証済みである
- When "sysadmin" が存在しない realm "ghost" を無効化する
- Then `tenant_not_found` の 404 が返り、イベントは発行されない
