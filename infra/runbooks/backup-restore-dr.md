# Backup / Restore / Disaster Recovery Runbook

## 概要

IdMagic は認証の単一障害点であり、この runbook はバックアップ対象の分類、
バックアップ／リストア手順、リストア後の検証、障害シナリオ別の対応を定める
(wi-101)。バックアップ対象は次の 2 系統のみである
(旧 Valkey は 撤去済み、旧 `idmagic-relay`/Kafka は
撤去済み):

1. **PostgreSQL**: durable テーブルと ephemeral テーブル (認可中間状態・
   コード・PAR・device code・リプレイ・WebAuthn チャレンジ・denylist・
   login throttle・SAML AuthnRequest リプレイ) を両方含む単一のデータベース。
2. **署名鍵素材**: `KeyProvider` が `Local`/`Postgres` なら
   PostgreSQL のバックアップに含まれる。`VaultTransit` なら private key は
   Vault 外に出ないため、Vault 側のスナップショットが正本であり
   PostgreSQL 側は public key のミラーに過ぎない。

## バックアップ手順

### PostgreSQL: 論理バックアップ (pg_dump, portable export / 小規模 drill 用途)

```sh
just backup-postgres <output-dir> [database-url]
```

`infra/backup/backup-postgres.sh` が `pg_dump -Fc` (custom format) で
timestamp 付きの dump を取得し、sha256 チェックサムを併記出力する。
接続先はデフォルトを持たず、呼び出し側が明示する。

### PostgreSQL: 物理バックアップ (PITR, 本番の正本)

本番規模の RPO を満たすには base backup + 継続的な WAL archive
(`pg_basebackup` + `archive_command` によるアーカイブ、または管理サービスの
継続アーカイブ機能) を正本とする。ローカル docker compose の `postgres`
サービスは WAL archiving を有効化していない単一ノード構成のため、この
runbook のローカルドリルは pg_dump による論理バックアップ経路を検証する。
本番導入時は WAL archive の宛先 (オブジェクトストレージ等) と retention を
別途決定し、この節を具体化する。

### 署名鍵素材

- `Local`/`Postgres` provider: private key は `signing_keys.private_jwk` に
  平文で入っており、PostgreSQL のバックアップにそのまま含まれる。**平文鍵を
  含むバックアップの保管先自体を暗号化する** (保存先ボリューム/バケットの
  暗号化、または dump をさらに `age`/`gpg` で包んでから保管する)。
- `VaultTransit` provider: private key は Vault 内に閉じる。Vault/OpenBao
  自身のスナップショット機構 (`vault operator raft snapshot save` 相当) で
  別途バックアップする。PostgreSQL 側の `signing_keys.private_jwk` はこの
  provider では空/プレースホルダであり、鍵の復旧に使えない。

## リストア手順

```sh
just restore-postgres <backup-file> [database-url]
```

`infra/backup/restore-postgres.sh` は次の順で実行する
(`infra/schema/check-convergence.sh` と同じ「使い捨て compose project +
trap cleanup」パターンを踏襲):

1. **non-production guard**: 明示フラグ (`--yes-restore-into-this-database`)
   と compose project 名 (`idmagic-restore-drill*` 等) の両方を要求し、
   本番接続文字列らしき対象への誤実行を防ぐ。
2. `infra/schema/postgres.sql` を空データベースへ先に apply する
   (schema と data restore を競合させない)。
3. `pg_restore --data-only` で data のみを復元し (schema は前段の psqldef 適用で既に存在するため二重に作らない)、dump のチェックサムを事前検証する。
4. ephemeral テーブル (UNLOGGED/LOGGED いずれも) を明示的に truncate する。
   UNLOGGED テーブルは PITR/物理バックアップ自体に含まれず自動的に空で
   戻る性質があるが、pg_dump 経由の論理バックアップには含まれ得るため、
   この truncate を省略しない。
5. `idmagic-batch restore-consistency-check` を実行し、以降の「リストア後
   検証チェックリスト」を機械的に検査する。

### リストア整合順序 (DR 全体)

KMS/Vault access 確認 → PostgreSQL restore (schema 先行 apply → data
restore) → schema/version 検査 → ephemeral テーブル truncate → signing
key の DB metadata と provider key version の整合チェック → API/worker
起動 → JWKS/token 検証。**鍵喪失を「DB だけ restore」で隠さない**:
signing key の検証に失敗したら fail-closed とし、DB restore とは別に
emergency rotation 手順 (下記) に進む。

## リストア後の検証チェックリスト

`just restore-postgres` の最後に自動実行される
`idmagic-batch restore-consistency-check` が次を検査する:

- [ ] tenant / user / client のレコード件数が 0 でない。
- [ ] 各 tenant の active signing key が解決可能で、JWKS を構成できる。
- [ ] `jobs` テーブルに `dedup_key` 違反 (同一 tenant+dedup_key の queued/
      running 重複) がない。
- [ ] ephemeral テーブル (UNLOGGED/LOGGED) がリストア直後に空である
      (truncate 漏れの検出)。

手動でも以下を確認する:

- [ ] 既発行 token が復旧後の JWKS で検証できる (kid が消えていない)。
- [ ] 代表 tenant で疎通 (ログイン/token 発行) を一度通す。

## 障害シナリオ別 DR 手順

### DB loss (PostgreSQL インスタンス喪失)

1. 直近の PITR ベース (production) または pg_dump (drill/小規模) から
   `just restore-postgres` を実行。
2. リストア整合順序に従い、consistency-check が green になるまで API を
   再開しない。
3. RPO は「最後の WAL archive / dump からの経過時間」。RTO は「検知から
   consistency-check green までの経過時間」で計測する。

### Point-in-time 誤削除 (特定 tenant/レコードの誤削除・誤更新)

1. 影響範囲 (tenant/user/client) を audit log (`audit_events`) から特定する。
2. PITR で誤削除直前の time point へ復元した使い捨て環境を用意し、
   該当レコードのみを本番へ再投入する (全体ロールバックは避け、影響範囲を
   最小化する)。
3. pg_dump drill しか使えない環境では、直近の定期 dump までしか戻せない
   ことを事前に周知し、誤削除検知までのリードタイムを短縮する運用
   (アラート、確認ダイアログ) を優先する。

### Region loss (リージョン喪失)

マルチリージョンのアクティブ/アクティブ構成は wi-101 の Out of Scope。
本 runbook が扱うのは「リージョン内の最新バックアップから、代替リージョンで
新規に環境を構築し `restore-postgres` を実行する」単一リージョン再構築の
手順のみ。RTO は環境構築時間 (IaC 適用) + restore 時間の合算になる。

### Key provider loss (Vault/OpenBao 喪失、または signing key 破損)

1. **PostgreSQL restore で鍵喪失を隠さない**: consistency-check の
   signing key 検証が失敗したら、DB restore が成功していても fail-closed
   のまま新規 token 発行を止める。
2. Vault/OpenBao 側のスナップショットから鍵ストアを復旧できる場合は復旧し、
   `signing_keys` の public key ミラーと Vault 側の kid が一致することを
   確認する。
3. 復旧不能な場合は emergency rotation (新しい signing key を発行し、旧
   kid は JWKS から段階的に外す) に進む。この間、旧 kid で署名された
   token の検証は継続できないことを利用者へ告知する。

### Partial restore (一部テーブル/一部 tenant のみの復元)

1. 対象範囲を限定した pg_dump (`pg_dump --table=...` 等) からの復元は
   外部キー制約・`jobs`/`signing_keys` などの整合を壊しやすいため、
   consistency-check を必ず通す。
2. 部分復元後に全体の整合が取れないと判断した場合は、全体リストアに
   切り替える (部分復元を無理に完遂しない) — rollback の判断基準は
   consistency-check の結果とする。

## 運用 (定期 drill・アクセスレビュー・expiry)

- **定期 drill**: `just restore-drill` を定期的に (最低でも四半期に 1 回)
  実行し、backup 成功だけでなく restore 可能性そのものを確認する。実測
  RPO/RTO を下記の表に追記する。
- **アクセスレビュー**: バックアップ保管先 (暗号化された保存先) と
  Vault/OpenBao の snapshot へのアクセス権限を、鍵ローテーションと同じ
  cadence で棚卸しする。
- **expiry**: バックアップの retention 期間を明示し、期限切れ dump を
  自動削除する運用を設ける。
- 上記のスケジューリング・アラート基盤への実配線 (cron/alertmanager 等)
  は本 runbook の記述時点では未実施。ローカル drill と手順の整備を先行し、
  実運用環境が用意でき次第、配線する。

## 実測 RPO/RTO (ローカル drill)

| 実行日 | 環境 | シナリオ | RPO (実測) | RTO (実測) | 備考 |
| --- | --- | --- | --- | --- | --- |
| 2026-08-01 | ローカル docker compose (`just restore-drill`) | tenant 1 / user 2 / client 3 / signing key 1 件の pg_dump backup → DB 破棄・再作成 → restore → consistency-check | 0s (backup 直後に破棄したため無視できる差分) | 6.3s (全体、うち restore+consistency-check は 2s) | データ量が小さいローカル drill の下限値。本番規模の RPO/RTO は PITR + staging 実測で別途確定する。 |

Vault Transit を含む staging 環境での実ドリルは、dev 環境に Vault/OpenBao
サービスが存在しないため未実施。実施可能な環境が用意され次第、上表に
追記する。
