# Feature: 関係タプルの例

## Rule: REQ-AUTHORIZATION-002 関係タプルの書き込みは登録済みモデルに適合するものだけを一括で適用する

### Background:

- Given テナントに認可モデルが登録済みである

### Example: EX-AUTHORIZATION-002-01 通常経路

- When 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
- Then 差分は 1 トランザクションで適用され、既に存在する組の再追加は冪等に扱われる
- Then レスポンスは書き込み後の整合トークンを返し、以後の判定へ渡せる

### Scenario Outline: 条件ごとの結果

- When 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-AUTHORIZATION-002-02 | モデルが宣言していない型・関係を含む | RelationTupleInvalidError で拒否し、1 件も適用しない |
  | EX-AUTHORIZATION-002-03 | `direct` 規則が許していない主体型またはワイルドカードを含む | RelationTupleInvalidError で拒否し、1 件も適用しない |
  | EX-AUTHORIZATION-002-04 | 同じ組が追加と削除の双方に現れる | RelationTupleInvalidError で拒否し、1 件も適用しない |
  | EX-AUTHORIZATION-002-05 | テナントに認可モデルが未登録である | AuthorizationModelNotFoundError で拒否する |

## Rule: REQ-AUTHORIZATION-008 オブジェクトの削除はその両側の関係タプルを取り除く

### Example: EX-AUTHORIZATION-008-01 通常経路

- Given あるオブジェクトが、リソース側と主体側の双方でタプルに現れている
- When 管理者がそのオブジェクトを削除対象として WriteRelationTuples へ渡す
- Then そのオブジェクトを参照するタプルは、リソース側・主体側のいずれも残らない
- Then 削除に依存していた間接的な関係は以後成立しなくなり、整合トークンが進む
