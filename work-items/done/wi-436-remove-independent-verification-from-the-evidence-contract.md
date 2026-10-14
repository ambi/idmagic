---
status: completed
archived_from: d05ccbac5bcfa4c724a39f78083cfb350bab1584
---

# 独立検証を証拠契約から外す

`mise run spec-diff` は `no normative specification change against main` を返す。製品の振る舞いも公開契約も動いていない。差分は証拠契約とその機械検査に閉じる。`risk: medium` 以上の完了記録に要求されるのは change-resistance と `spec-diff` を読んだ要約だけになり、`reversibility: irreversible` が追加要求を持つ条件節は消えた。`independent_verification` は任意項目として残り、完了済みの 150 件あまりが持つ記載はそのまま検証を通る。

全文は `git show d05ccbac5bcfa4c724a39f78083cfb350bab1584:<このファイルのパス>` で読む。
