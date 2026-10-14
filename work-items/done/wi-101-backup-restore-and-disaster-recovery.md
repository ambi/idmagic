---
status: completed
archived_from: 8e1d4ceb14f1897796ede5709ae6f19fd0c7da30
---

# PostgreSQL・署名鍵のバックアップ／リストアと DR runbook（RPO/RTO 目標付き）を整備する

当初 wi-101 は「PostgreSQL・Valkey・署名鍵」の3系統を前提にしていたが、
ADR-139 (揮発性状態を単一 PostgreSQL へ統合し Valkey を撤去) と ADR-147
(外部メッセージブローカーと `idmagic-relay` を撤去) により前提が古くなって
いたため、まず Motivation/Scope/Plan/Tasks から Valkey・`postgres_valkey`・
outbox/relay 前提の記述を除去し、ADR-139/147 後の実態 (単一 PostgreSQL、
UNLOGGED/LOGGED ephemeral テーブル、`jobs` テーブルの dedup_key 整合) に
書き換えた。その上で [[ADR-153-backup-restore-and-disaster-recovery]] で
バックアップ対象の分類・RPO/RTO 方針・リストア整合順序を決定し、
`docs/operations/runbooks/backup-restore-dr.md`
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

全文は `git show 8e1d4ceb14f1897796ede5709ae6f19fd0c7da30:<このファイルのパス>` で読む。
