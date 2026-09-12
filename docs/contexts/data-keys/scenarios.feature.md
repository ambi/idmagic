# Feature: DataKeys のシナリオ

## Rule: REQ-DATAKEYS-001 テナントの初回利用時に DEK を生成する

Primary actor: `System`

### Example: EX-DATAKEYS-001-01 通常経路

- Given テナント "tenant-a" にまだ DataEncryptionKey が存在しない
- When テナント "tenant-a" に対して BootstrapTenantDataKey を呼ぶ
- Then バージョン 1 の DataEncryptionKey が `active` として生成される
- Then `wrapped_dek` だけが永続化され、平文の DEK はどこにも残らない

### Example: EX-DATAKEYS-001-02 MasterKey プロバイダーに到達できない

- Given テナント "tenant-a" にまだ DataEncryptionKey が存在しない
- When テナント "tenant-a" に対して BootstrapTenantDataKey を呼ぶ
- But MasterKey プロバイダーに到達できない
- Then BootstrapTenantDataKey が DataKeyUnavailableError で失敗し、フェイルクローズのままテナントに DEK を作成しない

## Rule: REQ-DATAKEYS-002 DEK をローテーションしても既存の暗号文を復号できる

Primary actor: `System`

### Example: EX-DATAKEYS-002-01 通常経路

- Given テナント "tenant-a" に `active` のバージョン 1 の DataEncryptionKey があり、それで暗号化された EncryptedSecret が存在する
- When テナント "tenant-a" に対して RotateTenantDataKey を呼ぶ
- Then バージョン 2 が `active` になり、バージョン 1 は `retiring` に遷移する
- Then バージョン 1 で暗号化済みの既存 EncryptedSecret は、バージョン 1 が `retiring` である間、引き続き復号できる

## Rule: REQ-DATAKEYS-003 retiring の DEK を即時にロックアウトできる

Primary actor: `System`

### Example: EX-DATAKEYS-003-01 通常経路

- Given テナント "tenant-a" のバージョン 1 が `retiring` である
- When テナント "tenant-a" のバージョン 1 に対して DisableTenantDataKey を呼ぶ
- Then バージョン 1 が `disabled` に遷移する
- Then バージョン 1 で暗号化された EncryptedSecret の以後の復号リクエストは DataKeyUnavailableError でフェイルクローズに拒否される

## Rule: REQ-DATAKEYS-004 active の DEK は直接 disable できない

Primary actor: `System`

### Example: EX-DATAKEYS-004-01 通常経路

- Given テナント "tenant-a" のバージョン 2 が `active` である
- When テナント "tenant-a" のバージョン 2 に対して DisableTenantDataKey を呼ぶ
- Then DisableTenantDataKey は InvalidRequestError で拒否され、バージョン 2 は `active` のままである

## Rule: REQ-DATAKEYS-005 すべての参照を再暗号化した後に DEK を destroy できる

Primary actor: `System`

### Example: EX-DATAKEYS-005-01 通常経路

- Given テナント "tenant-a" のバージョン 1 が `retiring` で、Jobs 経由の再暗号化ジョブによって、バージョン 1 への参照がすべてバージョン 2 へ移行済みである
- When テナント "tenant-a" のバージョン 1 に対して DestroyTenantDataKey を呼ぶ
- Then バージョン 1 が `destroyed` に遷移し、`wrapped_dek` が破棄される
- Then バージョン 1 による復号は恒久的にできなくなる

### Example: EX-DATAKEYS-005-02 登録済みの参照元 Context に未移行の参照が残っている

- Given テナント "tenant-a" のバージョン 1 が `retiring` で、Jobs 経由の再暗号化ジョブによって、バージョン 1 への参照がすべてバージョン 2 へ移行済みである
- When テナント "tenant-a" のバージョン 1 に対して DestroyTenantDataKey を呼ぶ
- But 登録済みの参照元 Context に未移行の参照が残っている
- Then DestroyTenantDataKey が DataKeyStillReferencedError で拒否され、バージョン 1 は `retiring` のままである

## Rule: REQ-DATAKEYS-006 制御面主体はテナント横断で DEK の健全性を一覧できる

Primary actor: `SystemAdministrator`

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
