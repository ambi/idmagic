---
status: completed
---

# tools のテストの別プロセスの起動を減らし、表の行の分割を一つにする

`mise run spec-diff` は main に対して規範の差分なしを示す。
2026-10-29 に tools のテストを一本ずつ測ると、約 45 秒のうち `spec-impact` の受け入れテストが約 23 秒、`table-writes` のテストが約 6 秒を占めた。
`spec-impact` の受け入れテストは、40 件ごとに起動していた検査の CLI をやめ、同じプロセスで `checkSpecImpact` を呼ぶ。このファイルは 21.6 秒から 18.0 秒になり、`bun test` 全体は交互に二回ずつ測った平均で約 43.9 秒から約 38.0 秒になった。
残りの時間は、検査が過去のコミットごとに `git show` などを呼ぶ作りと fixture の git 操作によるもので、`spec-impact` の簡素化で扱う。
`checkSpecImpact` の所見を落とす故障を入れると、受け入れテストの 18 件が失敗する。
Markdown の表の行の分割は `markdown-table.ts` の `tableRowCells` 一つにし、`specification-doc`、`schema-tables`、`boundary-fitness` から使う。両端の区切りを省いた行も読み、`\|` を区切りにしない。末尾の区切りのない表の行は文書にないので、判定は変わらない。
重複を調べた結果、Go のソースを読む解析は既に共有されており、git を呼ぶ補助は数行ずつで統合すると依存が増えるので統合しない。`table-writes` の起動の統合も、効果が約 2.5 秒でテストが読みにくくなるので行わない。
