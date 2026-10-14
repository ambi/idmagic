---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 監査とジョブのテナント横断 API および画面をシステムコンソールへ移す

`mise run spec-diff` が示す規範上の差分は、追加が `REQ-AUDIT-007`、`REQ-JOBS-015`、`REQ-SYSTEM-020` の 3 つ、変更が `REQ-JOBS-012` と `REQ-JOBS-013` の 2 つ、そして TypeSpec 宣言の追加が `ListSystemAuditEvents`、`ExportSystemAuditEvents`、`GetSystemAuditEvent`、`ListSystemJobs`、`GetSystemJob`、`CancelSystemJob` の 6 つである。
意味の差は、テナント境界を越える範囲を利用者入力ではなく経路が決めるようになったことである。
`REQ-AUDIT-007` と `REQ-JOBS-015` は制御面主体が専用のシステム経路で全テナントを検索・参照・エクスポート・取り消せることを新たに所有し、`REQ-SYSTEM-020` は横断の入口をシステムコンソールだけに置くことを所有する。
`REQ-JOBS-012` と `REQ-JOBS-013` からは制御面の成功経路が抜け、代わりにテナント管理経路が横断を求める入力を添えられても要求先テナントへ閉じることを述べる例に変わった。
`AdminJobQuery` から `all_tenants` を削除したため、テナント管理 API には範囲を動かす入力がもう存在しない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
