---
status: completed
archived_from: 9556f99b4babf547b28b825a815e3f64c2359d1e
---

# 生成ドキュメントサイトの名称と配置を実体に合わせる

`mise run spec-diff` は規範的なプロダクト仕様の差分がないことを報告した。
生成サイトを「IdMagic ドキュメント」として `site/` に配置し、トップページの区分を閉じ、文書ページでは現在位置を含む区分だけを開くようにした。
API リファレンスが公開ディレクトリ内で完結するよう、生成済み OpenAPI を `site/openapi/` に複製する。

全文は `git show 9556f99b4babf547b28b825a815e3f64c2359d1e:<このファイルのパス>` で読む。
