# Feature: リソース上限のシナリオ

## 入力

### Rule: REQ-TENANCY-012 System管理者はテナントのクォータ上限を調整できる

Primary actor: `SystemAdministrator`

#### Example: EX-TENANCY-012-01 通常経路

- Given system_admin ロールを持つ "sysadmin" が認証済みである
- When "sysadmin" が UpdateTenantQuota を呼び出しユーザー上限を 20000 に増やす
- Then 対象テナントの quota.users が 20000 になる

### Rule: REQ-TENANCY-037 クォータの更新は上書きの全体を置き換える

- 要求に含めたリソースの上書きを保存し、含めなかったリソースの上書きを消してデフォルト値へ戻す。
- 現在の使用量を下回る上限も保存する。既存のリソースは残し、新しい作成だけを拒否する。
- **担保手段**：`QuotaRepository.SetQuota`
- **要判断**：負の上限を検証せずに保存する。負の上限はすべての作成を拒否する。負の値を拒否するかを決める。

#### Example: EX-TENANCY-037-01 一部のリソースだけを含む更新

- Given テナントは users を 20000、groups を 5 に上書きしている
- When System 管理者が users だけを 30000 にする更新を保存する
- Then users の上書きは 30000 になり、groups の上書きは消えてデフォルト値の 1000 に戻る

#### Example: EX-TENANCY-037-02 現在の使用量を下回る上限

- Given テナントの groups の使用量は 3 である
- When System 管理者が groups の上限を 2 にする更新を保存する
- Then 更新は保存され、使用量は 3 のままで、次の Group の作成は QuotaExceededError で拒否される

## 結果

### Rule: REQ-TENANCY-036 Hard Quota の実効値は、上書きがあればその値、なければデフォルト値である

| リソース | デフォルト値 |
| --- | --- |
| `users` | 10,000 |
| `groups` | 1,000 |
| `agents` | 100 |
| `applications` | 50 |
| `oauth2_clients` | 100 |
| `active_sessions` | 50,000 |
| `consents` | 10,000 |
| `active_jobs` | 10 |
| `ssf_streams` | 20 |

- 使用量に加算分を足した値が実効値を超える作成を拒否し、使用量を変えない。実効値と等しくなる作成は受け付ける。
- 上書きが 0 のリソースは、すべての作成を拒否する。
- 表にないリソース名の確認は失敗させ、使用量を変えない。
- 使用量の減算は 0 を下回らない。
- **理由**：[リソース上限の設計判断](decisions.md)
- **担保手段**：`TenantQuota.EffectiveLimit`、`QuotaRepository.CheckAndIncrement`、`QuotaRepository.Decrement`

#### Example: EX-TENANCY-036-01 上書きのあるリソース

- Given テナントは users の上限を 1 に上書きしている
- When users の使用量を 1 ずつ二回加算する
- Then 一回目は成功し、二回目は QuotaExceededError で拒否され、使用量は 1 のままである

#### Example: EX-TENANCY-036-02 上書きのないリソース

- Given テナントは上限を上書きしておらず、active_jobs の使用量は 10 である
- When active_jobs の使用量を 1 加算する
- Then デフォルト値の 10 を超えるため QuotaExceededError で拒否される

#### Example: EX-TENANCY-036-03 表にないリソース名

- When リソース名 "widgets" の使用量を 1 加算する
- Then 確認は失敗し、どのリソースの使用量も変わらない

#### Example: EX-TENANCY-036-04 使用量を超える減算

- Given テナントの groups の使用量は 1 である
- When groups の使用量を 5 減算する
- Then 使用量は 0 になる

## 拒否

### Rule: REQ-TENANCY-013 Hard Quota を超過したリソース作成は拒否される

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-013-01 通常経路

- Given 対象テナントの groups 上限が 1000、利用量が 1000 である
- When テナント内管理者が新しい Group を作成しようとする
- Then QuotaExceededError で拒否され作成されない
