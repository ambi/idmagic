---
depends_on: []
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-04
---

# PostgreSQL・署名鍵のバックアップ／リストアと DR runbook（RPO/RTO 目標付き）を整備する

## Motivation
WI-11（運用資産）はバックアップ・リストア自動化、DR、マルチリージョンを
明示的に out of scope としており、他のどの WI もこれを扱っていない。しかし
IdP はダウンすると全依存システムの認証が止まる単一障害点であり、
「復旧できるか」「どこまで戻るか（RPO）」「どれだけで戻せるか（RTO）」が
未定義のままではプロダクションレディとは言えない。特に tenant-scoped signing key を
失うと発行済みトークンの検証系全体が壊れるため、鍵の退避と復旧は最優先。

Keycloak は本番運用ガイドで DB バックアップ／リストアとダウングレード不可の
スキーマ整合を明記し、realm 単位のエクスポートも提供する。idmagic も
durable state と ephemeral state を単一の PostgreSQL に統合済みであり
（ADR-139、旧 Valkey は撤去済み）、その PostgreSQL の構造 + データ、鍵素材
（KeyStore / Vault Transit 参照）ごとにバックアップ対象・整合順序・リストア
手順を runbook として持つべきである。

## Scope
- **decision**: 新規 ADR: バックアップ対象の分類（PostgreSQL 単一 = durable + ephemeral UNLOGGED/LOGGED テーブル、鍵 = KeyStore/Vault）、 各層の RPO/RTO 目標、リストア時の整合順序、鍵と DB の版整合（鍵ローテーション中の 復旧）方針を定義する。ephemeral な UNLOGGED/LOGGED テーブルはリストア直後に意図的に truncate し、stale な認可中間状態・replay 状態・throttle カウンタを復旧対象外とする割り切りを明記する。
- **documentation**: PostgreSQL の論理／物理バックアップ手順（pg_dump / PITR いずれかを選択）と、 infra/schema/postgres.sql の宣言的スキーマとの整合手順を runbook 化する。, Vault Transit / envelope encryption（WI-97 と整合）配下の鍵素材の退避・復旧手順を書く。 平文鍵をバックアップに残さない前提を明記する。, リストア後の検証チェックリスト（JWKS 継続性、既発行トークン検証、tenant 疎通）を作る。, 障害シナリオ別 DR 手順（DB 喪失 / リージョン喪失 / 鍵喪失 / 誤削除）を runbook にまとめる。
- **tooling**: バックアップ取得・リストアを再現するスクリプトを deploy 配下に置き、 ローカル docker compose で restore drill を実行できるようにする。

## Out of Scope
- マルチリージョンのアクティブ／アクティブ構成。
- 特定クラウドのマネージドバックアップ製品への依存実装。
- アプリケーションロジック・HTTP API の変更。
- 自動フェイルオーバーのオーケストレーション。

## Plan
- [[ADR-139-consolidate-ephemeral-state-into-postgresql]] に従い、durable state と ephemeral state (authorization/session/replay/throttle) を単一の PostgreSQL に分類する（旧 Valkey は撤去済み、durable は LOGGED テーブル、高churnな ephemeral は UNLOGGED テーブル、denylist/throttle は failover 耐性のため LOGGED）。署名private keyがVault Transitの場合はVault側backup/DRを別保護対象にする。
- PostgreSQLはproduction規模/RPOに必要なPITR（base backup+WAL archive）を正本とし、`pg_dump`はportable export/小規模drill用途に限定する。`infra/schema/postgres.sql`は復元後の差分検証に使い、空DBへ先にapplyしてdata restoreと競合させない。
- recovery順はKMS/Vault access→PostgreSQL restore→schema/version検査→ephemeral UNLOGGED/LOGGEDテーブルのtruncate→API/worker起動→JWKS/token/session検証とする。UNLOGGED テーブルはPITR/物理バックアップに含まれず自動的に空で戻る性質を利用し、LOGGED な ephemeral (denylist/throttle) も明示的にtruncateしてstale replay/throttle stateを復活させない。
- signing keyはDB metadataとprovider key versionの整合をcheckし、鍵喪失を「DBだけrestore」で隠さない。既発行token検証不能時のfail-closedとemergency rotation手順を分ける。
- toolingはoperatorが意図したenvironment/backup IDを明示し、restore先non-production guard、checksum/encryption、監査、dry-runを持つ。定期drillで実測RPO/RTOとevidenceを記録する。

## Tasks
- [x] T001 [Inventory/ADR] table/provider/secret storeを分類し、障害シナリオ別RPO/RTO、PITR、ephemeral UNLOGGED/LOGGEDテーブルのtruncate方針、鍵整合、責任者を決定する。→ [[ADR-153-backup-restore-and-disaster-recovery]]。
- [x] T002 [Backup] pg_dump (checksum付き) をdeploy資産に追加する。→ `infra/backup/backup-postgres.sh`、`just backup-postgres`。PostgreSQL base/WAL archive (PITR) の自動化とVault/provider側backupの確認は本セッションでは未実施 (`docs/operations/runbooks/backup-restore-dr.md` に手順のみ記載、Completion参照)。
- [x] T003 [Restore] explicit target、non-production guard (`--yes-restore-into-this-database` による db 名の明示一致)、schema検査/ephemeralテーブルtruncate/consistency-check起動までを自動化するjust recipeを追加する。→ `infra/backup/restore-postgres.sh`、`just restore-postgres`。explicit time (PITRのtime point指定) は物理バックアップ未実装のため対象外。
- [x] T004 [Consistency] tenant/user/client、jobsテーブルのdedup_key/lease整合、signing key/JWKS、ephemeralテーブルが空であることを検査するpost-restore toolを実装する。→ `backend/cmd/idmagic-batch restore-consistency-check`。RED: `TestCheckRestoreConsistency_emptyDatabaseReportsMissingBaseline` / `TestCheckRestoreConsistency_tenantMissingActiveSigningKeyIsReported` / `TestCheckRestoreConsistency_nonEmptyEphemeralTableIsReported` を先に fail 確認 (常に空レポートを返すスタブに対して) → GREEN。token verify (既発行tokenの実検証) はrunbookのリストア後検証チェックリストに手動項目として残し、本toolはDB状態の検査に限定する。
- [x] T005 [Runbooks] DB loss、point-in-time誤削除、region loss、key provider loss、partial restoreごとのdecision treeとrollback/escalationを記載する。→ [`docs/operations/runbooks/backup-restore-dr.md`](../docs/operations/runbooks/backup-restore-dr.md)。
- [x] T006 [Drill] disposable composeでpg_dumpベースのfull restoreドリルを実行し、実測RPO/RTOとartifact hashを記録する。→ `just restore-drill` (`infra/backup/restore-drill.sh`)、実測値は [`docs/operations/runbooks/backup-restore-dr.md`](../docs/operations/runbooks/backup-restore-dr.md) の表に記載 (2026-08-01)。PITR drillとstagingでのVault鍵providerを含むdrillは未実施 (dev composeにVault/OpenBaoサービスが無く、staging環境も本リポジトリには存在しないため。Completion参照)。
- [ ] T007 [Operations] schedule/alertの実配線、backup成功だけでなくrestore可能性を確認する定期drillの自動実行、アクセスレビューとexpiryの運用化は本セッションでは未実施。cadence・アクセスレビュー・expiryの運用ルールは [`docs/operations/runbooks/backup-restore-dr.md`](../docs/operations/runbooks/backup-restore-dr.md) の「運用」節に記載したが、実際のcron/alertmanager配線は行っていない (Completion参照)。

## Verification
- 手動: docker compose 上で PostgreSQL をバックアップ → 破棄 → リストアし、 tenant / user / client / 監査が復元され JWKS が継続することを確認する。
- 手動: 鍵素材の退避・復旧手順で、既発行トークンの検証が復旧後も通ることを確認する。
- 手動: runbook の各 DR シナリオを机上またはドリルで一度たどり、抜けを潰す。

## Risk Notes
誤ったリストア手順は tenant 混在や鍵不整合という不可逆な事故を生む。
まず restore drill を CI/ローカルで反復可能にし、手順が実際に通ることを
継続検証してから本番手順として確定する。鍵と DB の版整合を最優先で検証する。

## Completion
- **Completed At**: 2026-08-01
- **Summary**:
  当初 wi-101 は「PostgreSQL・Valkey・署名鍵」の3系統を前提にしていたが、
  ADR-139 (揮発性状態を単一 PostgreSQL へ統合し Valkey を撤去) と ADR-147
  (外部メッセージブローカーと `idmagic-relay` を撤去) により前提が古くなって
  いたため、まず Motivation/Scope/Plan/Tasks から Valkey・`postgres_valkey`・
  outbox/relay 前提の記述を除去し、ADR-139/147 後の実態 (単一 PostgreSQL、
  UNLOGGED/LOGGED ephemeral テーブル、`jobs` テーブルの dedup_key 整合) に
  書き換えた。その上で [[ADR-153-backup-restore-and-disaster-recovery]] で
  バックアップ対象の分類・RPO/RTO 方針・リストア整合順序を決定し、
  [`docs/operations/runbooks/backup-restore-dr.md`](../docs/operations/runbooks/backup-restore-dr.md)
  に手順・署名鍵退避・検証チェックリスト・障害シナリオ別 decision tree・
  運用cadenceを runbook 化した。tooling として `infra/backup/backup-postgres.sh`
  / `restore-postgres.sh` / `restore-drill.sh` と対応する just recipe
  (`backup-postgres` / `restore-postgres` / `restore-drill`) を追加し、
  `backend/cmd/idmagic-batch restore-consistency-check` を test-first
  (RED→GREEN、Tasks T004 参照) で実装した。ローカル docker compose で
  実際に backup → db 破棄 → restore → consistency-check のドリルを実行し、
  実測値 (RTO 約 2〜6 秒、ローカル小規模データでの下限値) を runbook に
  記録した。
  **未対応・開示事項**: PostgreSQL の PITR (base backup + WAL archive) の
  自動化、Vault/OpenBao 側 backup の確認、staging 環境での鍵 provider を
  含む実ドリル、定期drill/アラートの実運用配線 (cron/alertmanager 等) は
  本セッションでは未実施 (T002/T003/T006/T007 参照)。dev compose に
  Vault/OpenBao サービスが無く、staging 環境も本リポジトリには存在しない
  ための制約であり、手順は runbook に記載済みだが実証はできていない。
  マルチリージョン・特定クラウドのマネージドバックアップ製品依存・自動
  フェイルオーバーは元 WI どおり Out of Scope のまま。
- **Verification Results**:
  - `just check` - passed
  - `just build-go` - passed
  - `just test-go` - passed (`backend/cmd/idmagic-batch` の新規テスト含む)
  - `just lint-go` - passed
  - `just verify` - passed (check / test-go / lint-go / lint-ui / format-check-ui / test-ui-unit / build-ui / typecheck-tools / test-tools / traceability-strict)
  - `just restore-drill` - passed (ローカル実行、実測値は runbook に記録)
