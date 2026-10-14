---
status: completed
archived_from: 86ff5dab4afa3e889fb424bf07dc0cef764167d2
---

# 実装変更に仕様影響の宣言を必須にし、暗黙の仕様を見つける

`mise run spec-diff` は `no normative specification change against main` を返す。
製品の規範仕様は変えず、開発の検査と手順を変えた。
`check-repository` に `spec-impact` を登録し、`mise run check` が基準リビジョンから作業ツリーまでについて、本番コードを変更したコミットの宣言の有無、`impact: modifies` の実在、`impact: conforms` の根拠、差分の申告漏れ、`spec_impact: none` と `Spec-Impact: none` の矛盾、理由の形を検査するようにした。
`affected_spec` の各項目に `impact`（`modifies` または `conforms`、省略時は `modifies`）を追加した。
`spec-diff` は既存の TypeSpec 宣言の変更を `changed TypeSpec declarations` として報告し、操作ごとの転送用の宣言の変更を、所有する操作の変更として数える。
標準の行と `affected_spec` の照合は `spec-impact` へ移し、`spec-diff` は何も失敗させない読み取り専用のタスクになった。
`mise run spec-review-candidates -- <package-directory>` を追加し、仕様 ID を引くテストだけが実行しない本番コードを、レビューの候補として報告する。
`SPECIFICATION_FORMAT.md` §6、`WORK_ITEM_FORMAT.md`、`docs/development/specification-first-workflow.md`、`AGENTS.md` に、仕様と実装詳細の境界、`impact`、トレーラー、書き起こしの手順と観点表を書いた。
導入時点の結果から、21 コンテキストの書き起こしの work item を起票した。
`check`、`check-repository`、`check-boundary-debt-ratchet` が基準リビジョンの引数を受け取れていなかった不具合を、shebang を加えて直した。
仕様のリビジョンの読み取りを `git cat-file --batch` にまとめ、一つのリビジョンあたり数百回だった git の起動を二回にした。

全文は `git show 86ff5dab4afa3e889fb424bf07dc0cef764167d2:<このファイルのパス>` で読む。
