---
status: completed
archived_from: ae0ce5e5e2c87f8b1f0610646319ccb96a14ea98
---

# 拒否の特別扱いをやめ、規範 id の網羅を 1 つの規則、1 つの台帳、1 つの検査に畳む

`mise run spec-diff` は `no normative specification change against main` を返す。規範の差分は無い。
変わったのは網羅を数える仕組みで、規則は 1 つ、台帳は 2 つ (シナリオ 191 件、standards 134 件)、
網羅の検査は `checkNormativeCoverage` 1 つになった。`security-refusal-debt.json`、
`checkRefusalCoverage`、`refusalScenarioIds`、15 語の `REFUSAL_WORDS`、`accounted` による
互いに素の不変条件を削除し、`declaredRefusalTypes` は述語を外して
`errorTypesNamedByScenarios` になった。R1、R2、R4 は残り、判定は変わらない。

全文は `git show ae0ce5e5e2c87f8b1f0610646319ccb96a14ea98:<このファイルのパス>` で読む。
