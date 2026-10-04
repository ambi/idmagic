# Authorization

## 責務と境界

リソース 1 件ごとの細粒度の認可を扱う。
テナントごとの認可モデル（リソース型と関係の定義）と関係タプル（`resource ⇄ subject` の事実）を持ち、それらをたどって「この主体はこのリソースに対してこの関係を持つか」を判定する。
粗いロールでは表せない「ユーザー U は文書 D を読めるか」に答えるための Context である。

| 扱わないもの | 担当 |
| --- | --- |
| 最終的な認可の判断の合成（ロール、スコープ、代行チェーン、プリンシパルの有効性との論理積） | `OAuth2` の AuthZEN スタイルの `Authorizer` の規則表 |
| 管理 API を誰が呼べるかというロールの認可 | `OAuth2` の規則表と[認可設計](../../design/security/authorization.md) |
| `Agent` の状態 | `IdManagement` |

この Context は、関係が成り立つかという事実を求め、評価器へ渡す。
外部の PDP へ差し替えても、合成の規則が重複しない。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `AuthorizationModel` | 追記だけで積み上げる版ごとの `ResourceTypeDefinition`、`RelationDefinition`、`RelationRewrite` | `Tenant` を `tenant_id` で参照する。最新の版が判定に使われる |
| `RelationTuple` | `(resource_type:resource_id, relation, subject)` の関係の事実 | `Tenant` を `tenant_id` で参照する。型と関係は認可モデルが宣言したものに限る |

`ResourceTypeDefinition`、`RelationDefinition`、`RelationRewrite` は `AuthorizationModel` の版の内側にあり、版を経ずに変更しない。
テナントごとに単調に増える書き込みの版を持ち、書き込みは整合トークンを返す。

- **判断**：関係の定義を和だけで組む理由は、[書き換え規則を和だけで組み交差と差集合を入れない](design/decisions.md#書き換え規則を和だけで組み交差と差集合を入れない)。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Authorization` のタグが、採用する標準の規則は[Authorization の標準仕様](standards.md)が定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `CheckAccess` と `ListAccessibleResources` のユースケース | データ資源を提供する呼び出し側 | この Context が提供する | 関係の事実を組み立て、評価器の `resource:access` の規則で判定する |
| `PrincipalStatusResolver` | `IdManagement` の `Agent` の状態を読むアダプター | この Context が定める | 代行チェーンの actor が有効かを解決する |
| ドメインイベント | 監査と下流が購読する | この Context が発行する | `AuthorizationModelPublished`、`RelationTupleWritten`、`RelationTupleDeleted`、`FgaCheckEvaluated`、`FgaResourcesEnumerated` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [認可モデル](model/README.md) | 認可モデルの版の登録と検証 |
| [関係タプル](relation-tuple/README.md) | 関係タプルの一括の書き込みと、オブジェクトの削除 |
| [関係の判定](check/README.md) | 関係の判定、代行チェーンの合成、リソースの列挙 |

| 文書 | 内容 |
| --- | --- |
| [Authorization の用語集](glossary.md) | この Context での語義 |
| [Authorization の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Authorization の設計](design/README.md) | 話題ごとの設計と重要な判断 |
