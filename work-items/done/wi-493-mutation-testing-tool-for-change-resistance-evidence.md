---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# 変異テストの道具を評価し、change-resistance の証拠を再現可能な mise タスクにする

`mise run spec-diff main` は規範差分なしを報告する。変わったのは change-resistance の証拠の作り方だけで、
`mise run test-go-mutation -- <パッケージディレクトリ>` が入り、Gremlins v0.6.0 が `[tools]` に pin された。
これまで使い捨てスクリプトでしか作れなかった系統的な変異が、誰でも同じ結果で再現できるコマンドになった。
task が引き受けているのは 2 つの落とし穴で、対象範囲がカレントディレクトリで決まること (リポジトリ root
から呼ぶと embedded-postgres まで巻き込む) と、既定の timeout 係数 3 では全件 TIMED OUT を終了コード 0 で
返すことである。あわせて `specification-first-workflow.md` に「ツールは構文の空間を、手書きの障害モデルは
仕様が気にする意味の空間を覆う」という切り分けと、生き残りを読んで点数化しない理由を書き、
`WORK_ITEM_FORMAT.md` の change-resistance の記述からそこへ繋いだ。ゲートには入れていない。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
