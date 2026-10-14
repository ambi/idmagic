---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# リポジトリツールの責任を整理し、検査の実行基盤をまとめる

`tools/check/` は 14 個の規則を `registry.ts` で一度だけ対応づけ、`runner.ts` が一つの
snapshot に対して並行実行する構成になった。個別の `mise run check-*` は残したが、すべて
同じランナーへ selector を渡す。集約実行は途中の例外を所見へ変換し、他の規則を打ち切らない。
`check-ids` が持っていたのは番号の衝突検査ではなく、ファイル名 stem 全体の二重配置検査だった。
この規則を `check-work-items` の依存関係集合へ吸収し、CLI、内部モジュール、単体テスト、mise task、
package script を削除した。番号だけが同じ記録は従来どおり許す。
`workspace/` は文書配置と遅延 snapshot だけを所有し、`check/` への依存を持たない。
work item の二つのディレクトリ、各本文、frontmatter は対象ごとに一度だけ作る。Go source、正準文書、
OpenAPI も snapshot の directory listing と本文 cache を共有する。API 互換性検査は `check/` へ移し、
合否を決めない coverage debt、security test gap、spec diff は照会用ディレクトリへ分けた。
実行入口を持つ TypeScript は 25 個から 10 個になった。集約検査が起動する Bun の検査入口は
19 個から、contract 生成、仕様 view、registry runner の 3 個になった。`tools/package.json` の script は
12 個から、mise が呼ぶ 3 個だけになった。
| 計測 | 変更前 | 変更後 |
| --- | ---: | ---: |
| `mise run time-verify -- check` の直列合計 | 9.90 秒 | 8.60 秒 |
| 同条件の `mise run check` 実時間 | 4.28 秒 | 4.33 秒 |
| 同実行の user + sys CPU | 12.54 秒 | 9.03 秒 |
| 同実行の最大 RSS | 604,028,928 bytes | 583,942,144 bytes |
並列実行の実時間は 0.05 秒増で誤差範囲だった。一方、比較可能な直列合計は 1.30 秒減り、CPU は
28%、最大 RSS は 3% 減った。集約時の重複起動と重複走査を減らす目的は、総仕事量で確認できた。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
