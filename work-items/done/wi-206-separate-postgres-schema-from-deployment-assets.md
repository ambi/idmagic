---
status: completed
archived_from: 31bf9bd9dbbfb17f7719dfc18475954ba4966df0
---

# deploy を infra に改名し PostgreSQL schema の infrastructure 所有を明確にする

- `deploy/` を削除し、Docker/Compose/OTel 資材を `infra/docker/`、PostgreSQL の宣言的 schema と
運用手順を `infra/schema/` へ移した。
- sqlc、embedded PostgreSQL テスト、開発用 infrastructure CLI、Docker/Compose、CI、README、
現行 work item の参照を新しい正本へ更新した。
- ADR-102 に Runtime と database infrastructure 資材の配置判断を記録し、ADR-071 と
`ARCHITECTURE.md` を現状へ同期した。

全文は `git show 31bf9bd9dbbfb17f7719dfc18475954ba4966df0:<このファイルのパス>` で読む。
