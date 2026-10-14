---
status: completed
archived_from: 59737b9222e4d478a8ad6fef724aeeb4f347809c
---

# idmanagement/user の契約テストへ、未検証の食い違い候補 6 件を加える

`mise run spec-diff` は規範仕様の差分なしを報告した。REQ-IDMANAGEMENT-005 は変えず、PostgreSQL 構成で状態が未設定の User が `status=active` の絞り込みの一覧と件数から漏れていたのを、規範に合わせて直した。あわせて、`idmanagement/user` の共有契約へ 6 件のサブテストを加え、メモリ実装を PostgreSQL 実装に揃えた。削除済みの User は一意性に数えず、保存と読み取りでは値を共有せず、保存し直しても作成時刻と所属テナントを保ち、時刻の列とトークンの時刻をマイクロ秒に切り捨て、トークンのペイロードを共有しない。

全文は `git show 59737b9222e4d478a8ad6fef724aeeb4f347809c:<このファイルのパス>` で読む。
