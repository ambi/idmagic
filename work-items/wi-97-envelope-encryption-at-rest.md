---
depends_on: [wi-126-async-job-runner]
status: pending
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
- [ ] T006 [Migration Job] [[wi-126-async-job-runner]]のJobKind/HandlerRegistryに登録する形でresumable backfill/re-encryption、per-field progress/checkpoint、verification queryと旧key destroy gateを実装する。
- [ ] T007 [Operations] provider config validation、system key health/rotation status、runbook/backup restore手順を追加する。
- [ ] T008 [Verify] ciphertext swap/tamper、wrong tenant/AAD/key version、provider outage/restart、rotation中read/write、log/DB/backup plaintext scanを検証する。

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
