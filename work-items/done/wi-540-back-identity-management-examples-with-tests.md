---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# IdManagement が宣言する具体例 66 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff -- 2a18c26f` は「no normative specification change」を返す。規範は動いていない。
動いたのは対応付けで、IdManagement が宣言する具体例のうち本項目が引き取った 66 件から 65 件を
`tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
台帳の IdManagement 行は 66 件から 1 件へ減り、残る 1 件は `blocked_by` と `finding` を持つ。
**製品コードは 1 行も変えていない。** 変わったのはテストと、テストが使う fixture の配線である。
併せて `idmRefusalFixture` が同意・パスワード履歴・テナントごとの OAuth2 クライアントを配るようになり、
`runExport` がエクスポートの発行を記録するようになった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
