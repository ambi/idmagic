---
status: completed
archived_from: aa5b77f4619fa355a8e10f8a2ccf5be8e98eefb2
---

# 子が 1 つしかない `docs/operations/runbooks/` を 1 階層畳む

`docs/operations/runbooks/` の 3 ファイルを `docs/operations/` 直下へ移し、子が 1 つしかない階層を畳んだ。DOCUMENTATION_GUIDE §4 の構成図と §11.4、SPECIFICATION_FORMAT.md §1 の構成図を追随させ、§11.4 には「runbook のためだけのサブディレクトリを最初から作らない。件数が増えてから与える。索引が要るなら `operations/README.md`」という判断基準を書いた。`AGENTS.md`、`README.md`、`docs/README.md`、`infra/backup/` のスクリプトと README、進行中の wi-290 と wi-400 の参照を張り替えた。

全文は `git show aa5b77f4619fa355a8e10f8a2ccf5be8e98eefb2:<このファイルのパス>` で読む。
