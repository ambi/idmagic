---
status: completed
---

# 手元の Go テストは race なしを既定にし、race は並行処理を作る作業で明示して選ぶ

`mise run spec-diff` は main に対して規範の差分なしを示す。
手元の Go のテストは通常のビルドを既定にした。`test-go-package` と `test-go-changed` は末尾に `--race` を付けたときだけ race 検出器を使い、`verify`、`verify-serial`、`verify-go` の Go テストは `test-go-race` から `test-go` に変えた。`test-go-race` は Go 全体の race 検査として残る。
`report-verification` が既定で読む報告は、`verify` が書く `.verification/test-go.json` にした。
`testing.md` の「Go の通常テストと race 検査」は、変更ごとに race の要否を判断させる規則をやめ、data race の自動検査は main への定期実行（wi-47187）だけが担い、並行処理そのものを作る作業だけが明示して race を選ぶ、とした。
検証の設計と形式の文書で Go のテストのタスクとして挙げていた `test-go-race` は `test-go` にした。
引数は、race なし、`--race`、誤った引数、引数なしの四通りで、報告の `command` と終了コードが期待どおりになることを確かめた。
最終検査の `verify-changed` は `mise.toml` の変更で全層を選び、frontend の依存を入れたうえですべて通った。
wi-47187 の定期実行は 2026-10-29 に Actions 上で、main での成功と、故意の data race での失敗を確かめた。
