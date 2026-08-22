# Authorization Scenarios

### REQ-AUTHORIZATION-001: 管理者は認可モデルを版として登録でき、整合しないモデルは拒否される
- ACTOR TenantAdministrator
- GIVEN `AdminAuthorizationModelManage` を持つ管理者として認証済みである
- WHEN 管理者がリソース型と関係の定義を PutAuthorizationModel へ渡す
  - ALT 定義が宣言されていない型または関係を参照する → AuthorizationModelInvalidError で拒否し、版を作らない
  - ALT 書き換え規則が循環する → AuthorizationModelInvalidError で拒否し、版を作らない
  - ALT 型名または関係名が書式に反する → AuthorizationModelInvalidError で拒否し、版を作らない
- THEN テナント内で単調増加する新しい版が作られ、以前の版は書き換わらない
- THEN 応答は整合トークンを含み、GetAuthorizationModel が新しい版を最新として返す

### REQ-AUTHORIZATION-002: 関係タプルの書き込みは登録済みモデルに適合するものだけを一括で適用する
- ACTOR TenantAdministrator
- GIVEN テナントに認可モデルが登録済みである
- WHEN 管理者が追加と削除を含む差分を WriteRelationTuples へ渡す
  - ALT モデルが宣言していない型・関係を含む → RelationTupleInvalidError で拒否し、1 件も適用しない
  - ALT `direct` 規則が許していない主体型またはワイルドカードを含む → RelationTupleInvalidError で拒否し、1 件も適用しない
  - ALT 同じ組が追加と削除の双方に現れる → RelationTupleInvalidError で拒否し、1 件も適用しない
  - ALT テナントに認可モデルが未登録である → AuthorizationModelNotFoundError で拒否する
- THEN 差分は 1 トランザクションで適用され、既に存在する組の再追加は冪等に扱われる
- THEN 応答は書き込み後の整合トークンを返し、以後の判定へ渡せる

### REQ-AUTHORIZATION-003: 判定は継承・グループ・親子関係をたどって関係の成否を決める
- ACTOR ResourceServer
- GIVEN 認可モデルが `computed_userset` と `tuple_to_userset` を含む関係を宣言している
- GIVEN グループの成員、親フォルダーの閲覧者、直接の編集者のタプルが登録されている
- WHEN 呼び出し元が主体とリソースと関係を CheckAccess へ渡す
  - ALT 主体が subject set の成員として間接的に関係を持つ → 許可する
  - ALT 主体が親オブジェクト側で関係を持つ → 許可する
  - ALT どの経路でも関係に到達しない → 許可しない
- THEN 結果は許可・不許可と、たどった関係名だけの経路を返す
- THEN 経路にはオブジェクト識別子と主体識別子を含めない

### REQ-AUTHORIZATION-004: 代行するエージェントは主体と自身の双方が関係を持つときだけ許可される
- ACTOR Agent
- GIVEN 代行されるユーザーが対象リソースに対して関係を持つ
- WHEN エージェントが自身を代行チェーンに載せて CheckAccess を呼ぶ
  - ALT エージェント自身が同じ関係を持たない → 許可しない
  - ALT 代行チェーン上のいずれかのプリンシパルが有効でない、または状態を解決できない → 許可しない
  - ALT 要求した関係に対応するスコープが提示トークンのスコープ集合に含まれない → 許可しない
- THEN 主体・全 actor・スコープ・テナントのすべてを満たしたときにだけ許可する
- THEN 判定はエージェントが代行するユーザーの権限を超えない

### REQ-AUTHORIZATION-005: 判定不能はフェイルクローズで不許可になる
- ACTOR ResourceServer
- GIVEN テナントに認可モデルが登録済みである
- WHEN 呼び出し元が CheckAccess を呼ぶ
  - ALT 探索の深さが上限を超える → 拒否理由を添えて許可しない
  - ALT モデルが宣言していない型または関係を指定した → 拒否理由を添えて許可しない
  - ALT タプルストアへ到達できない → エラーを返し、許可しない
  - ALT 関係の事実を組み立てないまま評価器へ届いた → 規則 `relationship_facts_present` により許可しない
- THEN いずれの場合も許可へ退避せず、拒否した規則名を結果に残す

### REQ-AUTHORIZATION-006: 他テナントの関係タプルは判定に寄与しない
- ACTOR ResourceServer
- GIVEN 別テナントに同じリソース識別子・関係・主体識別子のタプルが登録されている
- WHEN 呼び出し元が自テナントで CheckAccess を呼ぶ
  - ALT リクエスト本体が別テナントの識別子を含む → 呼び出し元のテナントで解決した境界が優先され、対象テナントは変わらない
  - ALT 別テナントで発行された整合トークンを提示した → ConsistencyNotSatisfiedError で拒否する
- THEN 別テナントのタプルは読み出されず、判定は不許可になる

### REQ-AUTHORIZATION-007: リソースの列挙は許可されたものだけを返し、打ち切りを隠さない
- ACTOR ResourceServer
- GIVEN 主体が一部のリソースにだけ関係を持つ
- WHEN 呼び出し元が主体・リソース型・関係を ListAccessibleResources へ渡す
  - ALT 走査が上限に達した → 打ち切りを示し、結果を完全な一覧として扱わせない
- THEN 許可されたリソース識別子だけが返り、関係を持たないリソースは含まれない
- THEN 判定は CheckAccess と同じ合成を通り、代行チェーンも同様に評価される
- THEN 監査には 1 件ごとの判定ではなく、候補数・許可数・打ち切りをまとめた 1 件だけが残る

### REQ-AUTHORIZATION-008: オブジェクトの削除はその両側の関係タプルを取り除く
- ACTOR TenantAdministrator
- GIVEN あるオブジェクトが、リソース側と主体側の双方でタプルに現れている
- WHEN 管理者がそのオブジェクトを削除対象として WriteRelationTuples へ渡す
- THEN そのオブジェクトを参照するタプルは、リソース側・主体側のいずれも残らない
- THEN 削除に依存していた間接的な関係は以後成立しなくなり、整合トークンが進む

### REQ-AUTHORIZATION-009: 判定の監査は非個人識別情報の要約だけを残す
- ACTOR ResourceServer
- GIVEN 判定に用いる認可モデルとタプルが登録済みである
- WHEN CheckAccess が判定を下す
- THEN 監査イベントはリソース型、関係、許可・不許可、モデルの版、関係名だけの経路、拒否理由、代行チェーンの段数を持つ
- THEN リソース識別子はダイジェストとして残り、主体識別子とタプルの内容は監査へ複製されない

### REQ-AUTHORIZATION-010: 認可モデルとタプルの更新も判定の呼び出しも管理者に限られる
- ACTOR TenantAdministrator
- GIVEN "alice" は認証済みだが `admin` ロールを持たない
- WHEN "alice" が PutAuthorizationModel または WriteRelationTuples を呼ぶ
  - ALT "alice" が CheckAccess または ListAccessibleResources を呼ぶ → AccessDeniedError で拒否される
- THEN AccessDeniedError で拒否される
- THEN 認可モデルの版は 1 つも作られず、タプルは 1 件も書き込まれない
