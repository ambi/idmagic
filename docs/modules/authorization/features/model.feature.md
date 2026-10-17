# Feature: 認可モデルの例

## Rule: REQ-AUTHORIZATION-001 管理者は認可モデルを版として登録でき、整合しないモデルは拒否される

### Example: EX-AUTHORIZATION-001-01 通常経路

- Given `AdminAuthorizationModelManage` を持つ管理者として認証済みである
- When 管理者がリソース型と関係の定義を PutAuthorizationModel へ渡す
- Then テナント内で単調増加する新しい版が作られ、以前の版は書き換わらない
- Then レスポンスは整合トークンを含み、GetAuthorizationModel が新しい版を最新として返す

### Scenario Outline: 条件ごとの結果

- Given `AdminAuthorizationModelManage` を持つ管理者として認証済みである
- When 管理者がリソース型と関係の定義を PutAuthorizationModel へ渡す
- But <condition>
- Then AuthorizationModelInvalidError で拒否し、版を作らない

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-AUTHORIZATION-001-02 | 定義が宣言されていない型または関係を参照する |
  | EX-AUTHORIZATION-001-03 | 書き換え規則が循環する |
  | EX-AUTHORIZATION-001-04 | 型名または関係名がフォーマットに反する |

## Rule: REQ-AUTHORIZATION-010 認可モデルとタプルの更新も判定の呼び出しも管理者に限られる

### Example: EX-AUTHORIZATION-010-01 通常経路

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" が PutAuthorizationModel または WriteRelationTuples を呼ぶ
- Then AccessDeniedError で拒否される
- Then 認可モデルの版は 1 つも作られず、タプルは 1 件も書き込まれない

### Example: EX-AUTHORIZATION-010-02 "alice" が CheckAccess または ListAccessibleResources を呼ぶ

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" が PutAuthorizationModel または WriteRelationTuples を呼ぶ
- But "alice" が CheckAccess または ListAccessibleResources を呼ぶ
- Then AccessDeniedError で拒否される
