---
status: completed
archived_from: e50ccb1f62ec962e4c3e592b991b0b096589a9d2
---

# IdentityManagement が宣言する未検証の拒否に効果まで確かめるテストを与え、台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 17 件が
「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
移ったことと、その 17 件が網羅台帳から消えたことである。
`check-spec` の網羅は「テストが名指しする id」75 件から 92 件へ増え、
`example-coverage-debt.json` は 655 件から 638 件へ、IdentityManagement 分は
84 件から 67 件へ減った。
製品コードの変更は無い。防護が欠けている経路は 1 つも見つからなかった。

全文は `git show e50ccb1f62ec962e4c3e592b991b0b096589a9d2:<このファイルのパス>` で読む。
