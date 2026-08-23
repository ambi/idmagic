# Provisioning Internals

## Same-transaction delivery capture

`Provisioning` は配信捕捉用の公開ポートを提供し、`IdManagement` のユーザー変更処理と `Application` の割り当て変更処理が、それぞれ自身の PostgreSQL トランザクション内でこのポートを呼び出す。ポートは一致する有効な接続ごとに `pending` の `ProvisioningDelivery` 行を 1 件挿入する。発火元の変更と配信行は同時にコミットまたはロールバックされるため、`ProvisioningDelivery` がこの Context 専用の Transactional Outbox となる。配信の冪等性には `(tenant, connection, source_type, source_id, source_version)` をキーとして使う。
