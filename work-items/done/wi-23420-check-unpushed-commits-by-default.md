---
status: completed
archived_from: ebf294b2e2b16505b5d6f6e3d0087b9d6670e37e
---

# リポジトリ検査の比較基準を、既定でまだ push していないコミットの起点にする

`mise run spec-diff` は規範の差分なしを報告する。
比較基準の既定値を `tools/check/src/runner.ts` の一か所で解決するようにした。
`--base-revision` が渡されなければ、上流ブランチ（`@{upstream}`）があればそれを、なければ `main` を使う。
`check`、`check-repository`、`check-boundary-debt-ratchet`、`check-spec-impact` の 4 タスクは、引数があるときだけ `--base-revision` を渡す。
`main` の上で直接コミットしても、まだ push していないコミットを手元の `mise run verify` が検査するようになった。
CI は比較基準を明示して渡すので、CI の検査範囲は変わらない。

全文は `git show ebf294b2e2b16505b5d6f6e3d0087b9d6670e37e:<このファイルのパス>` で読む。
