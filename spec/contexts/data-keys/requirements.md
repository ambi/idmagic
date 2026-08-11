# DataKeys Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-DATAKEYS-001: テナント初回利用時にDEKがbootstrapされる
- Actor: System
- Given: テナント "tenant-a" にまだ DataEncryptionKey が存在しない
- Then: テナント "tenant-a" に対して BootstrapTenantDataKey を呼ぶ
- Then: version 1 の DataEncryptionKey が active として生成される
- Then: wrapped_dek のみが永続化され平文 DEK はどこにも残らない
- Alternative (MasterKey provider に到達できない): BootstrapTenantDataKey が DataKeyUnavailableError で失敗し fail-closed のままテナントに DEK が作られない

### REQ-DATAKEYS-002: DEKをrotationしても既存暗号文が復号できる
- Actor: System
- Given: テナント "tenant-a" に active な version 1 の DataEncryptionKey があり、それで暗号化された EncryptedSecret が存在する
- Then: テナント "tenant-a" に対して RotateTenantDataKey を呼ぶ
- Then: version 2 が active になり version 1 は retiring に遷移する
- Then: version 1 で暗号化済みの既存 EncryptedSecret は version 1 が retiring である間 引き続き復号できる

### REQ-DATAKEYS-003: retiringのDEKを即時ロックアウトできる
- Actor: System
- Given: テナント "tenant-a" の version 1 が retiring である
- Then: テナント "tenant-a" の version 1 に対して DisableTenantDataKey を呼ぶ
- Then: version 1 が disabled に遷移する
- Then: version 1 で暗号化された EncryptedSecret の以後の復号要求は DataKeyUnavailableError で fail-closed に拒否される

### REQ-DATAKEYS-004: activeなDEKは直接disableできない
- Actor: System
- Given: テナント "tenant-a" の version 2 が active である
- Then: テナント "tenant-a" の version 2 に対して DisableTenantDataKey を呼ぶ
- Alternative (対象 version が active である): DisableTenantDataKey が InvalidRequestError で拒否され version 2 は active のままである

### REQ-DATAKEYS-005: 全参照の再暗号化後にDEKをdestroyできる
- Actor: System
- Given: テナント "tenant-a" の version 1 が retiring で、Jobs 経由の再暗号化 job が version 1 参照をすべて version 2 へ移行済みである
- Then: テナント "tenant-a" の version 1 に対して DestroyTenantDataKey を呼ぶ
- Then: version 1 が destroyed に遷移し wrapped_dek が破棄される
- Then: version 1 での復号は恒久的に不能になる
- Alternative (登録済みの所有 context に未移行の参照が残っている): DestroyTenantDataKey が DataKeyStillReferencedError で拒否され version 1 は retiring のままである

### REQ-DATAKEYS-006: systemAdminがテナント横断でDEK健全性を一覧する
- Actor: System
- Given: 複数テナントにそれぞれ DataEncryptionKey が存在する
- Then: system_admin が ListTenantDataKeyHealth を呼ぶ
- Then: 各テナントの active_version / status / provider 到達性が鍵材料を含まずに返る
- Alternative (呼び出し元が system_admin ではない): ListTenantDataKeyHealth が AccessDeniedError で拒否される

### REQ-DATAKEYS-007: BootstrapTenantDataKey
テナントの最初の DataEncryptionKey (version 1) を生成し、MasterKey provider で wrap して active にする内部インターフェース。provider 到達不能時は fail-closed で失敗する。
- Postcondition: active_key_count(input.tenant_id) <= 1

### REQ-DATAKEYS-008: RotateTenantDataKey
テナントに新しい DataEncryptionKey (version + 1) を生成して active に切り替え、旧 active を retiring に遷移させる。旧バージョンは即座に destroy せず、復号は継続できる。
- Precondition: tenant_has_active_data_key(input.tenant_id)
- Postcondition: active_key_count(input.tenant_id) <= 1

### REQ-DATAKEYS-009: DisableTenantDataKey
回転済み (retiring) の DataEncryptionKey 1 本を即時ロックアウトする。危殆化対応など、destroy (crypto-shredding) の前に先行して復号を止めたい場合に使う。active な version はこの操作の対象にできない (先に RotateTenantDataKey で退避させる)。
- Precondition: data_key_is_not_active(input.tenant_id, input.version)

### REQ-DATAKEYS-010: DestroyTenantDataKey
retiring または disabled の DataEncryptionKey 1 本の wrapped_dek を破棄し destroyed (crypto-shredding) にする。所有 context ごとに登録された再暗号化 job (Jobs 経由の data_key_reencryption) が全参照を active バージョンへ移行済みであることを呼び出し前に検証し、未移行の参照が残っていれば DataKeyStillReferencedError で拒否する (fail-closed)。不可逆。
- Precondition: data_key_is_not_active(input.tenant_id, input.version)
- Precondition: no_pending_reencryption_references(input.tenant_id)

### REQ-DATAKEYS-011: ListTenantDataKeyHealth
system_admin がテナント横断で DEK 健全性 (active version / status / provider 到達性) を一覧する。鍵材料は返さない。fail-closed 判定や rotation 漏れの検知に使う。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| DataEncryptionKey | レコード単位の可逆秘密を Tink AEAD で直接暗号化/復号するテナントスコープの対称鍵 (DEK)。 | DEK |
| MasterKey | DEK を wrap (エンベロープ暗号化) する KMS 側の鍵。provider (OpenBao Transit 互換、または dev/local の Tink cleartext keyset) が custody を持ち、アプリ DB には平文で残らない。 |  |
| Wrap | MasterKey で DEK を暗号化し、永続化可能な wrapped_dek にする操作。 | wrap |
| Active | 新規の暗号化操作に使われる、テナントにつき高々1本の DataEncryptionKey の状態。 |  |
| Retiring | 新規暗号化には使われないが、既存 ciphertext の復号にはまだ使われる状態。再暗号化 job (Jobs 経由) が全参照を移行するまで維持する。 |  |
| Disabled | 鍵材料の危殆化などにより手動で即時ロックアウトされた状態。この鍵で暗号化された ciphertext の復号は以後 fail-closed で拒否される。 | disabled |
| Destroyed | wrapped_dek を破棄した終端状態。復号は恒久的に不能になる (crypto-shredding の前提)。不可逆。 |  |
| FailClosed | unwrap 失敗・provider 到達不能・AAD/tamper 不一致など復号不能な場合に、平文へのフォールバックをせずアクセスを拒否する方針。 |  |
| System | DataKeys のライフサイクル usecase と再暗号化 job そのものを指す、人間の操作者を伴わない技術的な主体。 |  |

## State machines

### DataEncryptionKeyLifecycle

bootstrap で active として生成される。rotation で新バージョンが active になると旧バージョンは retiring へ遷移する。retiring は disable で即時ロックアウトでき、retiring/disabled は全参照の再暗号化 (Jobs 経由) 確認後にのみ destroy で終端 (destroyed) へ進む。active を直接 disable/destroy することはできない (先に rotation で退避させる)。

Initial: `active`  
Terminal: `destroyed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| active | DataEncryptionKeyRotated | "" | retiring |  |
| retiring | DataEncryptionKeyDisabled | "" | disabled |  |
| retiring | DataEncryptionKeyDestroyed | "" | destroyed |  |
| disabled | DataEncryptionKeyDestroyed | "" | destroyed |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
