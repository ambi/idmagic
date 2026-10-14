---
status: completed
archived_from: ae0ce5e5e2c87f8b1f0610646319ccb96a14ea98
---

# 生成仕様サイトのナビゲーションと参照ページを読める形に整える

生成仕様サイトのサイドバーが、ファイル名のアルファベット順ではなく、正規レイアウトが定める順序と各文書の H1 を示すようになった。グループは 4 つとも同じ折りたたみで、Method だけが既定で閉じ、Whole System の子は現在地が Whole System のときだけ並ぶ。Glossary の Term 列は語の途中で折り返されない。各 context の index ページは、自分が所有する API 操作とモデルを持ち、タグで絞った API Reference と Model Catalog の該当見出しへ降りられる。Model Catalog の見出しは、意味を持たない単一名前空間ではなく所有 context になった。TypeSpec の doc コメントは Markdown として描画される。Model Catalog の Filter models は、サイトスクリプトが読み込まれていなかったため入力しても何も起きない状態だったが、実際に絞り込むようになった。

全文は `git show ae0ce5e5e2c87f8b1f0610646319ccb96a14ea98:<このファイルのパス>` で読む。
