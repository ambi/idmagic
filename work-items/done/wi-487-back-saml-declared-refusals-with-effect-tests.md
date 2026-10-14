---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SAML が宣言する未検証の拒否 2 件に効果まで確かめるテストを与え、台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 3 件が
「応答だけ確かめられている」状態から「発行の不在まで確かめられている」状態へ
移ったことと、その 3 件が網羅台帳から消えたことである。
SAML 分の台帳は 21 件から 18 件へ減った。製品コードの変更は無い。
防護が欠けている経路は 1 つも見つからなかった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
