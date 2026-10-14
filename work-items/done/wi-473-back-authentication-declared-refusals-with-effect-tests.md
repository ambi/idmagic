---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# Authentication が宣言する未検証の拒否に効果まで確かめるテストを与え、台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。
規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 16 件が
「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
移ったことと、その 16 件が網羅台帳から消えたことである。
`check-spec` の網羅は「テストが名指しする id」59 件から 75 件へ増え、
`example-coverage-debt.json` は 671 件から 655 件へ、Authentication 分は 89 件から 73 件へ減った。
実装の変更は 1 か所だけで、`session/db_memory.SessionStore` が
`Find` / `FindOwned` / `Revoke` / `Touch` / `ListBySub` / `DeleteAllForSub` のいずれでも
テナントを見ていなかったのを、PostgreSQL 実装と同じくリクエストのテナントで絞るようにした。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
