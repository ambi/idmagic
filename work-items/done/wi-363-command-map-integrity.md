---
status: completed
archived_from: 0d89fe6d56243c535512758cbada9f7034aeb79d
---

# 壊れた CI 参照を直し、just --list をコマンドマップとして読める状態にする

CI が呼んでいた存在しない `just traceability-strict` を除去し、SCL を指していた job 名を
「Verify specification, backend, and UI」に直した。これで 2026-08-10 以降続いていた main の CI 失敗が
解消する見込み。再発防止として `just check-command-map` を追加し、`.github/workflows/**` が呼ぶ
`just <recipe>` が `justfile` に実在することを `just check` の一部として検査する。逆方向（recipe が
CI から使われているか）は検査しない。
`just --list` の説明は、`just` が直前コメントの最終行だけを表示する仕様に合わせて整えた。運用上の
注意を失わないよう、複数行の説明は `[doc("...")]` 属性で 1 行の説明を与えたうえでコメント本文を
残す形にした。`deploy-monitoring-operator` に説明を追加し、recipe コメントの `(wi-101)` /
`(wi-308)` 参照を除去した。recipe の統廃合は行っていない。参照実績を確認した結果、
`verify-go` / `verify-ui` / `verify-spec` / `restore-drill` / `k6-smoke` などはいずれも README や
work item から使われており、削減対象ではなかった。

全文は `git show 0d89fe6d56243c535512758cbada9f7034aeb79d:<このファイルのパス>` で読む。
