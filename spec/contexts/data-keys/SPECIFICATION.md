---
context: data-keys
updated_at: 2026-08-15
---

# DataKeys Specification

## Overview

テナントごとの `DataEncryptionKey`（DEK）のライフサイクル（`bootstrap`、`rotate`、`disable`、`destroy`）とメタデータを所有する。DEK はマスターキープロバイダー（OpenBao Transit 互換。開発環境とローカル環境では Tink の平文鍵セット）でラップし、データベースに保存する可逆なシークレット（TOTP seed など）のエンベロープ暗号化に使う。

暗号処理そのものは所有しない。`EnvelopeCrypto` ポートとその実装は技術的な共有アダプター (`backend/shared/security`) に置き、この Context が外部へ公開するのは `EncryptedSecret` と鍵のライフサイクルメタデータだけである。署名鍵（`private_jwk`）も管理せず、`SigningKeys` の責務とする。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| DataEncryptionKey | レコード単位の可逆なシークレットを Tink AEAD で直接暗号化・復号する、テナントスコープの対称鍵 (DEK)。 | DEK |
| MasterKey | DEK をラップしてエンベロープ暗号化する KMS 側の鍵。プロバイダー (OpenBao Transit 互換、または開発環境とローカル環境で使う Tink の平文鍵セット) が管理し、アプリケーションデータベースには平文で残らない。 |  |
| Wrap | `MasterKey` で DEK を暗号化し、永続化できる `wrapped_dek` にする操作。 | wrap |
| Active | 新規の暗号化操作に使われる、テナントにつき高々1本の DataEncryptionKey の状態。 |  |
| Retiring | 新規の暗号化には使わないが、既存の暗号文の復号には引き続き使う状態。再暗号化ジョブ (`Jobs` 経由) がすべての参照を移行するまで維持する。 |  |
| Disabled | 鍵素材の危殆化などにより、手動で即時ロックアウトした状態。この鍵で暗号化された暗号文の復号は、それ以降フェイルクローズで拒否する。 | disabled |
| Destroyed | `wrapped_dek` を破棄した終端状態。復号は恒久的に不可能となり、暗号学的消去が成立する。不可逆である。 |  |
| FailClosed | アンラップの失敗、プロバイダーへの到達不能、AAD や改ざんの不一致などで復号できない場合に、平文へフォールバックせずアクセスを拒否する方針。 |  |
| System | `DataKeys` のライフサイクルユースケースと再暗号化ジョブそのものを指す、人間の操作者を伴わない技術的な主体。 |  |

## State Transitions

### DataEncryptionKeyLifecycle

`bootstrap` で `active` として生成する。ローテーションで新しいバージョンが `active` になると、旧バージョンは `retiring` へ遷移する。`retiring` は `disable` で即時にロックアウトでき、`retiring` または `disabled` の鍵は、すべての参照を再暗号化したことを `Jobs` 経由で確認した後に限り、`destroy` で終端状態の `destroyed` へ進む。`active` を直接 `disable` または `destroy` することはできず、先にローテーションで退避させる。

Initial: `active` Terminal: `destroyed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| active | DataEncryptionKeyRotated | — | retiring |  |
| retiring | DataEncryptionKeyDisabled | — | disabled |  |
| retiring | DataEncryptionKeyDestroyed | — | destroyed |  |
| disabled | DataEncryptionKeyDestroyed | — | destroyed |  |

## Authorization Boundary

DEK のライフサイクル操作 (`bootstrap`、`rotate`、`disable`、`destroy`) は HTTP に公開しない。同じプロセス内の内部インターフェースとしてのみ呼べるため、テナント管理者がこれらを直接起動する経路は存在しない。

唯一の管理 API である `ListTenantDataKeyHealth` はテナント横断の運用情報を返すため、`system_admin` ロールを持つユーザーだけが呼べる。応答には鍵素材を一切含めず、`active_version`、`status`、プロバイダーへの到達性に限る。

## Design

### Internal Interfaces

#### BootstrapTenantDataKey
テナントの最初の `DataEncryptionKey` (バージョン 1) を生成し、`MasterKey` プロバイダーでラップして `active` にする内部インターフェース。プロバイダーへ到達できない場合はフェイルクローズで失敗する。
- Result invariant: active_key_count(input.tenant_id) <= 1

#### RotateTenantDataKey
テナントに新しい `DataEncryptionKey` (バージョン + 1) を生成して `active` に切り替え、以前の `active` を `retiring` に遷移させる。旧バージョンは即座に `destroy` せず、引き続き復号できる。
- Input invariant: tenant_has_active_data_key(input.tenant_id)
- Result invariant: active_key_count(input.tenant_id) <= 1

#### DisableTenantDataKey
ローテーション済み (`retiring`) の `DataEncryptionKey` 1 本を即時にロックアウトする。危殆化対応など、`destroy` による暗号学的消去の前に復号を止める場合に使う。`active` なバージョンはこの操作の対象にできず、先に `RotateTenantDataKey` で退避させる。
- Input invariant: data_key_is_not_active(input.tenant_id, input.version)

#### DestroyTenantDataKey
`retiring` または `disabled` の `DataEncryptionKey` 1 本について、`wrapped_dek` を破棄して `destroyed` とし、暗号学的に消去する。所有する Context ごとに登録された再暗号化ジョブ (`Jobs` 経由の `data_key_reencryption`) が、すべての参照を `active` バージョンへ移行済みであることを呼び出し前に検証する。未移行の参照が残っていれば `DataKeyStillReferencedError` でフェイルクローズに拒否する。この操作は不可逆である。
- Input invariant: data_key_is_not_active(input.tenant_id, input.version)
- Input invariant: no_pending_reencryption_references(input.tenant_id)

## Scenarios

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
  - ALT 登録済みの所有 Context に未移行の参照が残っている → DestroyTenantDataKey が DataKeyStillReferencedError で拒否され、バージョン 1 は `retiring` のままである
- THEN バージョン 1 が `destroyed` に遷移し、`wrapped_dek` が破棄される
- THEN バージョン 1 による復号は恒久的にできなくなる

### REQ-DATAKEYS-006: システム管理者はテナント横断で DEK の健全性を一覧できる
- ACTOR System
- GIVEN 複数テナントにそれぞれ DataEncryptionKey が存在する
- WHEN `system_admin` が ListTenantDataKeyHealth を呼ぶ
  - ALT 呼び出し元が `system_admin` ではない → ListTenantDataKeyHealth が AccessDeniedError で拒否される
- THEN 各テナントの `active_version`、`status`、プロバイダーへの到達性が、鍵素材を含まずに返る
