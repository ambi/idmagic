---
status: completed
archived_from: 5a4b020554c8d18adcf11dfe588b18c34230ac0d
---

# katex の脆弱性勧告 GHSA-238p-pmpm-9mq7 で失敗する依存監査を解消する

`mise run spec-diff` は規範の差分なしを報告する。
`tools/package.json` の `overrides` で `katex` を 0.18.2 に固定し、`tools/bun.lock` を更新した。
`mise run audit-dependencies` は GHSA-238p-pmpm-9mq7 を報告しなくなった。
文書サイトがブラウザへ配る `mermaid.min.js` には `katex` 0.16 系が同梱されたまま残る。
この残存と受け入れる理由は設計に記録した。

全文は `git show 5a4b020554c8d18adcf11dfe588b18c34230ac0d:<このファイルのパス>` で読む。
