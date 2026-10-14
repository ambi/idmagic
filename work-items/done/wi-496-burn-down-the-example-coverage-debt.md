---
status: completed
archived_from: e92e4db2bc105c6bc8f1e794732567bec287485a
---

# 具体例の被覆負債 614 件を Context 単位で消化し、`example-coverage-debt.json` を空にする

`mise run spec-diff` は規範仕様の差分を報告しない（`no normative specification change against main`）。
変更はリポジトリの検査に閉じる。20 件の子 work item と、それらが切り出した欠陥 work item の完了で、
具体例の被覆負債は 614 件から 0 件になった。空になった `tools/check/example-coverage-debt.json` を削除し、
`checkNormativeCoverage` は宣言した id をどのテストも名指さなければ落とす 1 規則だけを持つ。
台帳に仕えていた `coverage-debt-ratchet`（検査、`mise` タスク、CI の手順、runner の `--base-revision`）、
`report-coverage-debt`、`blocked_by` と `finding` の検査、Traceability ページの負債表示、
名指しの場所（`CitationSite`）も合わせて削除した。`SPECIFICATION_FORMAT.md` から免除の一覧の規則を外した。

全文は `git show e92e4db2bc105c6bc8f1e794732567bec287485a:<このファイルのパス>` で読む。
