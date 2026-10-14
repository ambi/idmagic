---
status: cancelled
archived_from: ae0ce5e5e2c87f8b1f0610646319ccb96a14ea98
---

# 拒否の許可リスト 131 件を消化し、`security-refusal-debt.json` を空にする

取り消す。本項目が引き受けた仕事は 2 つに分かれ、どちらも別の記録が持っている。
**保留した判断は下された。** T001 は進める単位、注記の書き方の型、`REQ-SYSTEM-*` の置き場所の 3 つを
着手時に決めるとして未定のまま残していた。1 つ目はコンテキスト単位に決まり、その形で
[[wi-472-back-oauth2-declared-refusals-with-effect-tests]] から
[[wi-489-back-api-tokens-declared-refusals-with-effect-tests]] までの 18 件が立っている。
18 件は台帳の 102 件を過不足なく 1 回ずつ持ち、拒否ごとに「何を読み直せば効果を確かめたことになるか」を
requirement 単位で書いている。3 つ目は
[[wi-488-resolve-system-entries-on-the-refusal-ledger]] が引き取り、`REQ-SYSTEM-001` は本物の
フェイルクローズ、`REQ-SYSTEM-015` は条件節の「できない」に当たった誤検出、と結論している。
2 つ目の「注記の書き方の型」は不要になった。18 件は注記だけの解消を認めないので、型ではなく
条件として書いてある。
**tooling 側は済んでいる。** T007 は「131 件が 0 になった時点で `security-refusal-debt.json` と
`checkRefusalCoverage` の `allowed` 引数を落とす」としていたが、
[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が債務を残したまま先に落とした。
102 件は `example-coverage-debt.json` へ理由付きで統合され、`checkRefusalCoverage`、
`refusalScenarioIds`、`REFUSAL_WORDS` は削除されている。順序が逆になったのは、測ってみると
拒否専用の検査と台帳が支えていたのが id の振り分けだけで、債務の消化を待つ理由が無かったためである。
本項目の Risk Notes — 検証の付いていない名指しは「未検証が見えている」状態を
「検証済みに見える」状態へ変え、R3 は二度と検出しない — は取り消しても失われない。
18 件がそれぞれ禁止事項として引き継ぎ、`docs/development/specification-first-workflow.md` の
該当節も wi-490 で拒否に限らない形へ広げた。
未着手のため、実装も台帳の変更も行っていない。

全文は `git show ae0ce5e5e2c87f8b1f0610646319ccb96a14ea98:<このファイルのパス>` で読む。
