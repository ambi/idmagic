---
status: completed
archived_from: 92e556a5d23d6cfc329c4005fe9c71f351945f13
---

# 人が読む文書を日本語で書く規則を、いま外れている文書へ適用する

`WORK_ITEM_FORMAT.md`、`SPECIFICATION_FORMAT.md`、`docs/development/specification-first-workflow.md` の人が読む散文を日本語で書き直し、識別子、パス、機械検査が読む名前は原表記を保った。
Context の用語集 21 件を `用語 | 定義 | 別名`、索引表 11 か所を `文書 | 内容` に揃え、採用した見出しを `SPECIFICATION_FORMAT.md` に明記した。
開発方法論の参照は、Extreme Programming を Kent Beck の Tidy First?、Screaming Architecture を Jimmy Bogard の Vertical Slice Architecture に置き換えた。Tidy First? は、小さな整理、整理を先に行うリファクタリング、その反復によるインクリメンタルな設計として記述した。
`mise run spec-diff` は main に対して規範の変更なしと報告した。

全文は `git show 92e556a5d23d6cfc329c4005fe9c71f351945f13:<このファイルのパス>` で読む。
