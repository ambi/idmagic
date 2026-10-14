---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 開発方式を specification-first workflow として再定義し仕様・設計文書体系を簡素化する

Regenerative Architecture / SCL 固有の方法論・CLI・ledger を、TypeSpec と root/context
`SPECIFICATION.md` を正本とする specification-first development workflow へ簡素化した。
requirements と source-side architecture 文書を context ごとの単一文書へ統合し、OpenAPI の
context tag と navigation 付き HTML view を追加した。独立 Requirements section は本文・条件の
意味監査で他の正本へ被覆できることを確認したうえで削除した。
scenario 文法は番号参照を廃止し、`ALT` を対象の `WHEN` / `THEN` の直下に置く形式にした。
全244 scenario を個別に読み、`GIVEN` は事前状態、367件の `WHEN` は操作・入力・外部 event、
564件の `THEN` は観測結果となるよう再分類・分割した。242件の `ALT` は対象 step の子要素へ
配置し、番号付き step は0件になった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
