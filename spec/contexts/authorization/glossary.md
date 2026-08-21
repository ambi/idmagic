# Authorization Glossary

| Term | Definition | Aliases |
|---|---|---|
| AuthorizationModel | テナントが公開しているリソース型と関係の定義の集合。版を追記のみで積み上げ、最新版が判定に使われる。未知の型・関係を参照する版、書き換え規則が循環する版は登録時に拒否する。 | 認可モデル |
| ResourceTypeDefinition | 認可モデルが宣言する 1 つのリソース型。関係タプルの `resource_type` と `subject_type` は、ここで宣言した型名しか取りえない。 | リソース型定義 |
| RelationDefinition | リソース型が公開する 1 つの関係の定義。書き換え規則の和で成立条件を表す。規則が空の関係は決して成立しない。 | 関係定義 |
| RelationRewrite | 関係の成立条件を構成する 1 つの規則。`direct` は直接タプル、`computed_userset` は同一オブジェクト上の別関係、`tuple_to_userset` は `tupleset_relation` でたどった先のオブジェクト上の関係を指す。交差と差集合は定義しない。 | 書き換え規則 |
| RelationTuple | `(resource_type:resource_id, relation, subject)` の関係事実。テナント内で一意な組で、同じ組の再書き込みは冪等である。 | 関係タプル, タプル |
| Subject set | `group:eng#member` のように、単一の主体ではなく「あるオブジェクトのある関係を持つ主体すべて」を指す主体表現。 | 主体集合, userset |
| Actor chain | RFC 8693 の `act` クレームが表す代行の連なりを、外側から内側の順に並べたもの。エージェントが主体を代行して行うアクセスで、各段が独立に関係を要求される。 | 代行チェーン |
| Consistency token | テナントごとの書き込み版を不透明に符号化した値。書き込みが返し、判定へ渡すと「ストアがその書き込み以降の状態であること」を要求できる。テナントを束縛しているため、他テナントのトークンは受理しない。 | 整合トークン |
| FgaCheckResult | 判定の結果。許可・不許可、用いたモデルの版、整合トークン、たどった関係名だけの経路要約、拒否した規則名を持つ。オブジェクト識別子と主体識別子は経路に含めない。 | 判定結果 |
