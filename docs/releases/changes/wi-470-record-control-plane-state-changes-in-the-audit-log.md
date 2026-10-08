# wi-470-record-control-plane-state-changes-in-the-audit-log

System 管理者によるテナントのクォータ更新と正規ロケーションの切替が、監査ログに記録されるようになった（[`REQ-TENANCY-011`](../../domain/tenancy/resolution/README.md)、[`REQ-TENANCY-012`](../../domain/tenancy/quota/README.md)）。

- クォータを更新すると `TenantQuotaUpdated` が記録される。これまでは仕様にイベントが宣言されていたものの、一度も記録されていなかった。新しく `changedFields` に要求に含まれたリソースの名前を載せる。上限の値は記録しない。
- `endpoint_style` を切り替えると、新しいイベント `TenantEndpointStyleChanged` が記録される。操作者、テナント、切り替える前と後の `endpoint_style` を載せ、前と同じ値への切替でも記録する。
- 拒否された要求と、クォータの保存に失敗した要求は記録しない。
- 監査イベントの検索の `category=tenant` は、この二つのイベントも返す。
