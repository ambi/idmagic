---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# `docs/domain/scenarios.feature.md` を作り、Context を跨ぐ保証に置き場所を与える

`docs/domain/scenarios.feature.md` を作り、Context を跨がないと成り立たない保証を 3 件置いた。`REQ-PLATFORM-001` は主体の無効化がログイン・既存セッション・エージェントのトークンという 3 経路を**同時に**閉じることを 1 つの保証として述べ、外部への伝播はこの保証に含まれないこと（内部で閉じ切るのが先）も明示した。`REQ-PLATFORM-002` は削除の予約と復元が到達経路の開閉と対応することを、`REQ-PLATFORM-003` は記録の正の変更と配信行が同じトランザクションでコミットまたはロールバックすることを述べる。断片 6 件は、自 Context が引き金を持てる形へ書き直したもの 4 件と、内容が丸ごと横断で退役させたもの 2 件に分けた。判定は全 21 Context の 451 件の `WHEN` から絞り込んで行い、Sourcing と Seeding は該当しないと判定した。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
