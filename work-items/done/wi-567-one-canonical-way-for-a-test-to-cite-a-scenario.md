---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# テストが具体例を被覆したと主張する書き方を 1 つに定め、検査をそれだけに合わせる

`mise run spec-diff` は `no normative specification change against main` を返す。規範も製品も動いていない。
変わったのは、テストが宣言済みの id を被覆したと主張する書き方が 1 つになったことである。
`//spec:covers <id>[, <id>]: <何を固定しているか>` だけが被覆と数えられ、散文で id に言及することは
安全になった。既存の注記 588 行を指示子へ寄せ、被覆と数えられる id の集合は変更の前後で一致している。
あわせて、被覆の指摘が「どのファイルのどの行が名指しているか」を言うようになった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
