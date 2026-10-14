---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# 仕様を、システム、コンテキスト、機能の木に組織し、規則一件の書式を定める

`mise run spec-diff` は再編の後も「no normative specification change against main」を報告し、規範要素の追加、削除、変更はない。
仕様をシステム、コンテキスト、機能の三階層の木に置く規則、機能ノードとコードの機能スライスの対応、継承の規則、機能ノードの中の節の並び、規則一件の書式を `SPECIFICATION_FORMAT.md` に定め、`DOCUMENTATION_GUIDE.md` と開発ワークフローから参照した。
`documentNames` を配置の唯一の判定として導入し、`documentKind`、文書検査、ファイル集合の検査、作業ツリーの文書探索、文書サイト、`spec-route`、`brief`、セキュリティ統制の検査、work item のスキーマが機能ノードの文書を一次情報として扱うようにした。
`spec-diff` は規則の本文（規則文、欄、表）を規則の事実に含め、状態機械をコンテキストで同定する。
`check-spec` に `specification-rules` の検査を加え、担保手段のシンボル、上位の規則のリンク、曖昧な語、機能ノードの節の並び、機能スライスと機能ノードの対応（`tools/check/feature-node-debt.json` を基準とする）を確かめる。
authentication の 37 件の規則のうち 33 件を 10 の機能ノードへ ID を変えずに移し、`states.md`、`decisions.md`、`internals.md` を機能ごとに分けた。
移した規則を指す work item と文書の参照を移動先へ書き換えた。
`glossary.md` は既存の分割の規則に従いコンテキストのルートに残し、計画の 6 から外した。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
