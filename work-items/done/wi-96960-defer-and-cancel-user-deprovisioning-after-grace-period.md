---
status: completed
archived_from: 8fde8e6d80edf2c191cea8b3913e305b829be78d
---

# 猶予期間を持つ User 削除の下流配信を延期し、再割り当て時に取り消す

`mise run spec-diff` は main に対する規範仕様の差分なしを報告した。`REQ-PROVISIONING-006` は宣言済みで、この作業は実装をその宣言へ合わせた。
`on_delete=delete` かつ `grace_period_days` が 1 以上の接続では、User の削除から配信を作らず `ScheduledDeprovision` を保存する。ディスパッチャーが期限に達した予約を delete の配信へ変え、worker が下流へ DELETE を送る。猶予期間中の同じ Application への割り当ては、その接続の古い予約だけを取り消す。
予約は `provisioning_scheduled_deprovisions` に保存し、実体化は予約の条件付き遷移と配信の挿入を一つの SQL 文で行う。内部設計、データベース設計、被覆負債の一覧（`EX-PROVISIONING-006-01`、`-02` を除去）、リリースノートを更新した。

全文は `git show 8fde8e6d80edf2c191cea8b3913e305b829be78d:<このファイルのパス>` で読む。
