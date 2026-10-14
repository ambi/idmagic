---
status: completed
archived_from: 33c4336e34d567e789644ccdd638904c6f52ccab
---

# 作業開始時の setup を 1 回にまとめ、狭い段の並列実行を手順に書く

`mise run spec-diff` は main に対して規範仕様の差分を報告しない。変わったのは実装中の手順だけである。
`implement-work-item` skill の第 1 段は、読み取りと並べて `mise run setup` と `mise run lint-go` を一度払う指示を持つ。`setup` が `setup-tools` と `install-ui` をまとめるので、Design が挙げた 3 つの準備は 2 行で呼べる。
第 8 段と `docs/development/local-development.md` は、狭いタスクを `mise run -c <task> <args> ::: <task> <args>` で並べる書き方と、`--` を挟むと後続が黙って走らないという注意を持つ。

全文は `git show 33c4336e34d567e789644ccdd638904c6f52ccab:<このファイルのパス>` で読む。
