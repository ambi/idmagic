# バックアップ、復元、災害復旧の運用手順書

## 概要

IdMagic は認証基盤の単一障害点になりうるため、この手順書ではバックアップ対象の分類、バックアップと復元の手順、復元後の検証、障害シナリオごとの対応を定める。バックアップ対象は次の 2 系統である。

1. **PostgreSQL**: 永続テーブルと一時テーブル（認可の中間状態、コード、PAR、デバイスコード、リプレイ記録、WebAuthn チャレンジ、拒否リスト、ログインスロットル、SAML AuthnRequest のリプレイ記録）を含む単一のデータベース。
2. **署名鍵素材**: `KeyProvider` が `Local` / `Database` なら PostgreSQL のバックアップに含まれる。`VaultTransit` では Vault 側のスナップショットが正本であり、PostgreSQL 側は公開鍵の写しにすぎない。どの提供元で鍵素材が平文になるかは [SigningKeys の集約](../modules/signing-keys/data.md#集約) が定める。
3. **データ暗号鍵のマスターキー**: `DATA_KEY_PROVIDER` が保持するマスターキー。`tenant_data_encryption_keys` の PostgreSQL バックアップにはマスターキーでラップした形しか含まれないため、それだけでは復旧できない。

### マスターキーの喪失は復旧できない

OpenBao の Transit 鍵をバックアップせずに失うと、すべてのテナントのラップ済み DEK を恒久的にアンラップできなくなる。これは `DestroyTenantDataKey` が意図的に実現する暗号学的消去と同じ結果が、事故として起きることを意味する。**Transit エンジンのストレージは OpenBao 自身のバックアップ手段で保護する。** PostgreSQL のバックアップ計画とは別に計画し、別に復元試験を実施する。

## バックアップ手順

### PostgreSQL の論理バックアップ

```sh
mise run backup-postgres <output-dir> [database-url]
```

`infra/backup/backup-postgres.sh` が `pg_dump -Fc`（カスタム形式）でタイムスタンプ付きのダンプを取得し、SHA-256 チェックサムを併記して出力する。接続先はデフォルト値を持たず、呼び出し側が明示する。

### PostgreSQL の物理バックアップ

本番規模の RPO を満たすには、ベースバックアップと継続的な WAL アーカイブ（`pg_basebackup` と `archive_command` によるアーカイブ、またはマネージドサービスの継続アーカイブ機能）を正本とする。ローカルの Docker Compose にある `postgres` サービスは WAL アーカイブを有効にしていない単一ノード構成なので、この手順書のローカルの復元試験では `pg_dump` による論理バックアップの経路を検証する。本番へ導入するときは WAL アーカイブの保存先（オブジェクトストレージなど）と保持期間を別途決定し、この節を具体化する。

### 署名鍵の素材

- `Local` / `Database` プロバイダー: バックアップは鍵素材そのものとして扱う。保存先を暗号化する要求は [SigningKeys モジュールの判断](../modules/signing-keys/decisions.md) が定めており、ここが示すのはその手段である。保存先のボリュームやバケットを暗号化するか、ダンプをさらに `age` / `gpg` で暗号化してから保存する。
- `VaultTransit` プロバイダー: 秘密鍵は Vault 内に閉じる。Vault / OpenBao 自身のスナップショット機構（`vault operator raft snapshot save` 相当）で別途バックアップする。PostgreSQL 側の `signing_keys.private_jwk` はこのプロバイダーでは空またはプレースホルダーであり、鍵の復旧には使えない。

## 復元手順

```sh
mise run restore-postgres <backup-file> [database-url]
```

`infra/backup/restore-postgres.sh` は次の順で実行する（`infra/schema/check-convergence.sh` と同じ、使い捨ての Compose プロジェクトを終了時に片付ける方式を使う）。

1. **本番環境への誤操作防止**: 明示的なフラグ（`--yes-restore-into-this-database`）と Compose プロジェクト名（`idmagic-restore-drill*` など）の両方を要求し、本番環境らしい接続文字列を持つ対象への誤実行を防ぐ。
2. `infra/schema/postgres.sql` を空のデータベースへ先に適用し、スキーマとデータの復元を競合させない。
3. ダンプのチェックサムを検証してから、`pg_restore --data-only` でデータだけを復元する。スキーマは前段の `psqldef` 適用で存在するため、二重に作成しない。
4. 一時テーブル（`UNLOGGED` / `LOGGED` の両方）を明示的に `TRUNCATE` する。`UNLOGGED` テーブルは PITR や物理バックアップには含まれず自動的に空で戻るが、`pg_dump` による論理バックアップには含まれうるため、この処理を省略しない。
5. `idmagic-batch restore-consistency-check` を実行し、以降の復元後検証チェックリストを機械的に検査する。

### 復元時の整合性を保つ順序

[リカバリ設計の復元の順序](../design/reliability/recovery.md#復元の順序)に従い、次の順に進める。
各手順の確認が済むまで、次の手順へ進まない。
復元前に API とレーン別ワーカーの復旧時に戻すレプリカ数、再開対象の CronJob を記録し、API とワーカーを停止する。
`idmagic-batch` の CronJob は `suspend: true` にし、実行中の Job も終了したことを確かめてから復元する。
ワーカーの各 Deployment はレプリカ数 0、CronJob は停止したまま手順 6 を迎える。

1. 利用する KMS / Vault へアクセスし、復元に必要な鍵を参照できることを確かめる。
2. PostgreSQL の空のデータベースへスキーマを先に適用し、データを復元する。
   スキーマ適用、チェックサム検証、データ復元がすべて成功したことを確かめる。
3. スキーマとバージョンを検査し、復旧するアプリケーションのバージョンと一致することを確かめる。
4. 一時テーブルを `TRUNCATE` し、対象の全テーブルが 0 件であることを確かめる。
5. 署名鍵のデータベースメタデータとプロバイダー側の鍵バージョンが一致し、[復元後の確認一覧](#復元後の確認一覧)の自動検査がすべて成功したことを確かめる。
6. `idmagic-api` だけを起動する。
   JWKS に発行済みトークンの `kid` があり、その JWKS で署名を検証できること、代表テナントでログインとトークン発行を 1 回ずつ成功できることを確かめる。
7. `latency_sensitive` の Deployment `idmagic-worker-latency-sensitive` のレプリカ数を 0 から記録した数へ戻す。
   準備完了レプリカ数がその数と一致し、手順 6 の認証確認が引き続き成功することを確かめる。
8. `default` の Deployment `idmagic-worker-default` を同じ方法で戻す。
   準備完了レプリカ数と認証の成功を確かめ、さらに PostgreSQL の使用中の接続数を利用可能な接続数で割った使用率が 70% 未満であることを確かめてから `bulk` へ進む。
   接続予算の判断は[キャパシティ設計](../design/performance/capacity.md#サイジング計算式)に従う。
9. `bulk` の Deployment `idmagic-worker-bulk` を同じ方法で戻す。
   準備完了レプリカ数と認証の成功を確かめ、PostgreSQL の接続使用率が引き続き 70% 未満であることを確かめる。
10. 最後に `idmagic-batch` の再開対象の CronJob を `suspend: false` に戻す。
    対象の再開を確認し、認証の成功と PostgreSQL の接続使用率を再確認する。

レーン別 Deployment の構成は[ワーカーの構成ファイル](../../infra/k8s/base/worker.yaml)を参照する。
手順 6 以降は[ロードシェディング順序](../design/performance/capacity.md#ロードシェディング順序)の逆に再開し、認証の中核へ接続枠を先に戻す。
確認に失敗した場合は、後続のレーンと CronJob を停止したまま原因を調べる。
署名鍵の検証に失敗した場合は、新しいトークンの発行を止め、[鍵プロバイダーの喪失への対応](#鍵プロバイダーの喪失)へ進む。

## 復元後の確認一覧

`mise run restore-postgres` の最後に自動実行される `idmagic-batch restore-consistency-check` が次を検査する。

- [ ] テナント、ユーザー、クライアントのレコード件数が 0 でない。
- [ ] 各テナントの有効な署名鍵を解決でき、JWKS を構成できる。
- [ ] `jobs` テーブルに `dedup_key` 違反（同じテナントと `dedup_key` に対する `queued` / `running` の重複）がない。
- [ ] 一時テーブル（`UNLOGGED` / `LOGGED`）が復元直後に空である。

手動でも次を確認する。

- [ ] 発行済みトークンを復旧後の JWKS で検証できる（`kid` が消えていない）。
- [ ] 代表テナントでログインとトークン発行を 1 回ずつ確認する。

## 災害復旧のシナリオ

### データベースの喪失

1. 直近の PITR ベースバックアップ（本番）または `pg_dump`（復元試験、小規模環境）から `mise run restore-postgres` を実行する。
2. 復元の整合順序に従い、整合性検査に合格するまで API を再開しない。
3. RPO は最後の WAL アーカイブまたはダンプからの経過時間、RTO は障害検知から整合性検査の合格までの経過時間で計測する。

### 誤削除からのポイントインタイム復旧

1. 影響範囲（テナント、ユーザー、クライアント）を監査ログ（`audit_events`）から特定する。
2. PITR で誤削除の直前へ復元した使い捨て環境を用意し、該当レコードだけを本番環境へ再投入する。全体のロールバックは避け、影響範囲を最小化する。
3. `pg_dump` による復元試験しか使えない環境では、直近の定期ダンプまでしか戻せないことを事前に周知する。アラートと確認ダイアログにより、誤削除の検知時間を短縮する運用を優先する。

### リージョンの喪失

複数リージョンの active-active 構成は対象外である。この手順書では、リージョン内の最新バックアップから代替リージョンに新しい環境を構築し、`restore-postgres` を実行する単一リージョンの再構築だけを扱う。RTO は環境の構築時間（IaC の適用）と復元時間の合計になる。

### 鍵プロバイダーの喪失

1. **PostgreSQL の復元で鍵の喪失を隠さない**: 整合性検査の署名鍵検証が失敗した場合は、データベースの復元が成功していてもフェイルクローズのまま新しいトークンの発行を止める。
2. Vault / OpenBao 側のスナップショットからキーストアを復旧できる場合は復旧し、`signing_keys` の公開鍵の写しと Vault 側の `kid` が一致することを確認する。
3. 復旧できない場合は緊急ローテーションへ進み、新しい署名鍵を発行して古い `kid` を JWKS から段階的に外す。この間、古い `kid` で署名されたトークンの検証を継続できないことを利用者へ告知する。

### テナントの DEK の緊急の入れ替え

バージョンの確認と操作は、制御面テナントの `system_admin` の対話セッションで行う。
ローテーションと無効化はそれを実行したプロセスの DEK の保持だけを破棄するので、`api` と `worker` の再起動を手順に含める（[DataKeys のリスク](../modules/data-keys/risks.md)）。

1. `GET /api/admin/v1/data-keys/health` で、対象の `tenant_id` と `active_version` を確かめる。
2. `POST /api/admin/v1/tenants/{tenant_id}/data-key/rotate` を実行する。応答の `previous_version` が漏洩したバージョンであることを確かめる。
3. `kubectl rollout restart` で `api` と `worker` を再起動し、新しい暗号化を新しいバージョンへ揃える。
4. 読み取りを止めるより値を残すことを優先できるなら、`data_key_reencryption` のジョブの完了を待つ。無効化の時点で移っていない値は読めなくなる。
5. `POST /api/admin/v1/tenants/{tenant_id}/data-key/versions/{version}/disable` を実行する。
6. もう一度 `api` と `worker` を再起動し、保持した DEK を捨てる。再起動が終わるまで、ほかのプロセスは無効化したバージョンで復号を続ける。
7. 監査イベントの `DataEncryptionKeyRotated` と `DataEncryptionKeyDisabled` に、操作者が残っていることを確かめる。

マスターキー自体のローテーションは OpenBao の `auto_rotate_period` で設定できる。
ただし、ラップ済みの DEK を新しいバージョンへ再ラップする経路はプロダクトにない。

### 部分的な復元

1. 対象範囲を限定した `pg_dump`（`pg_dump --table=...` など）からの復元は、外部キー制約や `jobs` / `signing_keys` などの整合性を壊しやすいため、整合性検査を必ず通す。
2. 部分的な復元後に全体の整合性を保てない場合は、全体の復元へ切り替える。ロールバックの判断基準は整合性検査の結果とする。

## 運用

| 運用 | 頻度または契機 | 内容 |
| --- | --- | --- |
| 定期的な復元試験 | 最低でも四半期に 1 回 | `mise run restore-drill` を実行し、バックアップの成功だけでなく復元できること自体を確認する。実測した RPO / RTO を下記の表に追記する |
| アクセスレビュー | 鍵ローテーションと同じ頻度 | 暗号化されたバックアップ保存先と Vault / OpenBao のスナップショットへのアクセス権限を棚卸しする |
| 有効期限 | バックアップの保持期間を過ぎたとき | 保持期間を明示し、期限切れのダンプを自動削除する運用を設ける |
| スケジュールとアラート | 復元試験の予定時刻と失敗時 | cron や Alertmanager などの運用基盤から、定期的な復元試験と失敗通知を実行する |

## 計測した RPO と RTO

| Date | Environment | Scenario | Measured RPO | Measured RTO | Notes |
| --- | --- | --- | --- | --- | --- |
| 2026-08-01 | ローカル Docker Compose（`mise run restore-drill`） | テナント 1 件、ユーザー 2 件、クライアント 3 件、署名鍵 1 件の `pg_dump` バックアップ → データベースの破棄・再作成 → 復元 → 整合性検査 | 0 秒（バックアップ直後に破棄したため差分は無視できる） | 6.3 秒（全体。うち復元と整合性検査は 2 秒） | データ量が少ないローカルの復元試験の下限値。本番規模の RPO / RTO は PITR を使うステージング環境で実測する。 |

Vault Transit を含むステージング環境での実際の復元試験は、開発環境に Vault / OpenBao サービスが存在しないため未実施である。実施可能な環境が用意され次第、上表に追記する。
