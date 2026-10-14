---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 明示選択した表示言語が `ui_locales` ヒントに負ける

`mise run spec-diff` は main に対する規範仕様の差分なしを報告した。規範は動かさず、`EX-SYSTEM-008-02` と用語集の `DisplayLanguage` が述べる優先順位に実装を揃えた。
`resolveLocale` の解決順を「保存済みの明示選択 > `ui_locales` ヒント > ブラウザー言語 > 起動時の既定」に変えた。`ui_locales` は OP が従わなくてもよい RP 側の推定であり、利用者が OP の画面で自分で選んだ言語より弱いと判断した。保存済み設定へ書くのは言語切り替え UI だけなので、推定値が RP の指定を覆すことはない。
`EX-SYSTEM-008-01` のテストは前提に反する保存済み設定を与えていたので外した。
`tools/check/example-coverage-debt.json` から `EX-SYSTEM-008-02` を外した。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
