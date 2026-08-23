# Provisioning Internals

## Protocol-agnostic core with protocol feature slices

接続の共通情報、`DeprovisionPolicy`、`AttributeMappingRule`、`ProvisioningDelivery` と `RemoteResourceLink` によるキューイング、再試行、隔離、順序制御、再同期は、いずれもプロトコルに依存しない。プロトコル固有なのは、通信クライアントと、認証方式、能力の探索、デフォルト属性スキーマなど接続設定の一部だけである。このため、Context のルート (`domain`、`ports`、`usecase`、`handlers_http`) にプロトコル非依存の中核を置き、プロトコルごとに `ProvisioningTargetClient` ポートを実装する薄い機能単位を設ける。現在は `client_scim` があり、将来は中核を変えずに `entraid` と `googledir` を同じ階層へ追加する。この Context ではドメインの大部分がプロトコルに依存しないため、リポジトリで一般的な「厚い機能単位、薄い共通ルート」とは逆に、中核を厚く保つ。

この Context の `client_scim` 機能と `Sourcing` の `scim` 機能の間には、共有の SCIM 通信中核を置かない。内向き側には、受信した SCIM リクエストを自身のデータに対して評価するためのフィルター構文解析、評価器、固定レスポンスの構造体が必要である。一方、外向き側には、フィルター文字列の組み立てと、対応付けに基づく広い属性集合 (`externalId`、Enterprise 拡張) の直列化が必要になる。重なるのは Discovery の構造体と RFC が定めるスキーマ URN 程度なので、実際の重複が増えるまでは共通化しない。

## Same-transaction delivery capture

配信処理は、共通の `outbox` テーブルを外部の Kafka、Pub/Sub、ログへ転送するイベントリレーを利用しない。そのリレーは `IdManagement` や `Application` の書き込みトランザクション内で `ProvisioningDelivery` を作成できないためである。代わりに `Provisioning` は配信捕捉用の公開ポートを提供し、`IdManagement` のユーザー変更処理と `Application` の割り当て変更処理が、それぞれ自身の PostgreSQL トランザクション内でこのポートを呼び出す。ポートは一致する有効な接続ごとに `pending` の `ProvisioningDelivery` 行を 1 件挿入する。発火元の変更と配信行は同時にコミットまたはロールバックされるため、`ProvisioningDelivery` がこの Context 専用の Transactional Outbox となる。配信の冪等性には `(tenant, connection, source_type, source_id, source_version)` をキーとして使う。
