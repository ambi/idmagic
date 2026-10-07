namespace IdMagic.Contract {

@doc("認可モデルがリレーションに使える書き換え規則の種類。`direct` は RelationTuple をそのまま読み、`computed_userset` は同じオブジェクトの別のリレーションを読み、`tuple_to_userset` は `tupleset_relation` でたどったオブジェクトのリレーションを読む。")
enum RelationRewriteKind {
  direct: "direct",
  computed_userset: "computed_userset",
  tuple_to_userset: "tuple_to_userset",
}


@doc("リレーションを構成する書き換え規則の一つ。リレーションは、その RelationDefinition の規則の和集合が成り立つときに成り立つ。積集合と除外は定義しない。")
model RelationRewrite {
  kind: RelationRewriteKind;
  @doc("`computed_userset` と `tuple_to_userset` が評価するリレーションの名前。`direct` では設定しない。")
  @maxLength(64)
  computed_relation?: string;
  @doc("`tuple_to_userset` が親のオブジェクトへたどるリレーション。他の種類では設定しない。")
  @maxLength(64)
  tupleset_relation?: string;
  @doc("`direct` が受け付けるサブジェクトの形。個々のサブジェクトには `user`、サブジェクトの集合には `group#member`、ワイルドカードには `user:*` を使う。ここに載っていない形の RelationTuple は、書き込みでも評価でも拒否する。他の種類では設定しない。")
  direct_subject_types?: string[];
}


@doc("ResourceTypeDefinition が公開する一つのリレーションの定義。書き換え規則のないリレーションは決して成り立たない。")
model RelationDefinition {
  @doc("リレーションの名前。`^[a-z][a-z0-9_]*$` に一致し、その ResourceTypeDefinition の中で一意である。")
  @minLength(1)
  @maxLength(64)
  name: string;
  rewrites: RelationRewrite[];
}


@doc("認可モデルが定義するリソースタイプ。RelationTuple の `resource_type` と `subject_type` には、ここで宣言した種類の名前だけを使える。")
model ResourceTypeDefinition {
  @doc("種類の名前。`^[a-z][a-z0-9_]*$` に一致し、AuthorizationModel の中で一意である。")
  @minLength(1)
  @maxLength(64)
  name: string;
  relations: RelationDefinition[];
}


@doc("テナントの認可モデル。リビジョンは追記するだけで、最新の公開済みのリビジョンがすべてのチェックを決める。未知のリレーションまたは種類を参照するリビジョンは、登録時にフェイルクローズで拒否する。")
model AuthorizationModel {
  @doc("生成した識別子（UUID v4）。変わらない。")
  id: string;
  @doc("属するテナント。テナントをまたぐ参照は禁止する。")
  tenant_id: string = "default";
  @doc("テナントの中で 1 から始まり単調に増えるリビジョン。")
  version: int32;
  resource_types: ResourceTypeDefinition[];
  created_at: utcDateTime;
}


@doc("リレーションの事実 `(resource_type:resource_id, relation, subject)`。この三つ組はテナントの中で一意なので、同じものを再び書いても冪等である。`subject_relation` を含むサブジェクトはサブジェクトの集合（`group:eng#member`）を表す。`*` の `subject_id` はワイルドカードで、`direct` の規則がその `subject_type` を許すときだけ受け付ける。")
model RelationTuple {
  @doc("リレーションが成り立つリソースタイプ。認可モデルが宣言した種類の名前である。")
  @minLength(1)
  @maxLength(64)
  resource_type: string;
  @doc("リソースの識別子。IdMagic は採番せず、呼び出し元自身のリソースの空間の id をそのまま保持する。")
  @minLength(1)
  @maxLength(256)
  resource_id: string;
  @minLength(1)
  @maxLength(64)
  relation: string;
  @doc("サブジェクトの種類。認可モデルが宣言した種類の名前である。")
  @minLength(1)
  @maxLength(64)
  subject_type: string;
  @doc("サブジェクトの識別子。`*` はワイルドカードを表す。")
  @minLength(1)
  @maxLength(256)
  subject_id: string;
  @doc("これをサブジェクトの集合にするリレーションの名前。省くと個々のサブジェクトになる。")
  @maxLength(64)
  subject_relation?: string;
}


@doc("WriteRelationTuples が一つのトランザクションで適用する変更。同じ三つ組を `writes` と `deletes` の両方に含めてはならない。`delete_objects` に載せたオブジェクトは、リソース側とサブジェクト側の両方の RelationTuple を削除するので、削除が連鎖する。")
model RelationTupleWriteRequest {
  writes?: RelationTuple[];
  deletes?: RelationTuple[];
  delete_objects?: FgaObjectReference[];
}


@doc("`type:id` の形で書くオブジェクトの参照。")
model FgaObjectReference {
  @minLength(1)
  @maxLength(64)
  type: string;
  @minLength(1)
  @maxLength(256)
  id: string;
}


@doc("チェックの時点でサブジェクトに代わって動く主体の一段。RFC 8693 の `act` の連鎖で、外側から順に並べる。")
model FgaActor {
  @doc("認可モデルが宣言するサブジェクトの種類（`agent`、`user` など）。チェックと有効性の解決で同じ語彙を使う。")
  @minLength(1)
  type: string;
  @minLength(1)
  id: string;
}


@doc("CheckAccess の入力。`subject` は代理される主体（通常はアクセストークンの `sub`）で、`actor_chain` はそれに代わって動くエージェントの並びである。`minimum_consistency` を与えると、チェックは判定の前に、ストアの書き込みのリビジョンが少なくともその時点に達していることを確かめる。")
model FgaCheckRequest {
  @minLength(1)
  resource_type: string;
  @minLength(1)
  resource_id: string;
  @minLength(1)
  relation: string;
  @minLength(1)
  subject_type: string;
  @minLength(1)
  subject_id: string;
  actor_chain?: FgaActor[];
  @doc("このチェックに必要な最小の一貫性トークン。WriteRelationTuples が返した値を渡す。")
  minimum_consistency?: string;
  @doc("呼び出し元のアクセストークンが含むスコープ。操作が求めるものを満たさないとき、チェックはフェイルクローズで拒否する。")
  granted_scopes?: string[];
  @doc("操作が求めるスコープ。")
  required_scopes?: string[];
}


@doc("CheckAccess の結果。`permitted` が false のときは、`reasons` が拒否した規則を示す。`relation_path` はたどったリレーションの名前だけをまとめ、オブジェクトやサブジェクトの識別子は含まない。")
model FgaCheckResult {
  permitted: boolean;
  @doc("チェックに使った認可モデルのリビジョン。")
  model_version: int32;
  @doc("チェックの時点でのストアの一貫性トークン。")
  consistency: string;
  relation_path: string[];
  reasons: string[];
}


@doc("ListAccessibleResources の結果。走査の上限に達すると `truncated` を設定し、呼び出し元は一覧を完全なものとして扱ってはならない。")
model FgaListAccessibleResourcesResult {
  resource_ids: string[];
  truncated: boolean;
  model_version: int32;
  consistency: string;
}


@doc("認可モデルの登録が検証で拒否された。未知の種類またはリレーションへの参照、命名規則に反する名前、書き換え規則の循環、必須のフィールドの欠如のいずれかである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:authorization_model_invalid` である。")
model AuthorizationModelInvalidError is ProblemDetails;


@doc("テナントに公開済みの認可モデルがない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:authorization_model_not_found` である。")
model AuthorizationModelNotFoundError is ProblemDetails;


@doc("RelationTuple が登録済みの認可モデルに合わない。未知の種類またはリレーション、許されていないサブジェクトの種類、許されていないワイルドカード、または `writes` と `deletes` の両方にある同じ三つ組のいずれかである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:relation_tuple_invalid` である。")
model RelationTupleInvalidError is ProblemDetails;


@doc("提示した一貫性トークンが、ストアの保持する書き込みのリビジョンより新しいか、他のテナントに属している。チェックはフェイルクローズで拒否する。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:consistency_not_satisfied` である。")
model ConsistencyNotSatisfiedError is ProblemDetails;


@doc("管理者が、テナントの認可モデルの新しいリビジョンを公開した。")
model AuthorizationModelPublished {
  occurredAt: utcDateTime;
  tenantId: string;
  modelId: string;
  version: int32;
  resourceTypeCount: int32;
}


@doc("RelationTuple を書き込んだ。個々の三つ組は監査証跡に写さず、件数と一貫性トークンだけを残す。")
model RelationTupleWritten {
  occurredAt: utcDateTime;
  tenantId: string;
  writtenCount: int32;
  consistency: string;
}


@doc("RelationTuple を削除した。オブジェクト単位の連鎖する削除も、このイベントで記録する。")
model RelationTupleDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  deletedCount: int32;
  consistency: string;
}


@doc("CheckAccess が判定を下した。リソースの識別子は SHA-256 の先頭 16 桁の 16 進数に縮め、サブジェクトの識別子も三つ組の内容も残さない。")
model FgaCheckEvaluated {
  occurredAt: utcDateTime;
  tenantId: string;
  resourceType: string;
  resourceIdDigest: string;
  relation: string;
  permitted: boolean;
  modelVersion: int32;
  @doc("たどった経路の要約。リレーションの名前だけからなる。")
  relationPath: string[];
  @doc("拒否した規則。`permitted` が true のときは空である。")
  reasons: string[];
  @doc("委任の連鎖の段数。0 は直接のアクセスを表す。監査の `delegation.depth` の軸に使う。参加者そのものは含めない。この Context は主体の識別子を監査証跡に写さず、連鎖は、それを確立した OAuth2 の TokenExchanged を通じて関連付ける（REQ-AUDIT-006）。")
  actorChainDepth: int32;
}


@doc("ListAccessibleResources が走査を終えた。候補ごとの判定は監査証跡に展開せず、この一つのイベントに件数と、走査を途中で打ち切ったかどうかだけを残す。リソースの識別子は残さない。")
model FgaResourcesEnumerated {
  occurredAt: utcDateTime;
  tenantId: string;
  resourceType: string;
  relation: string;
  @doc("走査した候補の数。")
  candidateCount: int32;
  @doc("許可した候補の数。")
  permittedCount: int32;
  @doc("走査が上限に達して途中で打ち切られたかどうか。")
  truncated: boolean;
  modelVersion: int32;
  actorChainDepth: int32;
}

}
