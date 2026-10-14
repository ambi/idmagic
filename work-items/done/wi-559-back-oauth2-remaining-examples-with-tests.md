---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# OAuth2 が宣言する残り 99 件の具体例にテストを対応付け、被覆台帳から外す

`mise run spec-diff -- 00a1aa20` は「no normative specification change」を返す。規範は動いていない。
動いたのは対応付けで、OAuth2 が宣言する具体例のうち本項目が引き取った 93 件から 89 件を
`tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
台帳の OAuth2 行は 96 件から 7 件へ減り、残る 7 件はすべて `blocked_by` と `finding` を持つ。
併せて、`testing_stack` がイベントの記録とブラウザー経由の認可を配るようになった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
