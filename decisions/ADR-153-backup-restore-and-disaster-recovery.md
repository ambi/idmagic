---
status: accepted
authors: [tn]
created_at: 2026-08-01
---

# ADR-153: バックアップ対象を単一 PostgreSQL + 鍵素材に分類し、RPO/RTO とリストア整合順序を定義する

## コンテキスト

idmagic は単一障害点である IdP のダウンから「復旧できるか」「どこまで戻るか
(RPO)」「どれだけで戻せるか (RTO)」を未定義のまま残してきた（wi-101）。
[ADR-139](ADR-139-consolidate-ephemeral-state-into-postgresql.md) により
durable state と ephemeral state (旧 Valkey 分) は既に単一の PostgreSQL に
統合済みで、[ADR-147](ADR-147-remove-external-message-brokers.md) により
外部メッセージブローカー (Kafka/Pub/Sub) と `idmagic-relay` も撤去済みである。
つまりバックアップ対象は「PostgreSQL 1 系統」と「鍵素材」の 2 系統に単純化
されており、wi-101 が当初前提としていた「PostgreSQL + Valkey + 鍵」という
3 系統モデルはもう実態と合わない。

鍵素材は [ADR-075](ADR-075-per-tenant-signing-keys-and-key-provider.md) の
`KeyProvider` で分岐する: `Local`/`Postgres` (dev/test、private key を
PostgreSQL の `signing_keys.private_jwk` に平文保持) と `VaultTransit`
(本番、private key は Vault 外に出ず、PostgreSQL は public key のミラーのみ
保持)。この分岐により、PostgreSQL のバックアップだけでは本番の鍵素材を
復旧できない場合があることを明示する必要がある。

## 決定

1. **バックアップ対象は PostgreSQL 単一 + 鍵素材の 2 系統とする。** PostgreSQL
   は durable テーブルと ephemeral テーブル (`oauth2_authorization_requests`
   等の UNLOGGED、`oauth2_access_token_denylist`/`login_throttle_counters`
   等の LOGGED) を両方含む一つのバックアップ対象。鍵素材は provider が
   `Local`/`Postgres` なら PostgreSQL のバックアップに含まれ、`VaultTransit`
   なら private key の正本は Vault 側スナップショットであり、PostgreSQL の
   `signing_keys` は public key のミラーに過ぎない。
2. **PITR (base backup + WAL archive) を本番の正本、`pg_dump` を portable
   export・小規模 drill 用途に限定する。** 手順・スクリプトの詳細は
   [`infra/runbooks/backup-restore-dr.md`](../infra/runbooks/backup-restore-dr.md)
   に置く（本 ADR には転記しない）。
3. **ephemeral テーブルはリストア後に意図的に truncate し、復旧対象外とする。**
   UNLOGGED テーブルは PITR/物理バックアップに含まれず自動的に空で戻る性質を
   利用し、LOGGED な ephemeral (denylist/login throttle) も明示的に
   truncate することで、stale な認可中間状態・リプレイ状態・throttle
   カウンタが復旧後に復活しないことを保証する。これは旧 Valkey 撤去前の
   「Valkey は復旧対象外」という割り切りの後継である。
4. **リストア整合順序を固定する**: KMS/Vault access 確認 → PostgreSQL restore
   (schema 先行 apply → data restore) → schema/version 検査 → ephemeral
   UNLOGGED/LOGGED テーブルの truncate → signing key の DB metadata と
   provider key version の整合チェック → API/worker 起動 → JWKS/token 検証。
   鍵喪失を「DB だけ restore」で隠さない: 検証不能なら fail-closed とし、
   DB restore とは別に emergency rotation 手順を踏む。
5. **RPO/RTO はローカルドリルの実測値を暫定目標とし、本番相当の値は staging
   実測で確定するまで確約しない。** ローカル docker compose での
   backup→破棄→restore ドリル実測値は
   [`infra/runbooks/backup-restore-dr.md`](../infra/runbooks/backup-restore-dr.md)
   に記録する。Vault Transit を含む staging 実ドリルは、dev 環境に
   Vault/OpenBao サービスが存在しないため本 ADR / wi-101 の範囲では未実施
   とし、runbook に手順のみ記載する。

## 却下した代替案

- **旧 wi-101 のまま「永続 = PostgreSQL、揮発 = Valkey」の 2 層 + 鍵で計画する。**
  ADR-139 で Valkey 自体が撤去済みのため、実装対象が存在しない前提になる。
- **鍵素材のバックアップを PostgreSQL のバックアップ手順に一本化する。**
  `VaultTransit` provider では private key が Vault 外に出ない設計
  (ADR-075) を壊すため、鍵の正本管理は provider ごとに手順を分ける。
- **本番相当の RPO/RTO を目算で確約する。** 実測なしの数値は runbook の
  信頼性を損なう。ローカルドリルの実測値を暫定目標とし、staging 実測で
  確定する運用とする。

## 影響

- **SCL**: 影響なし（運用手順であり、`spec/` が記述する振る舞い・契約は
  変わらない）。
- **runbook**: `infra/runbooks/backup-restore-dr.md` を新設。
- **tooling**: `infra/backup/backup-postgres.sh` /
  `infra/backup/restore-postgres.sh` / `infra/backup/restore-drill.sh` を
  新設し、`justfile` に `backup-postgres` / `restore-postgres` /
  `restore-drill` レシピを追加する。
- **backend**: `backend/cmd/idmagic-batch` に `restore-consistency-check`
  subcommand を追加する。
- **work-item**: [[wi-101-backup-restore-and-disaster-recovery]] を本 ADR
  に沿って更新・完了する。

## 関連

- [ADR-139](ADR-139-consolidate-ephemeral-state-into-postgresql.md) —
  ephemeral state を PostgreSQL に統合し Valkey を撤去した決定。
- [ADR-147](ADR-147-remove-external-message-brokers.md) — 外部メッセージ
  ブローカーと `idmagic-relay` を撤去した決定。
- [ADR-075](ADR-075-per-tenant-signing-keys-and-key-provider.md) —
  `KeyProvider` (Local/Postgres/VaultTransit) の分岐。
- [ADR-148](ADR-148-envelope-encryption-and-datakeys-context.md) — master
  key custody の provider 抽象 (OpenBao/Vault Transit 互換)。
