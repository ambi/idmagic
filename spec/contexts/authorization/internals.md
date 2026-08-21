# Authorization Internals

## 関係の言語

型名と関係名は `^[a-z][a-z0-9_]*$` で最大 64 文字とする。`RelationDefinition` は書き換え規則の**和**で成立条件を表し、規則は 3 種類だけである。

- `direct`: 直接の関係タプル。`direct_subject_types` が受け入れる主体の形を宣言する。`user` は個別の主体、`group#member` は subject set、`user:*` はワイルドカードを表す。宣言のない形のタプルは書き込み時に拒否し、モデルが変わって宣言から外れた既存タプルは判定時にも数えない。
- `computed_userset`: 同じオブジェクト上の別関係へ委ねる (`viewer` は `editor` を含む)。
- `tuple_to_userset`: `tupleset_relation` でたどった先のオブジェクトで `computed_relation` を判定する (`document#parent` をたどって `folder#viewer` を見る)。

主体は `type:id`、`type:id#relation` (subject set)、`type:*` (ワイルドカード) の 3 形を取る。ワイルドカードは `direct` 規則がその型を許した場合にだけ受理する。

## 評価

深さ制限付きの深さ優先探索で判定する。訪問済みの `(オブジェクト, 関係)` 対を記録するため、モデルが循環していても探索は停止する。深さの上限は 8 とし、上限超過、循環、未知の型・関係、ストアの読み出し失敗はいずれも**許可しない**。呼び出し側が判定不能を許可として扱えないよう、これらは通常の不許可と区別できる拒否理由を付けて返す。

結果は、たどった関係名だけを連ねた経路 (`document#viewer` → `folder#viewer`) を持つ。オブジェクト識別子と主体識別子は経路に含めない。経路は運用者がモデルの誤りを追うための情報であり、資源の名前を配る手段ではない。

## 代行チェーンの合成

`CheckAccess` は次を論理積で合成する。

1. 主体 (通常はアクセストークンの `sub`) が対象リソースに対して関係を持つ。
2. 代行チェーン上の**すべての** actor が同じ関係を持つ。
3. 代行チェーン上のすべての actor が有効である。`Agent` の状態は `PrincipalStatusResolver` ポートで解決し、解決できなければ有効とみなさない。
4. 要求した関係に対応するスコープが、提示されたトークンのスコープ集合に含まれる。
5. 主体とリソースのテナントが一致する。

1〜3 は本 Context が事実として組み立て、4〜5 と合わせて AuthZEN の `resource:access` 規則が評価する。事実が欠けたまま届いた要求は規則 `relationship_facts_present` が不許可にするので、事実の供給を忘れた経路が黙って許可になることはない。

## 整合

テナントごとに単調増加する書き込み版を持ち、タプル書き込みとモデル登録は同じトランザクションでこれを進める。書き込みはテナント識別子と版を束縛した不透明な整合トークンを返し、判定は `minimum_consistency` としてそれを提示できる。ストアの版がトークンより古い場合、およびトークンが別テナントのものである場合は fail-closed で拒否する。

単一の PostgreSQL を使う現在の構成では読み取りが強整合なので、このトークンは主に、書き込み直後の管理操作がその書き込みを参照できることの確認に使う。また、将来読み取り経路へキャッシュや複製を導入する場合に守るべき契約を、あらかじめ明示する役割も持つ。

## 永続化

`authorization_models` はモデルの版を追記のみで保持し、定義は JSONB に置く。定義は外部から与えられる構造であり、結合や絞り込みの対象にならないためである。`authorization_relation_tuples` はタプルそのものを列に展開し、`(tenant_id, resource_type, resource_id, relation, subject_type, subject_id, subject_relation)` を主キーとする。同じ組の再書き込みが冪等になり、判定の絞り込みが主キーの先頭から効く。主体側からの走査のために `(tenant_id, subject_type, subject_id, subject_relation, resource_type)` の索引を持つ。`authorization_write_versions` はテナントごとの書き込み版を 1 行で保持する。

メモリアダプターは同じ契約テストを共有し、テストとローカルデモの参照実装として PostgreSQL 版と同じ振る舞いを持つ。

## 監査

`AuthorizationModelPublished` / `RelationTupleWritten` / `RelationTupleDeleted` / `FgaCheckEvaluated` / `FgaResourcesEnumerated` を発行する。タプルの内容と主体識別子は監査へ複製しない。`FgaCheckEvaluated` はリソース識別子をテナントと型を混ぜた SHA-256 の先頭 16 桁ダイジェストにするので、同一資源への繰り返しアクセスは相関できるが、資源の名前そのものは監査ログから復元できず、テナントをまたいだ相関もできない。

## CheckAccess
主体、リソース、関係、代行チェーン、任意の整合トークンを受け取り、関係の成否を事実として組み立てて `Authorizer` の `resource:access` 評価へ渡す。結果は許可・不許可、用いたモデルの版、整合トークン、関係名だけの経路、拒否した規則名を持つ。モデルが未登録、整合トークンを満たせない、ストアへ到達できない場合はエラーを返し、許可へ退避しない。

## ListAccessibleResources
主体、リソース型、関係、代行チェーンを受け取り、そのテナント・そのリソース型に現れる識別子を上限つきで走査し、`CheckAccess` と同じ合成で許可されたものだけを返す。上限に達した場合は打ち切りを示す。

## PrincipalStatusResolver
代行チェーン上のプリンシパルが有効かどうかを解決する。`Agent` の状態は IdManagement が正であり、本 Context はポート越しに問い合わせるだけで判断の実体を持たない。解決できない場合は有効とみなさない。
