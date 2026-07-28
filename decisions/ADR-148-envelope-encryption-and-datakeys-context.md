---
status: suggested
authors: [tn]
created_at: 2026-07-29
---

# ADR-148: DB に残る可逆秘密を per-tenant DEK / OpenBao master key によるエンベロープ暗号化で保護する

## コンテキスト

署名鍵以外にも DB に平文で残る可逆秘密がある (TOTP シード `mfa_factors.secret`、
将来の Token Vault upstream token ([[wi-55-token-vault-federated-connections]]))。
[[ADR-075]] の `KeyProvider` (Local/Database/VaultTransit) は `transit/sign` 用途に
特化しており、汎用の `encrypt`/`decrypt`/`datakey` には使えない。[[ADR-054]] は
Token Vault の upstream token 暗号化を「新規の鍵管理システムを増設せず既存 KMS を
流用する」と決めており、本 ADR がその「既存 KMS」の実体を確定する。

## 決定

新規 bounded context `DataKeys` (`backend/datakeys`) を追加し、テナント単位の
wrapped DEK のメタデータとライフサイクル (bootstrap/rotate/disable/destroy) を
所有する。DEK の AEAD 暗号化と master key custody は Tink + OpenBao (初期実装、
Vault Transit 互換 API) を provider 抽象 (`EnvelopeCrypto` port) の背後に置き、
`backend/shared/security` の技術共有 adapter とする (署名用 `KeyStore` port とは
統合しない)。二段のエンベロープ暗号化・AAD 構成・rotation 手順・fail-closed の
具体的なメカニズムは
[ARCHITECTURE.md §Persistence #3](../ARCHITECTURE.md#3-envelope-encryption-for-reversible-secrets)
を正本とする (本 ADR には転記しない)。

署名鍵 (`private_jwk`) は [[wi-32-kms-hsm-and-per-tenant-signing-keys]] が引き続き
所有し、`DataKeys` は触れない。[[ADR-054]] の「既存 KMS を流用する」要件は本 ADR の
`DataKeys` + `EnvelopeCrypto` + OpenBao を指し、
[[wi-55-token-vault-federated-connections]] はこれを再利用して新規鍵管理
コンポーネントを追加しない。BYOK / customer-managed key は `EnvelopeCrypto` port の
provider 抽象を拡張点として残すのみで、本 ADR では実装しない (将来拡張)。

## 却下した代替案

- **HashiCorp Vault CE**: 2023 年のライセンス変更 (BUSL 1.1) で OSI 承認 OSS でなくなり、
  self-host OSS 既定方針 ([[ADR-075]] と同じ理由) に反する。OpenBao を採る。
- **クラウド KMS の直接統合**: クラウド SDK/アカウント/IAM 前提を持ち込み、self-host OSS
  既定方針と整合しない ([[ADR-075]] と同じ理由)。
- **自前の AEAD/keyset 実装**: nonce/AAD 組み立てを含む自前実装は復号不能・秘密漏洩の
  リスクが高く、Tink に委ねる。
- **既存 `KeyStore` port への統合**: sign/verify と encrypt/decrypt/datakey は操作・
  ライフサイクルが異なり、統合すると責務が混在する。
- **テナント共通の単一 DEK**: テナント境界での鍵分離と将来の crypto-shredding
  拡張余地 ([[wi-97-envelope-encryption-at-rest]] Out of Scope) を失うため per-tenant
  DEK を採る。

## 影響

- SCL: 新規 context `DataKeys` (`models.TenantDataEncryptionKey` /
  `models.EncryptedSecret`、`states.DataEncryptionKeyRotated` ほか)。
- Design: [ARCHITECTURE.md](../ARCHITECTURE.md) の Context Map / Persistence /
  Structural Decisions を同期済み。
- Go (未実装、本 WI 後続タスク): `backend/shared/security` に `EnvelopeCrypto` port と
  Tink/OpenBao adapter、`backend/datakeys` に DEK repository/usecase、
  `mfa_factors` repository の暗号化対応、[[wi-126-async-job-runner]] への
  再暗号化 JobKind 登録。
