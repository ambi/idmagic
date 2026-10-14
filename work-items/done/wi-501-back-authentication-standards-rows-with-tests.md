---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Authentication が宣言する標準 9 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `docs/domain/authentication/standards.md` の 10 行のうち、名指しを持たなかった 9 行に対する
被覆の状態である。8 行がその行の `Statement` を区別できる入力と観測を持つテストを得て
`tools/check/standards-coverage-debt.json` から消え、台帳は 130 件から 122 件になった。
残る `RFC8176-AMR-VOCABULARY` は、行と `scenarios.feature.md` が `federated` について両立しないため
行を満たすテストが書けない。台帳には残したうえで、理由の欄を「投入時からある」から食い違いと
切り出し先を名指す文へ書き換えた。製品コードは 1 行も変わっていない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
