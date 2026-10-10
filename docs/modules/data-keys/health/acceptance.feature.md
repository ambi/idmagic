# Feature: DEK の健全性の一覧の例

## Rule: REQ-DATAKEYS-006 制御面主体はテナント横断で DEK の健全性を一覧できる

### Example: EX-DATAKEYS-006-01 通常経路

- Given 複数テナントにそれぞれ DataEncryptionKey が存在する
- And "sys-operator" は制御面テナントに所属し、有効ロールに `system_admin` を含む有効な User である
- When "sys-operator" が制御面テナントの経路で ListTenantDataKeyHealth を呼ぶ
- Then 各テナントの `active_version`、`status`、プロバイダーへの到達性が、鍵素材を含まずに返る

### Example: EX-DATAKEYS-006-02 呼び出し元が `system_admin` を持たない

- Given 複数テナントにそれぞれ DataEncryptionKey が存在する
- And "sys-operator" は制御面テナントに所属し、有効ロールに `system_admin` を含む有効な User である
- When "sys-operator" が制御面テナントの経路で ListTenantDataKeyHealth を呼ぶ
- But 呼び出し元が `system_admin` を持たない
- Then AccessDeniedError で拒否される

### Example: EX-DATAKEYS-006-03 呼び出し元が `system_admin` を持つが制御面テナントの所属ではない

- Given 複数テナントにそれぞれ DataEncryptionKey が存在する
- And "sys-operator" は制御面テナントに所属し、有効ロールに `system_admin` を含む有効な User である
- When "sys-operator" が制御面テナントの経路で ListTenantDataKeyHealth を呼ぶ
- But 呼び出し元が `system_admin` を持つが制御面テナントの所属ではない
- Then AccessDeniedError で拒否され、レスポンスは他テナントの識別子も DEK の状態も含まず、テナント横断の健全性収集も実行されない
