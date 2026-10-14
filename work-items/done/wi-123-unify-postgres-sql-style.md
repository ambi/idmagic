---
status: completed
archived_from: 0fca46fb861aad86ad8b000c8feebf5172954e84
---

# PostgreSQL スキーマの SQL 記述スタイル統一

`deploy/schema/postgres.sql` において、データベーススキーマの記述スタイルを統一した。
- 複合キーではない PRIMARY KEY をカラム定義の直後にインラインで定義するよう統一。
- すべての外部キー制約について、インライン記述や制約名なしの記述を廃止し、別行で `CONSTRAINT <table_name>_<column_name>_fkey` と明示的に制約名を付与して定義するよう統一。
- 通常の UNIQUE 制約について、外部の `CREATE UNIQUE INDEX` やインライン定義を廃止し、`CREATE TABLE` 内の別行で `CONSTRAINT <table_name>_<column_name>_key UNIQUE` として定義するよう統一（部分インデックスを除く）。
また、スキーマ変更に伴い、`internal/shared/adapters/persistence/postgres/schema_test.go` の制約名の検証コードも更新し、テストが正常にパスすることを確認した。

全文は `git show 0fca46fb861aad86ad8b000c8feebf5172954e84:<このファイルのパス>` で読む。
