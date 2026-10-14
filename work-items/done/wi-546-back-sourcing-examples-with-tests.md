---
status: completed
archived_from: c85cd7cfbdf250a6aae6e4bd387749e4d3e2d7f5
---

# Sourcing が宣言する具体例 25 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して `no normative specification change` を返した。
Sourcing が台帳に残していた 25 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
既存テストへディレクティブを足すだけで済んだのは `EX-SOURCING-002-01` の 1 件で、他の 24 件は `/Groups` の照会、効果の不在（読み直した資源が拒否の前と同じ、User 数が増えない、メンバーシップが作られない）、保存先の値、有効ロールのいずれかの観測を足した。
`EX-SOURCING-007-03` の照合で、`PERSISTENCE=memory` の構成では拒否した UpdateScimUser と PatchScimUser が User を途中まで書き換える欠陥が分かった。
利用者の判断で本項目で直し、`UpdateUser` と `PatchUser` が保存先の User の複製を書き換える形にした。

全文は `git show c85cd7cfbdf250a6aae6e4bd387749e4d3e2d7f5:<このファイルのパス>` で読む。
