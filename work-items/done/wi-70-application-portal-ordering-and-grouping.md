---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 利用者ポータルのアプリケーション一覧に並び替えと分類を入れる

利用者ポータルのアプリ一覧に「並び替え」と「分類」を入れた。並び順 (slice 1) は
ApplicationCatalog が tenant_id + user_sub をキーに per-user の手動順を所有し、既定は
name 昇順。手動順を割当済み visible アプリの上に重ね、未割当 id は除外、手動順に無い
アプリは name 昇順で末尾に付ける。分類 (slice 2) は管理者が tenant 単位で
ApplicationCategory を定義し Application に 0..N 個付与する。利用者ポータルはカテゴリ
定義 (position 昇順) でタイルをセクション表示し、未分類は「その他」に集める。所有関係は
ADR-069 が定める。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
