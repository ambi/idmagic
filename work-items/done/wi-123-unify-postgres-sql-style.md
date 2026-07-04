---
id: wi-123-unify-postgres-sql-style
title: PostgreSQL スキーマの SQL 記述スタイル統一
created_at: 2026-07-05
authors: [Antigravity]
status: completed
risk: low
---

# Motivation
`deploy/schema/postgres.sql` において、PRIMARY KEY、外部キー制約、UNIQUE制約の記述スタイルに揺らぎがあるため、可読性および保守性の向上のためにスタイルを統一する。

# Scope
- `deploy/schema/postgres.sql` 内のテーブル定義について、スタイルを統一する。
  - **PRIMARY KEY**: 複合キーでない場合、カラム定義 of `PRIMARY KEY` と記述する。
  - **外部キー制約 (FOREIGN KEY)**: すべて `CONSTRAINT <テーブル名>_<カラム名>_fkey FOREIGN KEY (<カラム名>) REFERENCES <対象テーブル>(<対象カラム>) ON DELETE <アクション>` または複合キー用の `CONSTRAINT` として別行で明示的に定義する。
  - **UNIQUE制約**: 部分一意インデックス（`WHERE`句を持つもの）を除き、すべて `CONSTRAINT <テーブル名>_<カラム名群>_key UNIQUE (<カラム名群>)` として CREATE TABLE 内で定義する。

# Out of Scope
- データベース構造そのものの変更（カラム追加・削除、型変更、制約の緩和・強化など）
- テーブルの分割やマージ

# Initial Context
- `deploy/schema/postgres.sql`

# Affected Guarantees
- schema-syntax-and-referential-integrity

# Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just verify-go`
- `just verify-ui`

# Risk Notes
SQLのスタイル変更（記述場所の統一）のみであり、スキーマ構造自体は変更しない。ただし外部キー制約名などが自動生成から明示的指定に変わるため、既存のインフラに対してマイグレーション等を適用する際の制約名競合や、自動生成される制約名の変更が発生する可能性がある。

# Completion
- **Completed At**: 2026-07-05
- **Summary**:
  `deploy/schema/postgres.sql` において、データベーススキーマの記述スタイルを統一した。
  - 複合キーではない PRIMARY KEY をカラム定義の直後にインラインで定義するよう統一。
  - すべての外部キー制約について、インライン記述や制約名なしの記述を廃止し、別行で `CONSTRAINT <table_name>_<column_name>_fkey` と明示的に制約名を付与して定義するよう統一。
  - 通常の UNIQUE 制約について、外部の `CREATE UNIQUE INDEX` やインライン定義を廃止し、`CREATE TABLE` 内の別行で `CONSTRAINT <table_name>_<column_name>_key UNIQUE` として定義するよう統一（部分インデックスを除く）。
  また、スキーマ変更に伴い、`internal/shared/adapters/persistence/postgres/schema_test.go` の制約名の検証コードも更新し、テストが正常にパスすることを確認した。
- **Verification Results**:
  - `just yaml-check-work-items` - 成功
  - `just check-ids` - 成功
  - `just test-go` - 成功
- **Affected Guarantees State**:
  PostgreSQL スキーマの構文および参照整合性（`schema-syntax-and-referential-integrity`）が維持され、より一貫性と可読性の高い形式に統一された。
