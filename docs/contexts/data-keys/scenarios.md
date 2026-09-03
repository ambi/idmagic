# DataKeys Scenarios

### REQ-DATAKEYS-001: テナントの初回利用時に DEK を生成する
- ACTOR System
- GIVEN テナント "tenant-a" にまだ DataEncryptionKey が存在しない
- WHEN テナント "tenant-a" に対して BootstrapTenantDataKey を呼ぶ
  - ALT MasterKey プロバイダーに到達できない → BootstrapTenantDataKey が DataKeyUnavailableError で失敗し、フェイルクローズのままテナントに DEK を作成しない
- THEN バージョン 1 の DataEncryptionKey が `active` として生成される
- THEN `wrapped_dek` だけが永続化され、平文の DEK はどこにも残らない

### REQ-DATAKEYS-002: DEK をローテーションしても既存の暗号文を復号できる
- ACTOR System
- GIVEN テナント "tenant-a" に `active` のバージョン 1 の DataEncryptionKey があり、それで暗号化された EncryptedSecret が存在する
- WHEN テナント "tenant-a" に対して RotateTenantDataKey を呼ぶ
- THEN バージョン 2 が `active` になり、バージョン 1 は `retiring` に遷移する
- THEN バージョン 1 で暗号化済みの既存 EncryptedSecret は、バージョン 1 が `retiring` である間、引き続き復号できる

### REQ-DATAKEYS-003: retiring の DEK を即時にロックアウトできる
- ACTOR System
- GIVEN テナント "tenant-a" のバージョン 1 が `retiring` である
- WHEN テナント "tenant-a" のバージョン 1 に対して DisableTenantDataKey を呼ぶ
- THEN バージョン 1 が `disabled` に遷移する
- THEN バージョン 1 で暗号化された EncryptedSecret の以後の復号リクエストは DataKeyUnavailableError でフェイルクローズに拒否される

### REQ-DATAKEYS-004: active の DEK は直接 disable できない
- ACTOR System
- GIVEN テナント "tenant-a" のバージョン 2 が `active` である
- WHEN テナント "tenant-a" のバージョン 2 に対して DisableTenantDataKey を呼ぶ
- THEN DisableTenantDataKey は InvalidRequestError で拒否され、バージョン 2 は `active` のままである

### REQ-DATAKEYS-005: すべての参照を再暗号化した後に DEK を destroy できる
- ACTOR System
- GIVEN テナント "tenant-a" のバージョン 1 が `retiring` で、Jobs 経由の再暗号化ジョブによって、バージョン 1 への参照がすべてバージョン 2 へ移行済みである
- WHEN テナント "tenant-a" のバージョン 1 に対して DestroyTenantDataKey を呼ぶ
  - ALT 登録済みの参照元 Context に未移行の参照が残っている → DestroyTenantDataKey が DataKeyStillReferencedError で拒否され、バージョン 1 は `retiring` のままである
- THEN バージョン 1 が `destroyed` に遷移し、`wrapped_dek` が破棄される
- THEN バージョン 1 による復号は恒久的にできなくなる

### REQ-DATAKEYS-006: 制御面主体はテナント横断で DEK の健全性を一覧できる
- ACTOR SystemAdministrator
- GIVEN 複数テナントにそれぞれ DataEncryptionKey が存在する
- GIVEN "sys-operator" は制御面テナントに所属し、有効ロールに `system_admin` を含む有効な User である
- WHEN "sys-operator" が制御面テナントの経路で ListTenantDataKeyHealth を呼ぶ
  - ALT 呼び出し元が `system_admin` を持たない → AccessDeniedError で拒否される
  - ALT 呼び出し元が `system_admin` を持つが制御面テナントの所属ではない → AccessDeniedError で拒否され、応答は他テナントの識別子も DEK の状態も含まず、テナント横断の健全性収集も実行されない
- THEN 各テナントの `active_version`、`status`、プロバイダーへの到達性が、鍵素材を含まずに返る
