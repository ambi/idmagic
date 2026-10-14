---
status: completed
archived_from: 99ca6394cdc8425a66bf333d48847154ee88d4c6
---

# Authorization が宣言する具体例 31 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
シナリオも契約も変えていない。
変わったのは、`docs/domain/authorization/scenarios.feature.md` が宣言する具体例 31 件のそれぞれを、その id を名指しして何を固定しているかを述べるテストが支えるようになったことである。
`tools/check/example-coverage-debt.json` の `untested` は 31 件減り、Authorization の行は残っていない。
ディレクティブだけで済んだのは 13 件で、10 件は既存テストが具体例の `Then` の一部を見ていなかったため観測を足し、8 件はテストが存在しなかったため書いた。
実装が具体例と食い違う件は 1 件も出なかったので、切り出した欠陥はない。

全文は `git show 99ca6394cdc8425a66bf333d48847154ee88d4c6:<このファイルのパス>` で読む。
