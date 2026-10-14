---
status: completed
archived_from: bd8a165429d3da93b8fd5887ae35f843b1c77712
---

# 受け入れ例の共通前提をまとめ、保証と検出能力を維持して読む量を減らす

`mise run spec-diff` は main に対して規範仕様の差分を報告しない。
例の検査と `spec-diff` は、`Rule` 直下の `Background` の手順を各例の先頭に足し、`Scenario Outline` の行の値を手順へ埋めた経路で例を読む。
この比較の上で、受け入れ例の付録 49 文書の共通の前提を `Background` へ、値だけ異なる例を `Scenario Outline` へまとめ、付録の合計を 8,293 行から 7,710 行へ減らした。
グループ CSV は 429 行から 330 行、OAuth2 承認は 301 行から 213 行、OAuth2 トークンは 427 行から 398 行になった。
EX ID は 942 件のまま変わらず、テストの参照もそのまま解決する。

全文は `git show bd8a165429d3da93b8fd5887ae35f843b1c77712:<このファイルのパス>` で読む。
