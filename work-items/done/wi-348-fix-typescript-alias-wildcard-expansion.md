---
status: completed
archived_from: 8edc9a416a5f5141a162135d61dd417453bb5872
---

# TypeScript alias の wildcard を完全かつリテラルに展開する

TypeScript alias target の全 wildcard を callback 形式の `replaceAll` で展開し、capture を
JavaScript の置換記法ではなくリテラル値として扱うようにした。複数 wildcard と `$&`
capture の回帰テストを追加した。

全文は `git show 8edc9a416a5f5141a162135d61dd417453bb5872:<このファイルのパス>` で読む。
