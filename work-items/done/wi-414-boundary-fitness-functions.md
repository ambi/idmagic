---
status: completed
archived_from: 04120c8709ea0e6707a733b761c279f008075673
---

# Context 境界と作用の禁止を機械検査する適応度関数を足す

`mise run spec-diff` は `no normative specification change against main` を返した。規範シナリオ、規範 ID、TypeSpec シンボルは変更していない。`docs/domain/structure.md` に Context の公開言語、Supplier/Customer と Go import の向き、循環禁止、domain の作用禁止、shared 経由の迂回禁止を現在設計として記述し、`check-boundaries` が `docs/design/architecture/logical.md` から Context と関係を導出して検査するようにした。既存の 855 具体違反は理由つきの 335 項目へ固定し、基準 revision から具体 ID が増える変更を CI の ratchet が拒否する。

全文は `git show 04120c8709ea0e6707a733b761c279f008075673:<このファイルのパス>` で読む。
