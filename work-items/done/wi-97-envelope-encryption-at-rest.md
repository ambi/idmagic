---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# 保存時のエンベロープ暗号化 (per-tenant DEK / KMS master key) を導入する

per-tenant DEK / KMS master key によるエンベロープ暗号化を導入した。ADR-148
で設計を確定し、新規 bounded context `DataKeys` (`backend/datakeys`) が
wrapped DEK のメタデータとライフサイクル (bootstrap/rotate/disable/destroy)
を所有する。AEAD/keyset は Tink に委ね、master key custody は
`EnvelopeCrypto`/`MasterKeyProvider` port の背後で OpenBao (Vault Transit
互換) または Tink cleartext keyset (dev/local) に差し替え可能にした
(`backend/shared/security/{envelope_crypto,envelope_openbao,envelope_cleartext}`)。
第一の移行対象として MFA TOTP シード (`mfa_factors.secret`) を dual-read
移行 (legacy plaintext → ciphertext) で暗号化対応した。
移行/再暗号化は [[wi-42-async-job-runner]] の JobKind/HandlerRegistry に
`data_key_reencryption` を追加して実装し、`FieldMigrator` port
(`backend/datakeys/ports`) で DataKeys が消費側のスキーマを知らずに再暗号化
ジョブを駆動できるようにした。Rotate は登録済み migrator ごとに再暗号化
job を自動 enqueue し、Destroy は全 migrator の `PendingCount` が 0 になる
まで crypto-shredding を拒否する fail-closed gate を持つ。運用面は
`GET /api/admin/data-keys/health` (system_admin 限定の read-only 鍵健全性
一覧、フロントエンド `/system/data-keys` 画面を含む)、`DATA_KEY_PROVIDER`
起動時 config validation、`idmagic-batch data-key-reencryption-sweep`
(初回 backfill / 定期再走査) を追加した。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
