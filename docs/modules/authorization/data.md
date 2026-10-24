# AccessControl のデータ

この文書は、Authorization の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `AuthorizationModel` | 追記だけで積み上げる版ごとの `ResourceTypeDefinition`、`RelationDefinition`、`RelationRewrite` | `Tenant` を `tenant_id` で参照する。最新の版が判定に使われる |
| `RelationTuple` | `(resource_type:resource_id, relation, subject)` の関係の事実 | `Tenant` を `tenant_id` で参照する。型と関係は認可モデルが宣言したものに限る |

`ResourceTypeDefinition`、`RelationDefinition`、`RelationRewrite` は `AuthorizationModel` の版の内側にあり、版を経ずに変更しない。
テナントごとに単調に増える書き込みの版を持ち、書き込みは整合トークンを返す。

- **判断**：関係の定義を和だけで組む理由は、[書き換え規則を和だけで組み交差と差集合を入れない](decisions.md#書き換え規則を和だけで組み交差と差集合を入れない)。
