---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Authentication が宣言する具体例 71 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff -- 9b574186` は「no normative specification change」を返す。規範は動いていない。
動いたのは対応付けで、Authentication が宣言する具体例のうち本項目が引き取った 71 件から 69 件を
`tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
台帳の Authentication 行は 71 件から 2 件へ減り、残る 2 件はどちらも `blocked_by` と `finding` を持つ。
併せて、既存の e2e fixture 4 つがイベントを記録するようになり、`testing_stack` が
失敗回数・エンドポイント流量制限・成立しないログイン・WS-Federation・セッションの保管先を配るようになった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
