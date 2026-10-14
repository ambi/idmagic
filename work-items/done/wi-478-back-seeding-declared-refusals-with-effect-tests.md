---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Seeding が宣言する未検証の拒否 6 件に効果まで確かめるテストを与え、台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 6 件が
「エラーだけ確かめられている」状態から「書き込みの不在と拒否の位置まで確かめられている」
状態へ移ったことと、その 6 件に対照の `EX-SEEDING-008-01` を加えた 7 件が
網羅台帳から消えたことである。
Seeding 分の台帳は 12 件から 5 件へ減った。製品コードの変更は無い。
拒否の位置が宣言より後ろだった経路も、防護が欠けていた経路も見つからなかった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
