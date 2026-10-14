---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 標準の被覆負債 134 件を消化し、`standards-coverage-debt.json` を空にする

`mise run spec-diff` は `no normative specification change against main` を返す。標準の行も製品の
振る舞いも変わっていない。変わったのは、標準の被覆に免除の仕組みが無くなったことである。
`tools/check/standards-coverage-debt.json` を削除した。投入時点の 134 件は 0 件になっており、
内訳は本項目が T002 で消化した `sharedsignals` の 4 件と、16 件の子 work item が消化した 130 件である。
宣言した標準の行は 157 件あり、そのすべてが id を名指すテストを持つ。
**消えたのは一覧ではなく免除である。** 台帳を消しただけなら、同じ名前のファイルを作り直せば元へ戻る。
`checkNormativeCoverage` の台帳引数は省略可能になり、標準側はそれを渡さない。渡さない呼び出しでは、
行が未消化であることを報告する規則だけが残り、台帳そのものを検査する規則（重複、id 順、理由、
テストが付いた id の残留）は動かない。検査する台帳が無いからである。
**台帳引数は 2 つの省略可能な値ではなく 1 つの組にした。** `debt` と `debtPath` を別々に省略可能に
すると、`debt` だけを渡して `debtPath` を書き忘れた呼び出しが型検査を通り、そのとき台帳側の 4 つの
規則が黙って消える。残る `example-coverage-debt.json` は 614 件を抱えているので、その 4 つが黙って
外れる形は作れない。`ledger?: { entries, path }` にすれば、片方だけを渡すことがそもそも書けない。
**`COVERAGE_DEBT_PATHS` からも標準の台帳を外した。** ratchet は「基準 revision に無い id の追加を
拒否する」検査であり、守る対象が無くなった。標準の行は、名指すテストを持つか `check-spec` に落ちるか
のどちらかなので、流入を測る基準そのものが要らない。
**検査自身が記録の嘘を止めた。** `initial_context` が削除した台帳を指したままだったので
`check-work-items` が落ちた。読んだファイルの一覧は、消えたファイルを指し続けられない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
