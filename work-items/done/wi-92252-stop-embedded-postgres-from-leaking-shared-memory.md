---
status: completed
archived_from: fffd4da5d583fc549ac8276f07ed0f9cde57ed05
---

# テスト用 embedded-postgres が SysV 共有メモリを漏らさないようにする

`mise run spec-diff` は規範仕様の差分を報告しない。変更はテスト基盤に閉じる。
`testing_postgres.Main` は、SysV 共有メモリを扱えない環境（macOS のサンドボックス）では postgres を起動せずに DB テストをスキップする。
起動のたびに、持ち主のテストバイナリが終了した `idmagic-pgtest-*` を回収する。孤児の postmaster は `pg_ctl stop -m immediate` で止め、postmaster まで死んでいる場合は記録された共有メモリを作成者の pid と接続数 0 を確かめて削除する。
embedded-postgres は起動時に RuntimePath を消して作り直すので、持ち主の pid は `idmagic-pgtest-*/owner.pid` に置き、RuntimePath は `idmagic-pgtest-*/runtime` とした。

全文は `git show fffd4da5d583fc549ac8276f07ed0f9cde57ed05:<このファイルのパス>` で読む。
