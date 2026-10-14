---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 壊れた Markdown リンクを検出する検査を、仕組みとして持つ

`mise run spec-diff` は `no normative specification change against main` を返した。製品仕様は変更していない。Markdown のインラインリンク、参照定義、相対パス、見出しアンカー、明示的な HTML アンカーを検査する `mise run check-links` を追加し、標準の `mise run check` へ組み込んだ。検査対象を 0 件にして成功する退行を拒否し、コードフェンスとコードスパンを除外しながら複数行リンクを解析する。実装時に見つかった既存の壊れた参照 25 件も修正した。現在は 603 文書を検査する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
