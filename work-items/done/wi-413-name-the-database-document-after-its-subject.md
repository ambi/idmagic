---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# データベース設計文書を主題の名前にする — `persistence.md` を `database.md` にする

`docs/persistence.md` を `docs/database.md` へ改名し、見出しを `# Database` にした。本文は 1 行も変えていない。`ROOT_DOCUMENTS` を張り替え、正規文書の集合が改名後の名前を指すようにした。参照は 4 種類——索引（`docs/README.md`）、本文からの相対リンク（`docs/domain/glossary.md`、`authentication/internals.md` 2 か所、`tenancy/internals.md`。後ろの 3 つは `#tenant_id-retention-classes` のアンカー付き）、スキーマ側からの参照（`infra/schema/README.md` 2 か所）、Go コメント（`datakeys/db_memory`、`datakeys/usecases`、`jobs/db_postgres`）——をすべて張り替えた。`SPECIFICATION_FORMAT.md` の構成図と、進行中の work item 5 件（wi-164、wi-186、wi-282、wi-293、wi-295）も追随させた。節見出しを変えていないのでアンカーはそのまま解決する。設定値の `persistence=postgres` と、層を指す一般名詞としての `persistence` には触れていない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
