---
status: completed
archived_from: 8a21bf1bcb14907ddf36791c9ed902506cb6b932
---

# メモリと PostgreSQL の永続化アダプターを、1 つの契約テストへ通す

`mise run spec-diff` は `main` に対する規範仕様の差分がないと報告した。
比較可能な 34 Context すべてで、メモリ実装と PostgreSQL 実装が同じ `testing_contract` 本体を実行するようにした。
共有契約が検出した `preferred_username` の一意性、WebAuthn セッションのテナント分離、trust bundle issuer の一意性という 3 件の局所的不整合を、既存の PostgreSQL 制約へ合わせて修正した。

全文は `git show 8a21bf1bcb14907ddf36791c9ed902506cb6b932:<このファイルのパス>` で読む。
