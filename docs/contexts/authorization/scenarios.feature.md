# Feature: Authorization Scenarios

## Rule: REQ-AUTHORIZATION-001 管理者は認可モデルを版として登録でき、整合しないモデルは拒否される

Primary actor: `TenantAdministrator`

### Example: EX-AUTHORIZATION-001-01 通常経路

- Given `AdminAuthorizationModelManage` を持つ管理者として認証済みである
- When 管理者がリソース型と関係の定義を PutAuthorizationModel へ渡す
- Then テナント内で単調増加する新しい版が作られ、以前の版は書き換わらない
- Then レスポンスは整合トークンを含み、GetAuthorizationModel が新しい版を最新として返す

### Example: EX-AUTHORIZATION-001-02 定義が宣言されていない型または関係を参照する

- Given `AdminAuthorizationModelManage` を持つ管理者として認証済みである
- When 管理者がリソース型と関係の定義を PutAuthorizationModel へ渡す
- But 定義が宣言されていない型または関係を参照する
- Then AuthorizationModelInvalidError で拒否し、版を作らない

### Example: EX-AUTHORIZATION-001-03 書き換え規則が循環する

- Given `AdminAuthorizationModelManage` を持つ管理者として認証済みである
- When 管理者がリソース型と関係の定義を PutAuthorizationModel へ渡す
- But 書き換え規則が循環する
- Then AuthorizationModelInvalidError で拒否し、版を作らない

### Example: EX-AUTHORIZATION-001-04 型名または関係名が書式に反する

- Given `AdminAuthorizationModelManage` を持つ管理者として認証済みである
- When 管理者がリソース型と関係の定義を PutAuthorizationModel へ渡す
- But 型名または関係名が書式に反する
- Then AuthorizationModelInvalidError で拒否し、版を作らない

## Rule: REQ-AUTHORIZATION-002 関係タプルの書き込みは登録済みモデルに適合するものだけを一括で適用する

Primary actor: `TenantAdministrator`

### Example: EX-AUTHORIZATION-002-01 通常経路

- Given テナントに認可モデルが登録済みである
- When 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
- Then 差分は 1 トランザクションで適用され、既に存在する組の再追加は冪等に扱われる
- Then レスポンスは書き込み後の整合トークンを返し、以後の判定へ渡せる

### Example: EX-AUTHORIZATION-002-02 モデルが宣言していない型・関係を含む

- Given テナントに認可モデルが登録済みである
- When 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
- But モデルが宣言していない型・関係を含む
- Then RelationTupleInvalidError で拒否し、1 件も適用しない

### Example: EX-AUTHORIZATION-002-03 `direct` 規則が許していない主体型またはワイルドカードを含む

- Given テナントに認可モデルが登録済みである
- When 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
- But `direct` 規則が許していない主体型またはワイルドカードを含む
- Then RelationTupleInvalidError で拒否し、1 件も適用しない

### Example: EX-AUTHORIZATION-002-04 同じ組が追加と削除の双方に現れる

- Given テナントに認可モデルが登録済みである
- When 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
- But 同じ組が追加と削除の双方に現れる
- Then RelationTupleInvalidError で拒否し、1 件も適用しない

### Example: EX-AUTHORIZATION-002-05 テナントに認可モデルが未登録である

- Given テナントに認可モデルが登録済みである
- When 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
- But テナントに認可モデルが未登録である
- Then AuthorizationModelNotFoundError で拒否する

## Rule: REQ-AUTHORIZATION-003 判定は継承・グループ・親子関係をたどって関係の成否を決める

Primary actor: `ResourceServer`

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

Primary actor: `Agent`

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

Primary actor: `ResourceServer`

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

Primary actor: `ResourceServer`

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

Primary actor: `ResourceServer`

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

## Rule: REQ-AUTHORIZATION-008 オブジェクトの削除はその両側の関係タプルを取り除く

Primary actor: `TenantAdministrator`

### Example: EX-AUTHORIZATION-008-01 通常経路

- Given あるオブジェクトが、リソース側と主体側の双方でタプルに現れている
- When 管理者がそのオブジェクトを削除対象として WriteRelationTuples へ渡す
- Then そのオブジェクトを参照するタプルは、リソース側・主体側のいずれも残らない
- Then 削除に依存していた間接的な関係は以後成立しなくなり、整合トークンが進む

## Rule: REQ-AUTHORIZATION-009 判定の監査は非個人識別情報の要約だけを残す

Primary actor: `ResourceServer`

### Example: EX-AUTHORIZATION-009-01 通常経路

- Given 判定に用いる認可モデルとタプルが登録済みである
- When CheckAccess が判定を下す
- Then 監査イベントはリソース型、関係、許可・不許可、モデルの版、関係名だけの経路、拒否理由、代行チェーンの段数を持つ
- Then リソース識別子はダイジェストとして残り、主体識別子とタプルの内容は監査へ複製されない

## Rule: REQ-AUTHORIZATION-010 認可モデルとタプルの更新も判定の呼び出しも管理者に限られる

Primary actor: `TenantAdministrator`

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
