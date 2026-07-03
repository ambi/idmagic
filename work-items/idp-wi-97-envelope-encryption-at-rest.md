---
id: idp-wi-97-envelope-encryption-at-rest
title: "保存時のエンベロープ暗号化 (per-tenant DEK / KMS master key) を導入する"
created_at: 2026-07-03
authors: ["tn"]
status: pending
risk: high
---
# Motivation
可逆な秘密の一部が DB に平文で保存されている。ハッシュ化できる一方向秘密
(パスワード / reset token / client_secret) は既にハッシュ化されているが、
復号が必要な可逆秘密は暗号化 or 外部 KMS が要る:

- 署名秘密鍵 `signing_keys.private_jwk` (JSONB, 平文。ADR-024 に「現実装の
  簡略化」と明記)。
- TOTP シード `mfa_factors.secret` (平文)。
- 将来の Token Vault ([[wi-55-token-vault-federated-connections]]) が預かる
  外部 API アクセストークン。

代表的な IdP は customer-managed key / BYOK を含む保存時暗号化を提供する
(Okta / Entra)。本 WI は per-tenant DEK を KMS の master key で包む
**エンベロープ暗号化**を導入し、テナント境界での鍵分離・鍵ローテーション・
fail-closed な復号を実現する。署名鍵そのものの鍵管理は
[[wi-32-kms-hsm-and-per-tenant-signing-keys]] が担い、本 WI とは master key
基盤を共有する。

# Scope
- **decision**: 新規 ADR: エンベロープ暗号化の設計。master key は KMS 管理、per-tenant DEK を master key で暗号化保管し、DEK で AEAD (AES-GCM) 暗号化する。暗号化対象秘密の 選定 (TOTP シード / Token Vault トークン / その他可逆秘密)、鍵 ID の付与、 DEK / master の rotation、復号失敗時の fail-closed、local/dev fallback、 wi-32 (署名鍵の KMS 化) との責務分担 (private_jwk は wi-32 が所有し、本 WI は DB に残る可逆秘密を対象)、BYOK / customer-managed key を将来拡張とする境界を 記録する。
- **scl**: §3.2 models: TenantDataEncryptionKey / EncryptedSecret (envelope: 鍵 ID + ciphertext + nonce) を追加する。暗号化は主に adapter 層の実装で、SCL への 露出は最小に留める。, [object Object], §3.5 invariants: 可逆秘密を平文で保存しない、DEK はテナント単位、復号不能時は アクセスを拒否する (fail-closed) ことを明示する。
- **go**: crypto adapter (KMS master + per-tenant DEK キャッシュ + AEAD helper) を 追加し、KMS adapter は wi-32 と共有する (初期は AWS or GCP KMS 1 つ + local dev fallback)。, 対象 repository (mfa factor store / token vault 等) を暗号化対応にし、 migration で既存平文を暗号化へ再暗号化する。
- **http**: 管理面は最小 (鍵状態 / 暗号化状態の可視化程度) に留める。
- **ui**: AdminKeys / AdminSettings に暗号化・DEK 状態の表示を最小限追加する。
- **documentation**: README に暗号化方針・KMS 設定・dev fallback・鍵紛失時の注意を追記する。

# Out of Scope
- 署名鍵 (private_jwk) 自体の鍵管理 ([[wi-32-kms-hsm-and-per-tenant-signing-keys]])。
- Postgres TDE / ディスク暗号化 (インフラ層の責務)。
- 転送時暗号化 (既存 TLS)。
- フルの BYOK / customer-managed key の管理 UI (将来拡張)。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: TOTP を登録 → DB 上の secret が鍵 ID 付き ciphertext で保存され、平文で ないことを確認する。
- 手動: DEK を rotation しても既存秘密が復号でき、KMS を停止すると該当秘密の 利用が fail-closed になることを確認する。

# Risk Notes
暗号化の実装ミスは復号不能 (=データ喪失) ・秘密漏洩・鍵紛失時のリカバリ不能に
直結する。migration での平文→暗号化移行、DEK rotation 後の復号性、KMS 障害時の
fail-closed を必ずテストする。標準 AEAD (AES-GCM) と実績ある KMS SDK を用い、
自前の暗号プリミティブは実装しない。local/dev fallback を用意して開発時の
KMS 依存を避ける。
