---
status: completed
archived_from: 0eca043caf08f52509ec64f9de2bb129bb6c951b
---

# 費用対効果の低いツールと文書の重複を削り、保守する量を減らす

`mise run spec-diff` は `no normative specification change against main` を返した。
製品の振る舞い、規範要件と TypeSpec は変えていない。
`report-change-coupling`、`spec-route`、`time-lint-go`、`check-go-internal-visibility` と専用の 15 ファイル（1,100 行）、タスク定義 44 行を削除した。
検査 README の過去の棚卸し表と、ローカル開発、CONTRIBUTING、設計更新スキル、ツール README の規則の再掲を削り、所有元への参照へ整理した。
registry の検査、CI のゲート、共有処理、外部依存、製品コードと製品テストは維持した。

全文は `git show 0eca043caf08f52509ec64f9de2bb129bb6c951b:<このファイルのパス>` で読む。
