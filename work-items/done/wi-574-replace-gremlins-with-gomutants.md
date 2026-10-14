---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Go の変異テストを gomutants へ移行し、判定を検算できるようにする

`mise run spec-diff` は main に対する規範仕様差分がないと報告した。
change-resistance evidence の runner を Gremlins 0.6.0 から gomutants 0.6.1 へ変更し、既定の 5 operator、worker 数 2、cache off を固定した。
通常実行は JSON report を一時領域へ出し、明示した cache file による反復と stable mutant ID 1 件の再実行を `mise` task から利用できる。
固定 canary は lived と not viable を初回と cache 再利用後の両方で検算する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
