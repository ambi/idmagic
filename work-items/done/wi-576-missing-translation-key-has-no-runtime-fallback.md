---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 翻訳キーの欠落に実行時のフォールバックが無い

`mise run spec-diff` は main に対する規範仕様の差分なしを報告した。規範は変えず、`EX-SYSTEM-010-03` と用語集の `FallbackLocale` が述べる振る舞いに実装を揃えた。
`defineDictionary` が定義の時点で、`ja` の値が未定義または空文字列のキーを `en` の同じキーの値で埋める。フックを通らず辞書を直接引く 5 か所にも効かせるため、`useDictionary` ではなく定義の側で解決した。`en` は落ちる先がないので、空文字列も訳として扱う。
フォールバックが翻訳漏れを画面から隠さないよう、`defineDictionary` が埋めたキーを `untranslatedKeys` で取り出し、`dictionaries.test.ts` がすべての `*.i18n.ts` (52 個) でそれが空であることを確かめる。最初の版は `en` の空の値も漏れとして数えたが、`AdminSettingsPage.i18n.ts` の `countSuffix` が英語で単位を付けないため意図して空であることが分かったので、数えないことにした。
`tools/check/example-coverage-debt.json` から `EX-SYSTEM-010-03` を外した。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
