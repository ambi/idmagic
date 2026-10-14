---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# Tenancy が宣言する未検証の拒否 6 件に効果まで確かめるテストを与え、台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 11 件が
「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
移ったことと、その 11 件が網羅台帳から消えたことである。
Tenancy 分の台帳は 51 件から 39 件へ減った。製品コードの変更は無い。
防護が欠けている経路は 1 つも見つからなかった。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
