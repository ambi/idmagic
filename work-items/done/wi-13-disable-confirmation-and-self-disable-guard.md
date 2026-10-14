---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# ユーザ無効化に確認ダイアログと自爆防止を入れる (削除と対称にする)

削除側 (wi-8) と対称になるよう、無効化 (disable) に軽い確認ダイアログと
admin の自爆防止を入れた。enable (再有効化) は誤操作リスクが低いため
確認なしの即時実行を維持した (片側非対称)。新規 model / state / event /
permission / HTTP endpoint は追加せず、既存 `SetUserDisabled` use case と
`hasPrivilegedRole` ヘルパを流用した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
