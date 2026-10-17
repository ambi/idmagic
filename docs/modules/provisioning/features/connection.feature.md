# Feature: 接続の管理の例

## Rule: REQ-PROVISIONING-001 管理 API クライアントは Provisioning スコープの範囲でだけ接続とプロビジョニングタスクを操作できる

### Example: EX-PROVISIONING-001-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがアプリケーションの接続、テナントの接続、またはプロビジョニングタスクの操作を要求する
- Then `provisioning:read` スコープはアプリケーションの接続とテナントの接続一覧の参照だけを許可する
- Then `provisioning:write` スコープは接続の変更とプロビジョニングタスクの操作の実行だけを許可する

### Scenario Outline: 条件ごとの結果

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがアプリケーションの接続、テナントの接続、またはプロビジョニングタスクの操作を要求する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-PROVISIONING-001-02 | `provisioning:read` だけで接続の変更またはプロビジョニングタスクの操作を要求する | 操作は 403 と `insufficient_scope` で拒否される |
  | EX-PROVISIONING-001-03 | トークンのテナントとリクエスト先のテナントが一致しない | 操作は 401 の `InvalidAccessTokenError` で拒否される |
  | EX-PROVISIONING-001-04 | `admin` のロールを持たない利用者が変更の操作を要求する | 操作は 403 の AccessDeniedError（`access_denied`）で拒否され、接続とプロビジョニングタスクは変わらない |

## Rule: REQ-PROVISIONING-002 管理者は接続を登録し、接続テストで下流の対応機能を取得できる

### Example: EX-PROVISIONING-002-01 通常経路

- Given Application "app-1" は存在し ProvisioningConnection を持たない
- When 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
- Then ProvisioningConnectionRegistered が発行される
- When 管理者が TestProvisioningConnection を実行する
- Then 下流 /ServiceProviderConfig への到達性が確認され capabilities がキャッシュされる

### Scenario Outline: 条件ごとの結果

- Given Application "app-1" は存在し ProvisioningConnection を持たない
- When 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-PROVISIONING-002-02 | base_url が https でない、または内部/リンクローカル IP を指す | InvalidRequestError が返り ProvisioningConnection は作成されない |
  | EX-PROVISIONING-002-03 | Application "app-1" に既に ProvisioningConnection が存在する | 409 と `provisioning_conflict` で拒否される |

## Rule: REQ-PROVISIONING-012 管理者は On-Demand Provision で 1 人のユーザーを試験的にプロビジョニングできる

### Example: EX-PROVISIONING-012-01 通常経路

- Given User "ユーザー-1" は connection の scope 内である
- When 管理者が ProvisionOnDemand を実行する
- Then `ProvisioningTask`（`status=pending`）が直ちに作成される

### Example: EX-PROVISIONING-012-02 指定した subject が scope 外である (scope=assigned_only で未割り当て)

- Given User "ユーザー-1" は connection の scope 内である
- When 管理者が ProvisionOnDemand を実行する
- But 指定した subject が scope 外である (scope=assigned_only で未割り当て)
- Then 409 と `provisioning_conflict` で拒否される

## Rule: REQ-PROVISIONING-013 管理者はフル同期で適用範囲の全対象を収束できる

### Example: EX-PROVISIONING-013-01 通常経路

- Given connection の scope 内に複数 subject が存在し一部は下流と乖離している
- When 管理者が StartFullResync を実行する
- Then scope 内の全 subject に対して ProvisioningTask が作成される
- Then すべてのプロビジョニングタスクが収束すると FullResyncCompleted が発行される

## Rule: REQ-PROVISIONING-014 資格情報のローテーションは監査でき、`rotated_at` が更新される

### Example: EX-PROVISIONING-014-01 通常経路

- Given connection は bearer_token 認証で稼働している
- When 管理者が UpdateProvisioningConnection に新しい credential を渡す
- Then ProvisioningCredentialRotated が発行され credential.rotated_at が更新される
- Then 以後のプロビジョニングタスクは新しい credential で下流へ認証する

## Rule: REQ-PROVISIONING-015 他テナントの接続とプロビジョニングタスクはテナント境界を越えない

### Example: EX-PROVISIONING-015-01 通常経路

- Given テナント "tenant-a" の ProvisioningConnection と ProvisioningTask が存在する
- When テナント "tenant-b" の管理者が GetProvisioningConnection または GetProvisioningTask を同じ id/task_id で呼ぶ
- Then ProvisioningConnectionNotFoundError または ProvisioningTaskNotFoundError が返る
