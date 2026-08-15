---
context: authorization
updated_at: 2026-08-15
---

# Authorization Specification

## Overview

リソース 1 件ごとの細粒度な認可を所有する。テナントごとの認可モデル (リソース型と関係の定義) と関係タプル (`resource ⇄ subject` の事実) を持ち、それらをたどって「この主体はこのリソースに対してこの関係を持つか」を判定する。粗いロールでは表現できない「ユーザー U は文書 D を読めるか」に答えるための Context である。

判定の合成そのものは所有しない。関係の成否という**事実**を組み立て、OAuth2 が所有する AuthZEN スタイルの `Authorizer` ポートへ渡す。ロール、スコープ、代行チェーン、プリンシパルの有効性との論理積は評価器側の規則表が担う。これにより、外部 PDP へ差し替えても合成規則の所在が二重化しない。

管理面のロール認可 (誰が管理 API を呼べるか) は引き続き OAuth2 の規則表が持つ。本 Context が扱うのは、データ資源へのアクセス判定である。

## Glossary

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

## Standards

### OpenID AuthZEN Authorization API

https://openid.net/specs/authorization-api-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| AUTHZEN-FGA-EVALUATION | required | MUST | 関係に基づく判定は `{subject, action, resource, context}` の評価に載せ、関係の成否は判定 context の事実として渡す。 |
| AUTHZEN-FGA-ACTOR-CHAIN | required | MUST | 代行チェーンは判定 context に明示的に載せ、各段のプリンシパル種別・識別子・有効性を分離して表す。 |
| AUTHZEN-FGA-FAIL-CLOSED | required | MUST | 評価器が判定を返せない、事実が欠けている、深さ上限に達した、ストアへ到達できないいずれの場合も許可へ退避しない。 |
| AUTHZEN-FGA-SEARCH | optional | MAY | 主体を固定してリソースを列挙する探索 (Search) を提供する。上限つきの走査で、打ち切りを結果に示す。 |

### OAuth 2.0 Token Exchange

RFC 8693 — https://www.rfc-editor.org/rfc/rfc8693.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8693-FGA-ACTOR-AND | required | MUST | 代行トークンでの判定は、`sub` の主体と `act` チェーン上のすべての actor が同じ関係を持つときにだけ許可する。エージェントは代行するユーザーの権限を超えられない。 |

## Authorization Boundary

認可モデルと関係タプルの管理は `AdminAuthorizationModelManage` 権限 (AuthZEN action `admin:authorization_model_manage`) を要する。この権限はテナント管理者に属し、テナント境界を越えない。この Context の管理 API は対話セッション限定であり、API アクセストークンからはどのスコープを持っていても到達できない。`ApiTokenScope` は認可モデルと関係タプルに対応する語彙をまだ持たず、既存のスコープへ畳み込めば、そのスコープを持つトークンが黙って認可判定の書き換え能力を得るからである。

関係タプルの読み書きは、常に呼び出し元のテナントで解決した `tenant_id` に閉じる。リクエスト本体が別テナントの識別子を含んでいても、それが判定や書き込みの対象テナントを変えることはない。判定に用いるタプルの読み出しも同じ境界で行うため、他テナントへ書き込まれたタプルが判定へ寄与する経路は存在しない。

`CheckAccess` と `ListAccessibleResources` は診断と内部利用の接点である。関係の有無そのものが情報になるため、これらも管理権限を要する。データ資源を提供する呼び出し側が代行アクセスを判定する経路は、HTTP ではなくユースケースの直接呼び出しとする。

## Design

### 関係の言語

型名と関係名は `^[a-z][a-z0-9_]*$` で最大 64 文字とする。`RelationDefinition` は書き換え規則の**和**で成立条件を表し、規則は 3 種類だけである。

- `direct`: 直接の関係タプル。`direct_subject_types` が受け入れる主体の形を宣言する。`user` は個別の主体、`group#member` は subject set、`user:*` はワイルドカードを表す。宣言のない形のタプルは書き込み時に拒否し、モデルが変わって宣言から外れた既存タプルは判定時にも数えない。
- `computed_userset`: 同じオブジェクト上の別関係へ委ねる (`viewer` は `editor` を含む)。
- `tuple_to_userset`: `tupleset_relation` でたどった先のオブジェクトで `computed_relation` を判定する (`document#parent` をたどって `folder#viewer` を見る)。

主体は `type:id`、`type:id#relation` (subject set)、`type:*` (ワイルドカード) の 3 形を取る。ワイルドカードは `direct` 規則がその型を許した場合にだけ受理する。

交差 (intersection) と差集合 (exclusion) は導入しない。和だけで構成すると評価は単調になり、規則やタプルの追加が既存の許可を取り消さない。取り消しを表現する必要が生じた時点で、影響範囲を見積もったうえで改めて仕様を変える。

### 評価

深さ制限つきの深さ優先探索で判定する。既訪問の `(オブジェクト, 関係)` 対を記録するため、モデルが循環していても探索は止まる。深さの上限は 8 とし、超過、循環、未知の型・関係、ストアの読み出し失敗はいずれも**許可しない**。呼び出し側が「判定不能」を許可として扱う余地を残さないため、これらは結果としての不許可ではなく拒否理由付きの不許可として返る。

結果は、たどった関係名だけを連ねた経路 (`document#viewer` → `folder#viewer`) を持つ。オブジェクト識別子と主体識別子は経路に含めない。経路は運用者がモデルの誤りを追うための情報であり、資源の名前を配る手段ではない。

`ListAccessibleResources` は逆引きインデックスを持たず、そのテナント・そのリソース型に現れる識別子を上限つきで走査して 1 件ずつ判定する。上限に達したときは `truncated` を立て、呼び出し側が完全な一覧として扱えないようにする。正しさを先に固定するための構成であり、規模が問題になった時点で逆引きの持ち方を改めて設計する。

### 代行チェーンの合成

`CheckAccess` は次を論理積で合成する。

1. 主体 (通常はアクセストークンの `sub`) が対象リソースに対して関係を持つ。
2. 代行チェーン上の**すべての** actor が同じ関係を持つ。
3. 代行チェーン上のすべての actor が有効である。`Agent` の状態は `PrincipalStatusResolver` ポートで解決し、解決できなければ有効とみなさない。
4. 要求した関係に対応するスコープが、提示されたトークンのスコープ集合に含まれる。
5. 主体とリソースのテナントが一致する。

1〜3 は本 Context が事実として組み立て、4〜5 と合わせて AuthZEN の `resource:access` 規則が評価する。事実が欠けたまま届いた要求は規則 `relationship_facts_present` が不許可にするので、事実の供給を忘れた経路が黙って許可になることはない。

### 整合

テナントごとに単調増加する書き込み版を持ち、タプル書き込みとモデル登録は同じトランザクションでこれを進める。書き込みはテナント識別子と版を束縛した不透明な整合トークンを返し、判定は `minimum_consistency` としてそれを提示できる。ストアの版がトークンより古い場合、およびトークンが別テナントのものである場合は fail-closed で拒否する。

単一の PostgreSQL では読み取りは元から強整合なので、このトークンが実際に効くのは「書いた直後の管理操作が自分の書き込みを見たことを確かめる」場面である。同時に、読み取り経路にキャッシュや複製を入れたときに守るべき契約を、いま形として固定しておくためのものでもある。

### 永続化

`authorization_models` はモデルの版を追記のみで保持し、定義は JSONB に置く。定義は外部から与えられる構造であり、結合や絞り込みの対象にならないためである。`authorization_relation_tuples` はタプルそのものを列に展開し、`(tenant_id, resource_type, resource_id, relation, subject_type, subject_id, subject_relation)` を主キーとする。同じ組の再書き込みが冪等になり、判定の絞り込みが主キーの先頭から効く。主体側からの走査のために `(tenant_id, subject_type, subject_id, subject_relation, resource_type)` の索引を持つ。`authorization_write_versions` はテナントごとの書き込み版を 1 行で保持する。

メモリアダプターは同じ契約テストを共有し、テストとローカルデモの参照実装として PostgreSQL 版と同じ振る舞いを持つ。

### 監査

`AuthorizationModelPublished` / `RelationTupleWritten` / `RelationTupleDeleted` / `FgaCheckEvaluated` / `FgaResourcesEnumerated` を発行する。タプルの内容と主体識別子は監査へ複製しない。`FgaCheckEvaluated` はリソース識別子をテナントと型を混ぜた SHA-256 の先頭 16 桁ダイジェストにするので、同一資源への繰り返しアクセスは相関できるが、資源の名前そのものは監査ログから復元できず、テナントをまたいだ相関もできない。

列挙は 1 件ごとの判定を監査へ展開せず、走査全体を 1 件の `FgaResourcesEnumerated` にまとめる。1 回の列挙で候補数だけのイベントが並ぶと、本当に見るべき単発の判定がその中に埋もれるからである。

### Internal Interfaces

#### CheckAccess
主体、リソース、関係、代行チェーン、任意の整合トークンを受け取り、関係の成否を事実として組み立てて `Authorizer` の `resource:access` 評価へ渡す。結果は許可・不許可、用いたモデルの版、整合トークン、関係名だけの経路、拒否した規則名を持つ。モデルが未登録、整合トークンを満たせない、ストアへ到達できない場合はエラーを返し、許可へ退避しない。

#### ListAccessibleResources
主体、リソース型、関係、代行チェーンを受け取り、そのテナント・そのリソース型に現れる識別子を上限つきで走査し、`CheckAccess` と同じ合成で許可されたものだけを返す。上限に達した場合は打ち切りを示す。

#### PrincipalStatusResolver
代行チェーン上のプリンシパルが有効かどうかを解決する。`Agent` の状態は IdManagement が正であり、本 Context はポート越しに問い合わせるだけで判断の実体を持たない。解決できない場合は有効とみなさない。

## Scenarios

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
