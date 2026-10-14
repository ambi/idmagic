---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# 既存コードを変える前に特性化テストで現在の振る舞いを固定し、宣言のない振る舞いの変化を検出する

`mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範仕様の差分はない。
`mise run check-spec-impact`（`mise run check` にも含む）は、`Spec-Impact: none` のコミット、宣言のないコミット、`spec_impact: none` の work item の作業範囲が、基準に存在した `TestCharacterize*` 関数の本文を変えたか消したときに失敗するようになった。新しく足した特性化テストと、`none` の作業範囲の中で足した特性化テストの変更は通す。
仕様先行の開発ワークフローに「特性化テスト」の節を加え、手順、名前、置き場所、コミットの順序、書いた後の分類を定め、既存コードからの書き起こしの手順、検査の出力の表、仕様影響の検査の表から参照した。
`docs/development/testing.md` と `implement-work-item` スキルに、既存の Go コードを変える前に特性化テストを書く工程を加えた。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
