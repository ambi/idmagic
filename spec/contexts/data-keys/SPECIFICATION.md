---
context: data-keys
updated_at: 2026-08-11
---

# DataKeys Specification

## Overview

テナントごとの DataEncryptionKey (DEK) のライフサイクル (bootstrap/rotate/
disable/destroy) とメタデータを所有する。DEK は master key provider (OpenBao Transit
互換、dev/local は Tink cleartext keyset) が wrap し、DB に残る可逆秘密 (TOTP シード等) の
エンベロープ暗号化に使う。EnvelopeCrypto port 自体は Go 側の
shared technical adapter に置き、SCL への露出は EncryptedSecret / 鍵ライフサイクルの
メタデータに留める。署名鍵 (private_jwk) の鍵管理は所有しない (SigningKeys の範囲)。

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

## State Transitions

### DataEncryptionKeyLifecycle

bootstrap で active として生成される。rotation で新バージョンが active になると旧バージョンは retiring へ遷移する。retiring は disable で即時ロックアウトでき、retiring/disabled は全参照の再暗号化 (Jobs 経由) 確認後にのみ destroy で終端 (destroyed) へ進む。active を直接 disable/destroy することはできない (先に rotation で退避させる)。

Initial: `active`
Terminal: `destroyed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| active | DataEncryptionKeyRotated | — | retiring |  |
| retiring | DataEncryptionKeyDisabled | — | disabled |  |
| retiring | DataEncryptionKeyDestroyed | — | destroyed |  |
| disabled | DataEncryptionKeyDestroyed | — | destroyed |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### BootstrapTenantDataKey
テナントの最初の DataEncryptionKey (version 1) を生成し、MasterKey provider で wrap して active にする内部インターフェース。provider 到達不能時は fail-closed で失敗する。
- Result invariant: active_key_count(input.tenant_id) <= 1

#### RotateTenantDataKey
テナントに新しい DataEncryptionKey (version + 1) を生成して active に切り替え、旧 active を retiring に遷移させる。旧バージョンは即座に destroy せず、復号は継続できる。
- Input invariant: tenant_has_active_data_key(input.tenant_id)
- Result invariant: active_key_count(input.tenant_id) <= 1

#### DisableTenantDataKey
回転済み (retiring) の DataEncryptionKey 1 本を即時ロックアウトする。危殆化対応など、destroy (crypto-shredding) の前に先行して復号を止めたい場合に使う。active な version はこの操作の対象にできない (先に RotateTenantDataKey で退避させる)。
- Input invariant: data_key_is_not_active(input.tenant_id, input.version)

#### DestroyTenantDataKey
retiring または disabled の DataEncryptionKey 1 本の wrapped_dek を破棄し destroyed (crypto-shredding) にする。所有 context ごとに登録された再暗号化 job (Jobs 経由の data_key_reencryption) が全参照を active バージョンへ移行済みであることを呼び出し前に検証し、未移行の参照が残っていれば DataKeyStillReferencedError で拒否する (fail-closed)。不可逆。
- Input invariant: data_key_is_not_active(input.tenant_id, input.version)
- Input invariant: no_pending_reencryption_references(input.tenant_id)

## Scenarios

### REQ-DATAKEYS-001: テナント初回利用時にDEKがbootstrapされる
- ACTOR System
- GIVEN テナント "tenant-a" にまだ DataEncryptionKey が存在しない
- WHEN テナント "tenant-a" に対して BootstrapTenantDataKey を呼ぶ
  - ALT MasterKey provider に到達できない → BootstrapTenantDataKey が DataKeyUnavailableError で失敗し fail-closed のままテナントに DEK が作られない
- THEN version 1 の DataEncryptionKey が active として生成される
- THEN wrapped_dek のみが永続化され平文 DEK はどこにも残らない

### REQ-DATAKEYS-002: DEKをrotationしても既存暗号文が復号できる
- ACTOR System
- GIVEN テナント "tenant-a" に active な version 1 の DataEncryptionKey があり、それで暗号化された EncryptedSecret が存在する
- WHEN テナント "tenant-a" に対して RotateTenantDataKey を呼ぶ
- THEN version 2 が active になり version 1 は retiring に遷移する
- THEN version 1 で暗号化済みの既存 EncryptedSecret は version 1 が retiring である間 引き続き復号できる

### REQ-DATAKEYS-003: retiringのDEKを即時ロックアウトできる
- ACTOR System
- GIVEN テナント "tenant-a" の version 1 が retiring である
- WHEN テナント "tenant-a" の version 1 に対して DisableTenantDataKey を呼ぶ
- THEN version 1 が disabled に遷移する
- THEN version 1 で暗号化された EncryptedSecret の以後の復号要求は DataKeyUnavailableError で fail-closed に拒否される

### REQ-DATAKEYS-004: activeなDEKは直接disableできない
- ACTOR System
- GIVEN テナント "tenant-a" の version 2 が active である
- WHEN テナント "tenant-a" の version 2 に対して DisableTenantDataKey を呼ぶ
- THEN DisableTenantDataKey は InvalidRequestError で拒否され version 2 は active のままである

### REQ-DATAKEYS-005: 全参照の再暗号化後にDEKをdestroyできる
- ACTOR System
- GIVEN テナント "tenant-a" の version 1 が retiring で、Jobs 経由の再暗号化 job が version 1 参照をすべて version 2 へ移行済みである
- WHEN テナント "tenant-a" の version 1 に対して DestroyTenantDataKey を呼ぶ
  - ALT 登録済みの所有 context に未移行の参照が残っている → DestroyTenantDataKey が DataKeyStillReferencedError で拒否され version 1 は retiring のままである
- THEN version 1 が destroyed に遷移し wrapped_dek が破棄される
- THEN version 1 での復号は恒久的に不能になる

### REQ-DATAKEYS-006: systemAdminがテナント横断でDEK健全性を一覧する
- ACTOR System
- GIVEN 複数テナントにそれぞれ DataEncryptionKey が存在する
- WHEN system_admin が ListTenantDataKeyHealth を呼ぶ
  - ALT 呼び出し元が system_admin ではない → ListTenantDataKeyHealth が AccessDeniedError で拒否される
- THEN 各テナントの active_version / status / provider 到達性が鍵材料を含まずに返る
