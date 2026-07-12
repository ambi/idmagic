---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-04
---

# PostgreSQL・Valkey・署名鍵のバックアップ／リストアと DR runbook（RPO/RTO 目標付き）を整備する

## Motivation
WI-11（運用資産）はバックアップ・リストア自動化、DR、マルチリージョンを
明示的に out of scope としており、他のどの WI もこれを扱っていない。しかし
IdP はダウンすると全依存システムの認証が止まる単一障害点であり、
「復旧できるか」「どこまで戻るか（RPO）」「どれだけで戻せるか（RTO）」が
未定義のままではプロダクションレディとは言えない。特に tenant-scoped signing key を
失うと発行済みトークンの検証系全体が壊れるため、鍵の退避と復旧は最優先。

Keycloak は本番運用ガイドで DB バックアップ／リストアとダウングレード不可の
スキーマ整合を明記し、realm 単位のエクスポートも提供する。idmagic も
永続層（PostgreSQL の構造 + データ）、揮発層（Valkey に載る session/PAR/code の
復旧不能性の割り切り）、鍵素材（KeyStore / Vault Transit 参照）ごとに
バックアップ対象・整合順序・リストア手順を runbook として持つべきである。

## Scope
- **decision**: 新規 ADR: バックアップ対象の分類（永続 = PostgreSQL、揮発 = Valkey、鍵 = KeyStore/Vault）、 各層の RPO/RTO 目標、リストア時の整合順序、鍵と DB の版整合（鍵ローテーション中の 復旧）方針を定義する。Valkey 揮発データは復旧対象外とする割り切りを明記する。
- **documentation**: PostgreSQL の論理／物理バックアップ手順（pg_dump / PITR いずれかを選択）と、 infra/schema/postgres.sql の宣言的スキーマとの整合手順を runbook 化する。, Vault Transit / envelope encryption（WI-97 と整合）配下の鍵素材の退避・復旧手順を書く。 平文鍵をバックアップに残さない前提を明記する。, リストア後の検証チェックリスト（JWKS 継続性、既発行トークン検証、tenant 疎通）を作る。, 障害シナリオ別 DR 手順（DB 喪失 / リージョン喪失 / 鍵喪失 / 誤削除）を runbook にまとめる。
- **tooling**: バックアップ取得・リストアを再現するスクリプト（またはジョブ定義）を deploy 配下に置き、 ローカル docker compose で restore drill を実行できるようにする。

## Out of Scope
- マルチリージョンのアクティブ／アクティブ構成。
- 特定クラウドのマネージドバックアップ製品への依存実装。
- アプリケーションロジック・HTTP API の変更。
- 自動フェイルオーバーのオーケストレーション。

## Plan
- [[ADR-016-persistence-adapter-selection]] に従いPostgreSQL（業務状態/event log/outbox/key metadata）をdurable正本、Valkey（authorization/session/replay/throttle）を再生成・全失効可能なvolatile状態として分類する。署名private keyがVault Transitの場合はVault側backup/DRを別保護対象にする。
- PostgreSQLはproduction規模/RPOに必要なPITR（base backup+WAL archive）を正本とし、`pg_dump`はportable export/小規模drill用途に限定する。`infra/schema/postgres.sql`は復元後の差分検証に使い、空DBへ先にapplyしてdata restoreと競合させない。
- recovery順はKMS/Vault access→PostgreSQL restore→schema/version検査→Valkey flush→API/worker/relay起動→outbox再開→JWKS/token/session検証とする。Valkeyをbackupから戻してstale session/replay stateを復活させない。
- signing keyはDB metadataとprovider key versionの整合をcheckし、鍵喪失を「DBだけrestore」で隠さない。既発行token検証不能時のfail-closedとemergency rotation手順を分ける。
- toolingはoperatorが意図したenvironment/backup IDを明示し、restore先non-production guard、checksum/encryption、監査、dry-runを持つ。定期drillで実測RPO/RTOとevidenceを記録する。

## Tasks
- [ ] T001 [Inventory/ADR] table/provider/secret/object storeを分類し、障害シナリオ別RPO/RTO、PITR、Valkey discard、鍵整合、責任者を決定する。
- [ ] T002 [Backup] PostgreSQL base/WAL archive、encryption/checksum/retention/restore-pointとVault/provider backup確認をdeploy資産に追加する。
- [ ] T003 [Restore] explicit target/time、non-production guard、停止/復元/schema検査/Valkey flush/start順を自動化するjust recipeを追加する。
- [ ] T004 [Consistency] tenant/user/client、event-log/outbox cursor、signing key/JWKS、token verify、relay duplicateを検査するpost-restore toolを実装する。
- [ ] T005 [Runbooks] DB loss、point-in-time誤削除、region loss、key provider loss、partial restoreごとのdecision treeとrollback/escalationを記載する。
- [ ] T006 [Drill] disposable composeでfull restoreとPITRを実行し、次にstagingで鍵providerを含むdrillを行い、実測RPO/RTOとartifact hashを記録する。
- [ ] T007 [Operations] schedule/alert、backup成功だけでなくrestore可能性を確認する定期drill、アクセスレビューとexpiryを運用化する。

## Verification
- 手動: docker compose 上で PostgreSQL をバックアップ → 破棄 → リストアし、 tenant / user / client / 監査が復元され JWKS が継続することを確認する。
- 手動: 鍵素材の退避・復旧手順で、既発行トークンの検証が復旧後も通ることを確認する。
- 手動: runbook の各 DR シナリオを机上またはドリルで一度たどり、抜けを潰す。

## Risk Notes
誤ったリストア手順は tenant 混在や鍵不整合という不可逆な事故を生む。
まず restore drill を CI/ローカルで反復可能にし、手順が実際に通ることを
継続検証してから本番手順として確定する。鍵と DB の版整合を最優先で検証する。
