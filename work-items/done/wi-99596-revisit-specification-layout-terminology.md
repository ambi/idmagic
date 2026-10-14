---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 仕様文書の木の用語とファイルの名前を見直す

`mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範の差分はない。
「機能ノード」を廃止し、仕様のディレクトリとコードのディレクトリからなる一つの機能の単位を「機能スライス」と定義した。規約の文書、検査と生成器の識別子と診断、work item のスキーマ、コメントをこの語へそろえた。
例の付録 84 個を `examples.feature.md` から `acceptance.feature.md` へ移し、検査、生成器、`generate-spec-examples`、テストの準備データ、文書のリンク、`spec-change` スキルの言及をそろえた。
`docs/domain/scenarios.feature.md` の前書きを、Context ごとの要件は機能仕様で宣言するという現在の規則に合わせた。
完了した work item の `affected_spec` は旧い付録を参照していないので、旧パスの対応表には項目を加えなかった。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
