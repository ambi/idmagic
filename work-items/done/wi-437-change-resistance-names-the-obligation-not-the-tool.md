---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 変更耐性の証拠を、道具の名前ではなく義務の記述で定める

`mise run spec-diff` は `no normative specification change against main` である。意味上の差は 2 つある。1 つは、[[wi-411-evaluate-go-mutation-testing-pilot]] が固定した Gremlins `0.6.0` と `test-go-mutation-package` を外し、`mise` の固定バージョンが 1 個減ったことである。もう 1 つは、変更耐性の証拠契約が**特定のコマンド名ではなく義務**を述べる形になったことで、`docs/development/specification-first-workflow.md` の梯子と `WORK_ITEM_FORMAT.md` の両方から `mise run test-go-mutation-package -- <package> <git-ref>` という repository 固有のタスク名が消えた。要求そのものは弱めていない（下の Unit RED を参照）。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
