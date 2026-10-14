---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# PostgreSQL schema を sqldef で宣言的に管理する

PostgreSQL schema を宣言的 sqldef/psqldef 管理へ移行した。現在形 schema を
`deploy/schema/postgres.sql` に正本として集約し、`deploy/schema/README.md` に
dry-run / apply / drift check 手順を追加した。番号付き migration ディレクトリ、
`schema_migrations` テーブル、起動時 migration runner を削除し、bootstrap は
外部 CLI を起動時に呼ばない。決定は ADR-071 に記録し、README / ARCHITECTURE の
persistence 運用規約を「構造変更は schema 正本・データ移行は別管理」へ更新した。
SCL の contract は変更していない (外側 adapter の再生成形のみ)。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
