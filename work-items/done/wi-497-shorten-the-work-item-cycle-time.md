---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# work-item 1 件あたりの所要時間を、実測した律速から順に短縮する

`mise run spec-diff main` は規範差分なしを報告する。TypeSpec の宣言も `docs/` の規範シナリオも変えておらず、変わったのは検証ゲートの構成、狭い段のビルド構成、テストが払う鍵導出コスト、E2E の起動回数、検査の出力量、着手時の入口、および手順文書である。観測できる差分は所要時間と出力行数に出る。`test-go-race` は 72 秒から 54 秒、`test-ui-e2e` は 56 秒から 28 秒、`mise run check` の出力は 685 行から 50 行、`test-ui-e2e` の出力は 41,515 行から 10 行になった。集約ゲートは 1 回の実行で落ちたゲートを全部報告するようになり、`test-ui-e2e` は `verify` から `verify-full` と CI の独立した job へ移った。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
