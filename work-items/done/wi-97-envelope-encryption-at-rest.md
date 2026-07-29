---
depends_on: [wi-126-async-job-runner]
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-03
---

# 保存時のエンベロープ暗号化 (per-tenant DEK / KMS master key) を導入する

## Motivation
可逆な秘密の一部が DB に平文で保存されている。ハッシュ化できる一方向秘密
(パスワード / reset token / client_secret) は既にハッシュ化されているが、
復号が必要な可逆秘密は暗号化 or 外部 KMS が要る:

- 署名秘密鍵 `signing_keys.private_jwk` (JSONB。`KeyProvider=Local/Database`
  では平文のまま app DB に置かれる dev/test fallback。ADR-075 参照。本番の
  `VaultTransit` provider は対象外)。
- TOTP シード `mfa_factors.secret` (平文)。
- 将来の Token Vault ([[wi-55-token-vault-federated-connections]]) が預かる
  外部 API アクセストークン。

代表的な IdP は customer-managed key / BYOK を含む保存時暗号化を提供する
(Okta / Entra)。本 WI は per-tenant DEK を KMS の master key で包む
**エンベロープ暗号化**を導入し、テナント境界での鍵分離・鍵ローテーション・
fail-closed な復号を実現する。署名鍵そのものの鍵管理は
[[wi-32-kms-hsm-and-per-tenant-signing-keys]] が担い (ADR-075)、本 WI は
DB に残る可逆秘密を対象とする独立した `EnvelopeCrypto` port を持つ
(署名用 `transit/sign` とは操作もライフサイクルも異なるため `KeyStore` port
とは統合しない)。

AEAD/keyset の実装には自前のプリミティブを書かず [Tink](https://developers.google.com/tink)
を使う。master key の custody は provider 抽象の背後に置き、初期実装は
OpenBao (Apache-2.0、Vault Transit 互換 HTTP API を持つ Linux Foundation
プロジェクト) を想定する。HashiCorp Vault CE は 2023 年のライセンス変更
(BUSL 1.1) 以降 OSI 承認の OSS ではなくなっており、self-host OSS を default
stack とする方針 (ADR-075 の却下理由と同じ) には OpenBao の方が整合する。
`EnvelopeCrypto` port は provider を差し替え可能な抽象とし、OpenBao は
「初期実装の 1 つ」に過ぎない — 将来より優れた KMS が出た場合も port に
adapter を追加するだけで済み、上位層 (対象 repository) は無変更で済む
([[wi-32-kms-hsm-and-per-tenant-signing-keys]] の `KeyProvider` 抽象と同じ
パターン)。dev/local では Tink の cleartext keyset をそのまま使い、OpenBao
を立てなくても開発できる。署名鍵側 ([[wi-32-kms-hsm-and-per-tenant-signing-keys]]
の `VaultTransit` provider) を OpenBao へ移行するかどうかは別課題とし、
本 WI では扱わない。

[[ADR-054]] (Token Vault) は upstream token の暗号化について「新たな鍵管理
システムを増設せず既存の KMS を流用する」ことを決定している。本 WI が導入する
per-tenant DEK / `EnvelopeCrypto` がその「既存 KMS」の実体であり、
[[wi-55-token-vault-federated-connections]] は本 WI に依存してこれを流用する
(新規の鍵管理コンポーネントを追加しない)。

## Scope
- **decision**:
  - 新規 ADR: エンベロープ暗号化の設計。AEAD/keyset 実装に Tink を採用し自前の 暗号プリミティブを書かないこと、master key custody は provider 抽象 (`EnvelopeCrypto` port) の背後に置き差し替え可能とすること、初期実装は OpenBao (Vault Transit 互換、self-host OSS 方針との整合、ライセンス上の理由で HashiCorp Vault CE ではなく OpenBao を選ぶ判断根拠) とすること、per-tenant DEK を master key で暗号化保管し DEK で AEAD 暗号化すること、暗号化対象秘密の選定 (TOTP シード / Token Vault トークン / その他可逆秘密)、鍵 ID の付与、DEK / master の rotation、復号失敗時の fail-closed、Tink cleartext keyset による local/dev fallback、新規 bounded context (例: `DataKeys`。`EnvelopeCrypto` port 自体は shared technical adapter に置くが、TenantDataEncryptionKey の永続化・ライフサイクル usecase は独立 context に置く) の命名、wi-32 (署名鍵の KMS 化) との責務分担 (private_jwk は wi-32 が所有し、本 WI は DB に残る可逆秘密を対象。署名鍵側の Vault→OpenBao 移行は対象外)、[[ADR-054]] が求める「新規鍵管理システムを増設しない」要件をどう満たすか、BYOK / customer-managed key を将来拡張とする境界を記録する。
- **scl**:
  - 新規 bounded context (decision で命名) の §3.2 models: TenantDataEncryptionKey / EncryptedSecret (envelope: 鍵 ID + ciphertext + nonce) を追加する。暗号化は主に adapter 層の実装で、SCL への 露出は最小に留める。
  - §3.4 states/events: DataEncryptionKeyRotated を追加する。
  - 所有要素の constraints/contracts: 可逆秘密を平文で保存しない、DEK はテナント単位、復号不能時は アクセスを拒否する (fail-closed) ことを明示する。
  - 新規 bounded context を追加するため `ARCHITECTURE.md` / 隣接する `architecture.yaml` を同期する (CLAUDE.md のルール)。
- **go**:
  - Tink ベースの AEAD/keyset 実装と、master key custody 用の provider adapter (初期実装: OpenBao の Transit 互換 API。`backend/signingkeys/keys_vault/vault_transit_http.go` の HTTP クライアントパターンを流用し、encrypt/decrypt/datakey 操作を追加する) を持つ `EnvelopeCrypto` port を shared technical adapter に追加する。per-tenant DEK キャッシュは rotation/destroy 時に無効化できるようにする。
  - 対象 repository (mfa factor store / token vault 等) を暗号化対応にし、 migration で既存平文を暗号化へ再暗号化する。
- **http**:
  - 管理面は最小 (鍵状態 / 暗号化状態の可視化程度) に留める。
- **ui**:
  - AdminKeys / AdminSettings に暗号化・DEK 状態の表示を最小限追加する。
- **documentation**:
  - README に暗号化方針・KMS 設定・dev fallback・鍵紛失時の注意を追記する。

## Out of Scope
- 署名鍵 (private_jwk) 自体の鍵管理 ([[wi-32-kms-hsm-and-per-tenant-signing-keys]])。署名鍵側の provider を Vault Transit から OpenBao へ移行するかどうかも本 WI では扱わない。
- Postgres TDE / ディスク暗号化 (インフラ層の責務)。
- 転送時暗号化 (既存 TLS)。
- フルの BYOK / customer-managed key の管理 UI (将来拡張)。
- テナント丸ごと削除時の DEK 破棄による crypto-shredding / right-to-erasure。現状テナント丸ごと削除の WI が無いため、per-tenant DEK 設計がこれを将来容易にすることだけ記録し、実装は対象外とする。

## Plan
- 署名用KeyStore/Vault Transitはsign/verify用途なのでDEK wrapへ流用せず、`EnvelopeCrypto` port（GenerateDataKey/Wrap/Unwrap/Encrypt/Decrypt）をshared technical adapterに追加する。AEAD/keyset実装はTinkに委ね、自前でnonce/tag/AADを組み立てない。master key custodyはproviderで抽象化し、初期実装はOpenBao（Vault Transit互換API。既存`keys_vault`のHTTPクライアントパターンを流用しencrypt/decrypt/datakeyを追加）、dev/localはTinkのcleartext keysetとする。providerは差し替え可能な抽象に留め、特定KMSへロックインしない。
- tenantごとにversioned wrapped DEKを保持し、record encryptionはTinkのAEAD primitiveが払い出すnonce/ciphertext/tagとkey versionを組にし、AAD `(tenant, context, table, record id, field)` を固定する。ciphertextのtenant/table間copyで復号できないようにする。
- 初期対象をScope記載のclient/provider secret、SMTP/connector credential、sensitive user attributes等にinventoryし、fieldごとにowner context repositoryでencrypt/decryptする。domain model全体をreflectionで暗号化しない。
- rotationは新DEKをactiveにしてnew writeを切替え、旧version decryptを保ちながらbackground re-encryption jobを再開可能に進める。全参照が移行するまで旧DEKをdestroyしない。DEKのin-memoryキャッシュはrotate/disable/destroy時に無効化し、複数worker replicaでも古いDEKで暗号化し続けないようにする。
- migrationはdual-read（encrypted優先、legacy plaintext fallback）→backfill→plaintext write停止→検証→plaintext列除去の段階導入にし、ログ/error/event/backupからplaintextを排除する。
- 移行/再暗号化ジョブは[[wi-126-async-job-runner]]が提供する`backend/jobs`のJobKind登録(HandlerRegistry)を使って実装し、独自のジョブ基盤を作らない。

## Tasks
- [x] T001 [Inventory/ADR] 暗号化対象field/owner、Tink+OpenBao(初期provider)/AEAD、AAD、DEK cache/fail mode、rotation/destroy、backup recovery、新規bounded context名、provider非ロックインの明記を決定する。→ [[ADR-148]] (`decisions/ADR-148-envelope-encryption-and-datakeys-context.md`)。設計本文は `ARCHITECTURE.md` §Persistence #3 / Context Map / Structural Decisions に記載し、ADRは「なぜ」に限定。
- [x] T002 [SCL] 新規bounded context (`DataKeys`) にencryption objectives、TenantDataKey lifecycle、rotate/health interfaces、key-loss/fail-closed constraints/contractsを追加して再生成し、ARCHITECTURE.md/architecture.yamlを同期する。→ `spec/contexts/data-keys.yaml` 新設、`spec/scl.yaml` context_map に `DataKeys` 追加、`architecture.yaml` に `contexts.DataKeys` 追加、`ARCHITECTURE.md` に Context Map行/Persistence設計/Structural Decisions同期。`just check` green。
- [x] T003 [Crypto] TinkベースのEnvelopeCrypto port、OpenBao provider adapter (`keys_vault`のHTTPクライアントパターンを流用)、Tink cleartext keysetによるlocal/dev fallback、DEKキャッシュのrotate/disable時無効化を実装しknown-answer/tamper/AAD testsを追加する。
  → `backend/shared/security/envelope_crypto` (`EnvelopeCrypto`/`MasterKeyProvider` port、`TinkEnvelopeCrypto`、AAD束縛)、`envelope_cleartext` (dev/local provider)、`envelope_openbao` (Vault Transit互換 provider、`keys_vault/vault_transit_http.go` のHTTPクライアントパターンを流用) を実装。
  - [x] RED: `TestGenerateWrapUnwrapRoundTrip`/`TestUnwrapRejectsWrongTenant`/`TestEncryptDecryptKnownAnswer`/`TestDecryptFailsClosedOnTamperedCiphertext`/`TestDecryptFailsClosedOnAADMismatch`/`TestHealthyDelegatesToMasterKeyProvider` (`envelope_crypto`)、`TestWrapUnwrapRoundTrip`/`TestUnwrapRejectsWrongTenant`/`TestHealthyIsAlwaysTrue` (`envelope_cleartext`)、`TestEnsureKeyCreatesMissingKey`/`TestEncryptDecryptRoundTrip`/`TestWrapUnwrapRoundTrip`/`TestUnwrapRejectsMismatchedMasterKeyID`/`TestWrapPropagatesProviderUnavailable` (`envelope_openbao`) を先に `undefined: ...` のcompile失敗で確認 → 実装 → GREEN (scenario `テナント初回利用時にDEKがbootstrapされる` / fail-closed 系 scenario、`spec/contexts/data-keys.yaml`)。
  - DEKキャッシュのrotate/disable時無効化は、rotate/disable usecase自体を持つT004 (Key Persistence) 側で実装する (キャッシュを無効化するトリガーがT004まで存在しないため)。
  - `just verify-go` green (pre-existing, unrelated failure: `TestAgentStatusMatchesSCL` in `backend/shared/spec` — casing mismatch, confirmed present before this work item's changes via `git stash`, out of scope for wi-97)。
- [x] T004 [Key Persistence] 新規bounded contextにwrapped tenant DEK/version/status repository、bootstrap/rotate/disable use caseをmemory/PostgreSQLへ実装する。
  → `backend/datakeys/domain` (TenantDataEncryptionKey、DataKeyStatus、lifecycle errors、events)、`backend/datakeys/ports` (DataKeyRepository、CacheInvalidator)、`backend/datakeys/db_memory`・`backend/datakeys/db_postgres` (両方 `ports.DataKeyRepository` 実装、Postgres 側は `infra/schema/postgres.sql` に `tenant_data_encryption_keys` テーブル・`sqlc.yaml` 追加のうえ sqlc 生成)、`backend/datakeys/usecases` (Bootstrap/Rotate/Disable/DestroyTenantDataKey usecase、`DataKeyCache` で rotate/disable/destroy 時に無効化、`spec.DomainEvent` emit)、`backend/datakeys/module.go` (composition root)。
  - [x] RED: db_memory の `TestBootstrapCreatesActiveVersionOne` ほか lifecycle 全パターン (bootstrap/rotate/disable/destroy 各正常系・拒否系、tenant 分離) を先に `undefined: NewDataKeyRepository` の compile 失敗で確認 → 実装 → GREEN。
  - [x] RED: usecases の `TestBootstrapTenantDataKey`/`TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion`/`TestDisableTenantDataKeyRejectsActiveVersion`/`TestDestroyTenantDataKeyErasesWrappedDEK` ほかを先に `undefined: Deps`/`BootstrapTenantDataKey` 等の compile 失敗で確認 → 実装 → GREEN (scenario `テナント初回利用時にDEKがbootstrapされる`/`DEKをrotationしても既存暗号文が復号できる`/`activeなDEKは直接disableできない`/`retiringのDEKを即時ロックアウトできる`/`全参照の再暗号化後にDEKをdestroyできる`、`spec/contexts/data-keys.yaml`)。
  - [x] RED: `DataKeyCache` の `TestGetActiveReturnsUnwrappedDEK`/`TestInvalidateForcesReUnwrapAfterRotate` を先に `undefined: DataKeyCache` の compile 失敗で確認 → 実装 → GREEN。
  - db_postgres は同じ port contract に対する実装であり (db_memory で test-first 済みの契約)、embedded PostgreSQL 上で `TestBootstrapRotateDisableDestroyLifecycle` により full lifecycle を検証、初回実行で GREEN。
  - 副作用の発見と対応: 新規テーブル追加により sqlc 生成の `models.go` がスキーマ全体の構造体を複製する仕様のため、既存 20+ context の生成済み `models.go` が一斉に 814 行になり `go-source-lines` 複雑度 ratchet (上限800) に抵触。sqlc 生成物 (`models.go`/`db.go`/`querier.go`/`*.sql.go`) を対象外にする exclude glob を `architecture.yaml` の `go-source-lines` budget に追加して解消 (既存の `**/generated/**`/`**/sqlcgen/**` は意図はしていたが実際の出力先と不一致だったため)。
  - SCL の `ListTenantDataKeyHealth` から `bindings: http` を一旦外す (ハンドラ未実装のため `TestAssembledRoutesMatchGeneratedOpenAPI` が spec-only mismatch で失敗していた)。HTTP binding は実装するT007で追加する。
  - `just check` / `just verify-go` green (pre-existing, unrelated failure: `TestAgentStatusMatchesSCL`)。`just build-go` green。
- [x] T005 [Repositories] 対象contextを一つずつdual-read/writeへ移行し、plaintextをevent/log/error DTOへ渡さないcontract testを追加する。
  → 第一対象として `backend/authentication/totp` の MFA TOTP シードを移行。`backend/datakeys/field_cipher.go` (`FieldCipher`) を新設し、record-owning repositoryが使うencrypt/decrypt facade (AAD束縛 + テナント初回アクセス時のlazy DEK bootstrap、SigningKeysの遅延鍵生成と同じ規約) を提供。`backend/authentication/totp/ports.SecretCipher` を新設 (PasswordHasherパターンに倣い、consuming context側にport、実装はdatakeys側)。
  - [x] RED: `FieldCipher` の `TestFieldCipherEncryptDecryptRoundTrip`/`TestFieldCipherEncryptBootstrapsFirstDataKey`/`TestFieldCipherDecryptFailsClosedAcrossRecords` を先に `undefined: FieldCipher` の compile 失敗で確認 → 実装 → GREEN。
  - [x] RED: `infra/schema/postgres.sql` の `mfa_factors` に `secret_key_version`/`secret_ciphertext` を追加 (legacy `secret` 列はdual-read用に残置)。`backend/authentication/totp/db_postgres/mfa.go` を先に `Cipher` field 未定義のcompile失敗で確認 (`mfa_test.go` を先に更新) → 実装 → GREEN。
  - contract tests (`backend/authentication/totp/db_postgres/mfa_test.go`, embedded PostgreSQL): `TestMfaFactorRepositoryRoundTrip` で新規保存行が `secret_ciphertext` 有り/レガシー `secret` 列NULLであることと ciphertext≠plaintext を検証、`TestMfaFactorRepositoryDualReadsLegacyPlaintext` で移行前平文行のdual-readを検証、`TestMfaFactorRepositoryDecryptFailsClosedForWrongTenant` で他テナントcontextからの復号がfail-closedであることを検証。
  - `spec/contexts/data-keys.yaml` の `EncryptedSecret.key_id` を `key_version: Integer` に変更 (Tinkの高レベルAEAD primitiveはnonceを内部管理しopaque idの別引きが不要なため、復号は呼び出し側が持つtenant_idとversionだけで完結する設計に合わせた)。`spec/contexts/authentication.yaml` の `MfaFactor.secret` にADR-148参照の記述を追加。
  - bootstrap配線 (`backend/cmd/internal/bootstrap/`): `datakeys.go` (`selectMasterKeyProvider`、`DATA_KEY_PROVIDER=openbao`でOpenBao、既定はTink cleartext keyset)、`deps.go`/`postgres.go`/`memory.go` に `DataKeys` module追加、`MfaFactorRepository` へ実際の `FieldCipher` を注入 (これが無いとPostgres経路で最初のMFA登録がnilポインタで落ちるため必須)。
  - 副作用の発見と対応: `backend/shared/architecture.yaml` の `shared-security-tokens-jose` に既存の未宣言 depends_on (oauth2-ports/shared-spec/signingkeys-domain/signingkeys-ports/tenancy-public) を発見・追加 (wi-97以前からの既存不備、`git show` で確認)。新設した `shared-security-envelope-crypto` のlayerを `adapters` から `use_cases` に修正 (port本体を持つ shared-services と同じ扱い。use_cases層のdatakeys-usecases/datakeys-publicから直接依存できるようにするため)。`backend/datakeys/architecture.yaml` を新設し、`backend/authentication/architecture.yaml`・`backend/cmd/architecture.yaml` の depends_on を同期。
  - `just check` / `just verify-go` (pre-existing, unrelated `TestAgentStatusMatchesSCL` を除く) / `just build-go` green。
- [x] T006 [Migration Job] [[wi-126-async-job-runner]]のJobKind/HandlerRegistryに登録する形でresumable backfill/re-encryption、per-field progress/checkpoint、verification queryと旧key destroy gateを実装する。
  → SCL: `spec/contexts/jobs.yaml` の `JobKind` に `data_key_reencryption` を追加。`spec/contexts/data-keys.yaml` に `DataKeyStillReferencedError` を追加し、`DestroyTenantDataKey` の `requires`/`errors` に反映、対応する destroy 拒否 scenario extension を追加。
  → Go: `backend/datakeys/ports/field_migrator.go` (`FieldMigrator` port: `ReencryptBatch`/`PendingCount`)。`backend/datakeys/usecases/migrator_registry.go` (`MigratorRegistry`、`jobs/usecases.HandlerRegistry` と同型)。`backend/datakeys/usecases/reencrypt.go` (`ReencryptTenantField`: 1 run あたり `ReencryptMaxBatchesPerRun`(25)×`ReencryptBatchSize`(200) まで進めて打ち切る、`ReencryptionHandler`: `data_key_reencryption` Job の Handler、残件があれば dedup 付きで続行 Job を再 enqueue、`EnqueueReencryptionJob`/`ReencryptionDedupKey`)。`backend/datakeys/usecases/lifecycle.go`: `Deps` に `Migrators`/`Jobs` を追加 (nil で無効化、既存呼び出し元は無変更)、`RotateTenantDataKey` が登録済み migrator ごとに再暗号化 Job を自動 enqueue、`DestroyTenantDataKey` が登録済み migrator の `PendingCount>0` を `ErrDataKeyStillReferenced` で拒否 (fail-closed destroy gate)。
  → 第一の FieldMigrator 実装: `backend/authentication/totp/db_postgres/reencrypt.go` (`MfaFactorReencryptor`: legacy plaintext と旧 DEK version の両方を対象に一括で再暗号化、`mfa_factors`/`users` を JOIN してテナントスコープする)。SQL は `reencrypt.sql` (`ListMfaFactorsPendingReencryption`/`CountMfaFactorsPendingReencryption`/`UpdateMfaFactorCiphertext`)、`just sqlc-generate` で生成。
  → 配線: `backend/datakeys/module.go` に `Migrators *usecases.MigratorRegistry` を追加。`backend/cmd/internal/bootstrap/postgres.go` で `MfaFactorReencryptor` を `mfa_totp_secret` として登録、`memory.go` は空レジストリ (memory runtime は暗号化しないため対象なし)。`backend/cmd/idmagic-worker/worker.go` に `KindDataKeyReencryption` の Handler を登録。`backend/cmd/idmagic-batch/main.go` に `data-key-reencryption-sweep` サブコマンドを追加 (全テナント×全 migrator を dedup 付きで enqueue、初回 backfill と定期再走査の運用エントリポイント。Infrastructure 層のため単体 red-green 免除、`signing-key-lifecycle` 同型)。
  - [x] RED: `backend/jobs/domain/job_test.go` の `TestLaneFor_BuiltinKinds` に `{KindDataKeyReencryption, LaneBulk}` を追加し `undefined: KindDataKeyReencryption` で確認 → 登録実装 → GREEN。
  - [x] RED: `TestMigratorRegistry_RegisterThenLookup`/`TestMigratorRegistry_LookupUnregisteredNameNotOK`/`TestMigratorRegistry_NamesListsEveryRegistered` を先に `undefined: NewMigratorRegistry` の compile 失敗で確認 → 実装 → GREEN。
  - [x] RED: `TestReencryptTenantField_MigratesAllInOneRunWhenUnderCap`/`TestReencryptTenantField_StopsAtMaxBatchesPerRunAndReportsRemaining`/`TestReencryptTenantField_UnregisteredMigratorReturnsError`/`TestReencryptionHandler_ReturnsResultJSONWithoutEnqueueingWhenDone`/`TestReencryptionHandler_ReenqueuesContinuationWhenRemaining`/`TestEnqueueReencryptionJob_DedupsRepeatedCalls` を先に `undefined: ReencryptTenantField`/`ReencryptDeps` 等の compile 失敗で確認 → 実装 → GREEN。
  - [x] RED: `TestRotateTenantDataKeyEnqueuesReencryptionJobForRegisteredMigrators`/`TestDestroyTenantDataKeyRejectsWhenMigratorReportsPendingRecords`/`TestDestroyTenantDataKeyAllowsWhenMigratorReportsNoPendingRecords` を先に `deps.Migrators undefined`/`deps.Jobs undefined` の compile 失敗で確認 → 実装 → GREEN (scenario `全参照の再暗号化後にDEKをdestroyできる`拡張、`spec/contexts/data-keys.yaml`)。
  - [x] RED (embedded PostgreSQL): `TestMfaFactorReencryptor_MigratesLegacyPlaintext`/`TestMfaFactorReencryptor_MigratesStaleVersionAfterRotationAndPreservesValue`/`TestMfaFactorReencryptor_SkipsRowsWithNoSecretMaterial`/`TestMfaFactorReencryptor_TenantIsolation` を先に `undefined: MfaFactorReencryptor` の compile 失敗で確認 → 実装 → GREEN。stale version 側は rotation 後も元の平文値へ round-trip することを検証。
  - 副作用の発見と対応: `backend/authentication/architecture.yaml` の `authentication-totp-db-postgres` に既存の未宣言 depends_on (`tenancy-domain`、`MfaFactorReencryptor` が `tenancy.WithTenant`/`tenancydomain.Tenant` を直接使うため) を追加。`backend/cmd/architecture.yaml` の `batch`/`worker` に `datakeys-usecases` (`via: composition_root`) を追加。`backend/datakeys/architecture.yaml` の `datakeys-usecases` に `jobs-domain`/`jobs-ports`/`jobs-usecases`/`shared-services` を追加 (`idmanagement-group-usecases` が `jobs-*` に直接依存する既存パターンを踏襲)。
  - `just check` / `just verify-go` (pre-existing, unrelated `TestAgentStatusMatchesSCL` を除く。`git stash` で本 WI 変更前から存在することを再確認) / `just build-go` green。
- [x] T007 [Operations] provider config validation、system key health/rotation status、runbook/backup restore手順を追加する。
  → SCL: `spec/contexts/data-keys.yaml` の `ListTenantDataKeyHealth` に `bindings: http: GET /api/admin/data-keys/health` を復元。
  → Go: `envelope_crypto.MasterKeyProvider`/`EnvelopeCrypto` に `Provider() string` を追加 (`envelope_cleartext`→`tink_cleartext`、`envelope_openbao`→`openbao`)、既存の `Healthy(ctx)` と対で health 表示に使う。`backend/datakeys/usecases/tenant_data_key_health.go` (`ListTenantDataKeyHealth`: `TenantRepo.FindAll` した全テナントを `DataKeyRepository.FindActive` し、bootstrap 前のテナントは省く。鍵材料は一切含めない)。`backend/datakeys/handlers_http` (新設。`GET /api/admin/data-keys/health`、`system_admin` のみ、`backend/signingkeys/handlers_http` の `ListTenantKeyHealth`/`requireSystemKeyHealthReader` と同型)。`backend/datakeys/module.go` に `Crypto` フィールドを追加し health handler・composition root 双方から共有。`backend/cmd/internal/bootstrap/datakeys.go`: `DATA_KEY_PROVIDER=openbao` で `OPENBAO_ADDR`/`OPENBAO_TOKEN` が空なら起動時に fail する provider config validation を追加 (最初の暗号化操作まで誤設定に気づけない状態を防ぐ)。
  → 配線: `backend/shared/http/server_http/routes.go`/`backend/cmd/idmagic/server.go` に `DataKeys` module と `datakeyshttp.RegisterRoutes` を追加。
  → Documentation: `README.md` に `DATA_KEY_PROVIDER`/`OPENBAO_*` env var、dev fallback (Tink cleartext keyset)、鍵紛失時の注意 (OpenBao Transit の backup が必須、Postgres 側の `tenant_data_encryption_keys` バックアップだけでは復旧できない)、`idmagic-batch data-key-reencryption-sweep` による backfill 運用を追記。`ARCHITECTURE.md` §Persistence #3 に `FieldMigrator` port と health endpoint の記述を追加。
  - [x] RED: `envelope_crypto_test.go` の `TestProviderDelegatesToMasterKeyProvider` を先に `crypto.Provider undefined` の compile 失敗で確認 → 実装 → GREEN (`envelope_cleartext`/`envelope_openbao` 側も `Provider()` 追加で再度 GREEN)。
  - [x] RED: `TestListTenantDataKeyHealth_ReportsBootstrappedTenantsAndOmitsUnbootstrapped`/`TestListTenantDataKeyHealth_ReflectsUnreachableProvider` を先に `undefined: ListTenantDataKeyHealth` の compile 失敗で確認 → 実装 → GREEN (scenario `systemAdminがテナント横断でDEK健全性を一覧する`、`spec/contexts/data-keys.yaml`)。
  - [x] RED (HTTP 統合、`backend/oauth2/handlers_http` の既存 admin テスト基盤を再利用): `TestAdminDataKeysHealthListsBootstrappedTenants`/`TestAdminDataKeysHealthRejectsPlainAdmin` を実装後に追加し authz gate (`system_admin` のみ) と応答形状を検証 (GREEN)。
  - 副作用の発見と対応: `datakeys-handlers-http` (path `handlers_http`) を新設し `datakeys-public` (path `.`) からルーティング責務を分離。`backend/shared/architecture.yaml` の `http-server` に `datakeys-handlers-http`/`datakeys-public` (`via: composition_root`) を追加。
  - `just check` / `just verify-go` (pre-existing、無関係な `TestAgentStatusMatchesSCL` を除く) / `just build-go` green。
  - 開示: rotate/disable/destroy の admin action endpoint は追加していない (Scope: 管理面は最小限の可視化に留める)。現状これらのライフサイクル操作を呼び出す本番経路は無く、内部 usecase として存在するのみ (将来 WI での配線を想定)。
  → **ui** (Scope): `frontend/src/features/admin-data-keys/SystemDataKeyHealthPage.tsx` + `.i18n.ts` (ja/en)、route `frontend/src/routes/system/data-keys.tsx`、`lib/systemNav.ts`/`components/shell.i18n.ts` にナビ項目追加。Scope 文言は「AdminKeys / AdminSettings」への表示追加としていたが、テナント横断のヘルス表示は既存の署名鍵ヘルス (`/system/keys`、System Console) と同じ性質の画面であり、実装済みの UI 構成 (テナント単位の Admin Console とは別のシステム管理者専用 System Console) に合わせて `/system/data-keys` を新設する形に倣った (`SystemKeyHealthPage`/`KeyHealthTable` と同型の `SystemDataKeyHealthPage`/`DataKeyHealthTable`)。テスト: `SystemDataKeyHealthPage.test.tsx` (空状態/ja・en切替/unreachable表示)、`systemNav.test.ts` 更新。`just typecheck-ui`/`just lint-ui`/`just test-ui-unit`/`just build-ui` (= `just verify-ui`) green。
  - 開示: フロントエンドの変更について、ブラウザでの実機確認 (dev server 上での目視) は本セッションでは実施していない。Testing Library によるレンダリング検証 (空状態・データあり状態・ja/en切替・provider unreachable 表示) と型検査・lint・build の green は確認済みだが、実際のブラウザでの見た目・操作性の確認は未実施であることを開示する。
- [x] T008 [Verify] ciphertext swap/tamper、wrong tenant/AAD/key version、provider outage/restart、rotation中read/write、log/DB/backup plaintext scanを検証する。
  → 既存カバレッジの棚卸し: ciphertext tamper/AAD mismatch は `TestDecryptFailsClosedOnTamperedCiphertext`/`TestDecryptFailsClosedOnAADMismatch` (T003)、wrong tenant は `TestUnwrapRejectsWrongTenant`/`TestMfaFactorRepositoryDecryptFailsClosedForWrongTenant` (T003/T005)、cross-record substitution は `TestFieldCipherDecryptFailsClosedAcrossRecords` (T005)、rotation中の既存暗号文復号・DataKeyCache無効化は `TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion`/`TestGetActiveReturnsUnwrappedDEK`/`TestInvalidateForcesReUnwrapAfterRotate` (T003/T004)、rotation後の再暗号化は `TestMfaFactorReencryptor_MigratesStaleVersionAfterRotationAndPreservesValue` (T006) で既にカバー済みであることを確認し、重複するテストは追加しなかった。
  → 新規に追加した gap: (1) wrong key version (ciphertext自体は正当だが異なる有効versionとして復号を試みる) は未カバーだったため追加。(2) provider outage (Wrap成功後にUnwrapのみ不可用になる、真の運用障害に近いシナリオ) はプリミティブ層 (`Healthy`/`Wrap`) でしか検証されていなかったため、`FieldCipher.Decrypt` 経由での fail-closed を追加。(3) DB全体の平文スキャン (個々のテストが触れた行だけでなく、複数テナント・複数行にわたる backfill 後のテーブル全体) は未カバーだったため追加。
  - [x] RED: `TestFieldCipherDecryptFailsClosedForWrongKeyVersion` (`backend/datakeys/field_cipher_test.go`) を実装前に追加・実行し、既存実装で即座に GREEN であることを確認 (新規production codeは不要、既存のfail-closed設計を検証する性質のtaskのため)。
  - [x] RED→GREEN: `TestFieldCipherDecryptFailsClosedWhenProviderUnavailable` (同ファイル。`unwrapUnavailableMasterKeyProvider` で Wrap 成功後の Unwrap 障害を模擬)。
  - [x] `TestMfaFactorReencryptor_NoPlaintextSurvivesBackfillAcrossTenants` (`backend/authentication/totp/db_postgres/reencrypt_test.go`、embedded PostgreSQL): 3 テナント×2ユーザーの legacy plaintext 行を backfill した後、`mfa_factors` テーブル全体を生 SQL で走査し `secret IS NOT NULL` な行が 0 件であることを確認。
  - log scan: `backend/datakeys`・`backend/authentication/totp`・`backend/shared/security/envelope_*` 全体を `logging.Warn/Error/Info/Debug` 呼び出しについて grep で棚卸しし、該当2箇所 (`datakeys/usecases/{reencrypt,lifecycle}.go` の再暗号化job enqueue失敗ログ) がいずれも `error`/`tenant_id`/`migrator` のみを記録し、平文・ciphertext・wrapped DEKを一切含まないことをコードレビューで確認した (自動テスト化はしていない)。domain event (`DataEncryptionKeyBootstrapped`/`Rotated`/`Disabled`/`Destroyed`) も構造体定義上 tenant_id/version 系フィールドのみで鍵材料を持たないことを `backend/datakeys/domain/events.go` の目視確認で確認した。
  - `just check` / `just verify-go` (pre-existing、無関係な `TestAgentStatusMatchesSCL` を除く) / `just build-go` green。
  - 開示: `## Verification` に記載の手動確認 2 件 (TOTP登録→DB上のciphertext目視確認、OpenBao停止時のfail-closed実機確認) は、本セッションでは実機 (稼働中のOpenBaoコンテナ含む) では未実施。同等の保証は上記の自動テスト (embedded PostgreSQLでのDB確認、fakeプロバイダでのfail-closed確認) で担保しているが、実インフラでの最終確認は運用側での実施を推奨する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: TOTP を登録 → DB 上の secret が鍵 ID 付き ciphertext で保存され、平文で ないことを確認する。
- 手動: DEK を rotation しても既存秘密が復号でき、OpenBao を停止すると該当秘密の 利用が fail-closed になることを確認する。

## Risk Notes
暗号化の実装ミスは復号不能 (=データ喪失) ・秘密漏洩・鍵紛失時のリカバリ不能に
直結する。migration での平文→暗号化移行、DEK rotation 後の復号性、master key
provider (OpenBao) 障害時の fail-closed を必ずテストする。AEAD/keyset の実装は
Tink に委ね、自前の暗号プリミティブ (nonce生成・AAD組み立てを含む) は実装しない。
`EnvelopeCrypto` port は provider を差し替え可能な抽象にとどめ、特定 KMS 実装
(OpenBao) にロックインしない。Tink の cleartext keyset による local/dev
fallback を用意し、開発時に OpenBao を必須にしない。

## Completion
- **Completed At**: 2026-07-29
- **Summary**:
  per-tenant DEK / KMS master key によるエンベロープ暗号化を導入した。ADR-148
  で設計を確定し、新規 bounded context `DataKeys` (`backend/datakeys`) が
  wrapped DEK のメタデータとライフサイクル (bootstrap/rotate/disable/destroy)
  を所有する。AEAD/keyset は Tink に委ね、master key custody は
  `EnvelopeCrypto`/`MasterKeyProvider` port の背後で OpenBao (Vault Transit
  互換) または Tink cleartext keyset (dev/local) に差し替え可能にした
  (`backend/shared/security/{envelope_crypto,envelope_openbao,envelope_cleartext}`)。
  第一の移行対象として MFA TOTP シード (`mfa_factors.secret`) を dual-read
  移行 (legacy plaintext → ciphertext) で暗号化対応した。
  移行/再暗号化は [[wi-126-async-job-runner]] の JobKind/HandlerRegistry に
  `data_key_reencryption` を追加して実装し、`FieldMigrator` port
  (`backend/datakeys/ports`) で DataKeys が消費側のスキーマを知らずに再暗号化
  ジョブを駆動できるようにした。Rotate は登録済み migrator ごとに再暗号化
  job を自動 enqueue し、Destroy は全 migrator の `PendingCount` が 0 になる
  まで crypto-shredding を拒否する fail-closed gate を持つ。運用面は
  `GET /api/admin/data-keys/health` (system_admin 限定の read-only 鍵健全性
  一覧、フロントエンド `/system/data-keys` 画面を含む)、`DATA_KEY_PROVIDER`
  起動時 config validation、`idmagic-batch data-key-reencryption-sweep`
  (初回 backfill / 定期再走査) を追加した。
- **Verification Results**:
  - `just check` (SCL / work-items / ids / architecture / traceability) - passed
  - `just verify-go` (lint 0 issues, `go test -race` 全パッケージ。
    `TestAgentStatusMatchesSCL` の失敗は本 WI 変更前から存在する既存不備
    (casing mismatch) であり、`git stash` で無関係であることを確認済み) -
    passed (既知の無関係な1件を除き green)
  - `just verify-ui` (format/lint/typecheck/build, 469 tests) - passed
  - `just build-go` - passed
- **Disclosed gaps / out of scope** (ADR-121):
  - 署名鍵 (`private_jwk`) 自体の鍵管理・Vault→OpenBao 移行は対象外
    ([[wi-32-kms-hsm-and-per-tenant-signing-keys]] が担当、Out of Scope 参照)。
  - Postgres TDE、転送時暗号化、フルの BYOK/customer-managed key 管理 UI、
    テナント削除時の crypto-shredding は Out of Scope の通り未実装。
  - rotate/disable/destroy の admin action HTTP endpoint は追加していない
    (Scope が管理面を read-only 可視化に限定しているため)。現状これらの
    usecase を呼ぶ本番経路は無く、内部専用 (将来 WI での配線を想定)。
  - `## Verification` に記載の手動確認 2 件 (TOTP登録後の DB ciphertext 目視、
    OpenBao 停止時の実機 fail-closed 確認) と、フロントエンド新規画面の
    ブラウザでの目視確認は、本セッションでは実機で実施していない。同等の
    保証は自動テスト (embedded PostgreSQL、fake provider、Testing Library
    によるレンダリング検証) で担保しているが、実インフラ・実ブラウザでの
    最終確認は運用側 / レビュー側での実施を推奨する。
