# wi-71372-rewrite-tenancy-specifications-in-the-requirement-format

Tenancy の機能仕様を、外部から観測できる結果を述べる要件だけで書き直した。
振る舞いは変わらない。
これまで例の付録にしか書かれていなかった要件にも本文を書き、利用者が依存してよい応答、状態、イベントを要件から読めるようにした。

- System 管理者によるテナントの設定の更新が、テナント管理者の更新と同じ規則で検証し、`TenantUpdated` を記録することを、要件として約束した（[`REQ-TENANCY-044`](../../domain/tenancy/settings/README.md)）。
- System 管理者による realm を指定したテナントの取得が、上限と使用量を添えて返し、存在しない realm を 404 `tenant_not_found` で拒否することを、要件として約束した（[`REQ-TENANCY-045`](../../domain/tenancy/lifecycle/README.md)）。
- テナントのライフサイクルに状態遷移表（マトリクス形式）を加えた。無効化、再開、正規ロケーションへの要求の結果を、状態ごとに読める（[TenantLifecycle](../../domain/tenancy/lifecycle/README.md)）。
