---
status: completed
archived_from: 2de4573913eeeb148c260406a1dab725348575db
---

# 制御面のクォータ更新と正規ロケーション切替を監査ログへ記録する

`mise run spec-diff` は、REQ-TENANCY-011、REQ-TENANCY-012、REQ-TENANCY-037 の変更、`TenantEndpointStyleChanged` の追加、`TenantQuotaUpdated` の変更を示した。
System 管理者によるクォータ更新は、操作者、テナント、要求に含まれたリソースの名前（`changedFields`）を載せた `TenantQuotaUpdated` を記録する。
正規ロケーションの切替は、操作者、テナント、切り替える前と後の `endpoint_style` を載せた新しい `TenantEndpointStyleChanged` を記録し、前と同じ値への切替でも記録する。
拒否された要求と、上書きを保存できなかったクォータ更新は記録しない。
監査の `category=tenant` は二つの型も返す。
`check-event-contract` は、本番コードのどこも値を作らないイベント型を落とすようになり、既存の 4 件は `tools/check/event-emission-debt.json` に負債として載る。

全文は `git show 2de4573913eeeb148c260406a1dab725348575db:<このファイルのパス>` で読む。
