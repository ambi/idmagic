---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# `lint-go` の実行時間の 86% を占める goimports を外し、失われる検査が無いことを示す

`mise run spec-diff main` は規範差分なしを報告する。変わったのは Go の lint ゲートの構成だけで、`formatters` から `goimports` が外れ、一度も一致していなかった `goimports.local-prefixes: idmagic` の設定が消え、`gofumpt.module-path` が実際のモジュールパスに直った。検出される指摘の集合は変わらない (変更前後とも 0 件、`mise run format-go` の出力も同一)。変わったのは費用で、cold 実行が 103.2 秒から 15.1 秒、`backend/shared/kernel` を 1 行触った後が 77 秒から 10 秒になった。あわせて `mise run time-lint-go` を追加し、この内訳を再現できるようにした。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
