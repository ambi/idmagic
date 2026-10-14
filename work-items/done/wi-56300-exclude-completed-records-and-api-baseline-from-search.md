---
status: completed
archived_from: 8f9e9a564f2dc8fee4044bfcd80edff091bee9ca
---

# 完了した作業記録と OpenAPI のベースラインを既定の検索対象から外す

`mise run spec-diff` に規範の差分はない。
リポジトリ直下の `.ignore` で `work-items/done/` と `spec/*.openapi.baseline.json` を `rg`、`fd`、`ast-grep` の既定の検索から外し、探し方をリポジトリ構成へ 1 行で記した。
`spec-where` の作業項目の欄は未完了の記録だけを返すようになる。現在の用語の所在を探すタスクなので、このまま変えない。

全文は `git show 8f9e9a564f2dc8fee4044bfcd80edff091bee9ca:<このファイルのパス>` で読む。
