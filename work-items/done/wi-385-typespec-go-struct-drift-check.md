---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# TypeSpec が宣言する要求・応答本体と Go が実際に読み書きするものを機械的に突き合わせる

`mise run spec-diff` は `added TypeSpec declarations: UpdateAuthorizationDetailTypeRequest, AdminSettingsUpdateRequest` を返す。`mise run check-contract-drift` を足し、`mise run check` に組み込んだ。operation ごとに、生成 OpenAPI の要求・応答本体と、Go の route → handler → decode 先構造体を突き合わせ、要求プロパティの差 (D1)、path/query と本体の二重宣言 (D2)、応答の第 1 段キーの差 (D3) を落とす。入れ子モデルへも降りる —— `wi-381` の欠陥 (`UserAttributeDef` が 10 項目中 2 つしか宣言しない) は入れ子にあり、第 1 段だけを見る検査では再発を捕まえられない。
**検査が見つけた 14 件はすべて解決した。** 許可リストも閾値も置いていない。実ツリーは 333 operation 中 184 を突き合わせ、149 は追えていない (大半は応答を変数で返す handler)。この件数は毎回の出力に書く。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
