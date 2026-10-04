# Feature: 関係の判定の例

## Rule: REQ-AUTHORIZATION-003 判定は継承・グループ・親子関係をたどって関係の成否を決める

### Example: EX-AUTHORIZATION-003-01 通常経路

- Given 認可モデルが `computed_userset` と `tuple_to_userset` を含む関係を宣言している
- And グループの成員、親フォルダーの閲覧者、直接の編集者のタプルが登録されている
- When 呼び出し元が主体とリソースと関係を CheckAccess へ渡す
- Then 結果は許可・不許可と、たどった関係名だけの経路を返す
- Then 経路にはオブジェクト識別子と主体識別子を含めない

### Example: EX-AUTHORIZATION-003-02 主体が subject set の成員として間接的に関係を持つ

- Given 認可モデルが `computed_userset` と `tuple_to_userset` を含む関係を宣言している
- And グループの成員、親フォルダーの閲覧者、直接の編集者のタプルが登録されている
- When 呼び出し元が主体とリソースと関係を CheckAccess へ渡す
- But 主体が subject set の成員として間接的に関係を持つ
- Then 許可する

### Example: EX-AUTHORIZATION-003-03 主体が親オブジェクト側で関係を持つ

- Given 認可モデルが `computed_userset` と `tuple_to_userset` を含む関係を宣言している
- And グループの成員、親フォルダーの閲覧者、直接の編集者のタプルが登録されている
- When 呼び出し元が主体とリソースと関係を CheckAccess へ渡す
- But 主体が親オブジェクト側で関係を持つ
- Then 許可する

### Example: EX-AUTHORIZATION-003-04 どの経路でも関係に到達しない

- Given 認可モデルが `computed_userset` と `tuple_to_userset` を含む関係を宣言している
- And グループの成員、親フォルダーの閲覧者、直接の編集者のタプルが登録されている
- When 呼び出し元が主体とリソースと関係を CheckAccess へ渡す
- But どの経路でも関係に到達しない
- Then 許可しない

## Rule: REQ-AUTHORIZATION-004 代行するエージェントは主体と自身の双方が関係を持つときだけ許可される

### Scenario Outline: 主体、代行者、スコープ、テナント境界から許可を決める

- Given 代行されるユーザーが対象リソースに対して関係を <subject_relation>
- And エージェント自身が同じ関係を <actor_relation>
- And 代行チェーン上の <chain_status>
- And 要求した関係に対応するスコープが提示トークンのスコープ集合に <scope>
- And 主体と全 actor は同じ <tenant_boundary> に属する
- When エージェントが自身を代行チェーンに載せて CheckAccess を呼ぶ
- Then 主体・全 actor・スコープ・テナントのすべてを満たしたときにだけ <decision>
- And 判定はエージェントが代行するユーザーの権限を超えない

#### Examples: Decision table (Unique)

  | example_id | subject_relation | actor_relation | chain_status | scope | tenant_boundary | decision |
  | --- | --- | --- | --- | --- | --- | --- |
  | EX-AUTHORIZATION-004-01 | 持つ | 持つ | プリンシパルはすべて有効である | 含まれる | テナント | 許可する |
  | EX-AUTHORIZATION-004-02 | 持つ | 持たない | プリンシパルはすべて有効である | 含まれる | テナント | 許可しない |
  | EX-AUTHORIZATION-004-03 | 持つ | 持つ | いずれかのプリンシパルが有効でない、または状態を解決できない | 含まれる | テナント | 許可しない |
  | EX-AUTHORIZATION-004-04 | 持つ | 持つ | プリンシパルはすべて有効である | 含まれない | テナント | 許可しない |

## Rule: REQ-AUTHORIZATION-005 判定不能はフェイルクローズで不許可になる

### Example: EX-AUTHORIZATION-005-01 通常経路

- Given テナントに認可モデルが登録済みである
- When 呼び出し元が CheckAccess を呼ぶ
- Then いずれの場合も許可へ退避せず、拒否した規則名を結果に残す

### Example: EX-AUTHORIZATION-005-02 探索の深さが上限を超える

- Given テナントに認可モデルが登録済みである
- When 呼び出し元が CheckAccess を呼ぶ
- But 探索の深さが上限を超える
- Then 拒否理由を添えて許可しない

### Example: EX-AUTHORIZATION-005-03 モデルが宣言していない型または関係を指定した

- Given テナントに認可モデルが登録済みである
- When 呼び出し元が CheckAccess を呼ぶ
- But モデルが宣言していない型または関係を指定した
- Then 拒否理由を添えて許可しない

### Example: EX-AUTHORIZATION-005-04 タプルストアへ到達できない

- Given テナントに認可モデルが登録済みである
- When 呼び出し元が CheckAccess を呼ぶ
- But タプルストアへ到達できない
- Then エラーを返し、許可しない

### Example: EX-AUTHORIZATION-005-05 関係の事実を組み立てないまま評価器へ届いた

- Given テナントに認可モデルが登録済みである
- When 呼び出し元が CheckAccess を呼ぶ
- But 関係の事実を組み立てないまま評価器へ届いた
- Then 規則 `relationship_facts_present` により許可しない

## Rule: REQ-AUTHORIZATION-006 他テナントの関係タプルは判定に寄与しない

### Example: EX-AUTHORIZATION-006-01 通常経路

- Given 別テナントに同じリソース識別子・関係・主体識別子のタプルが登録されている
- When 呼び出し元が自テナントで CheckAccess を呼ぶ
- Then 別テナントのタプルは読み出されず、判定は不許可になる

### Example: EX-AUTHORIZATION-006-02 リクエスト本体が別テナントの識別子を含む

- Given 別テナントに同じリソース識別子・関係・主体識別子のタプルが登録されている
- When 呼び出し元が自テナントで CheckAccess を呼ぶ
- But リクエスト本体が別テナントの識別子を含む
- Then 呼び出し元のテナントで解決した境界が優先され、対象テナントは変わらない

### Example: EX-AUTHORIZATION-006-03 別テナントで発行された整合トークンを提示した

- Given 別テナントに同じリソース識別子・関係・主体識別子のタプルが登録されている
- When 呼び出し元が自テナントで CheckAccess を呼ぶ
- But 別テナントで発行された整合トークンを提示した
- Then ConsistencyNotSatisfiedError で拒否する

## Rule: REQ-AUTHORIZATION-007 リソースの列挙は許可されたものだけを返し、打ち切りを隠さない

### Example: EX-AUTHORIZATION-007-01 通常経路

- Given 主体が一部のリソースにだけ関係を持つ
- When 呼び出し元が主体・リソース型・関係を ListAccessibleResources へ渡す
- Then 許可されたリソース識別子だけが返り、関係を持たないリソースは含まれない
- Then 判定は CheckAccess と同じ合成を通り、代行チェーンも同様に評価される
- Then 監査には 1 件ごとの判定ではなく、候補数・許可数・打ち切りをまとめた 1 件だけが残る

### Example: EX-AUTHORIZATION-007-02 走査が上限に達した

- Given 主体が一部のリソースにだけ関係を持つ
- When 呼び出し元が主体・リソース型・関係を ListAccessibleResources へ渡す
- But 走査が上限に達した
- Then 打ち切りを示し、結果を完全な一覧として扱わせない

## Rule: REQ-AUTHORIZATION-009 判定の監査は非個人識別情報の要約だけを残す

### Example: EX-AUTHORIZATION-009-01 通常経路

- Given 判定に用いる認可モデルとタプルが登録済みである
- When CheckAccess が判定を下す
- Then 監査イベントはリソース型、関係、許可・不許可、モデルの版、関係名だけの経路、拒否理由、代行チェーンの段数を持つ
- Then リソース識別子はダイジェストとして残り、主体識別子とタプルの内容は監査へ複製されない
