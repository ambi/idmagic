namespace IdMagic.Contract {

@doc("認可モデルが定める関係の書き換え規則の種別。direct は直接 RelationTuple、computed_userset は同一オブジェクト上の別関係、tuple_to_userset は tupleset_relation でたどった先のオブジェクト上の関係を指す。")
enum RelationRewriteKind {
  direct: "direct",
  computed_userset: "computed_userset",
  tuple_to_userset: "tuple_to_userset",
}


@doc("1 つの関係を構成する書き換え規則。RelationDefinition が持つ規則の和 (union) が、その関係の成立条件になる。intersection と exclusion は定義しない。")
model RelationRewrite {
  kind: RelationRewriteKind;
  @doc("computed_userset と tuple_to_userset で評価する関係名。direct では指定しない。")
  @maxLength(64)
  computed_relation?: string;
  @doc("tuple_to_userset で親オブジェクトへたどる関係名。ほかの kind では指定しない。")
  @maxLength(64)
  tupleset_relation?: string;
  @doc("direct で受け入れる主体の形。user は個別の主体、group#member は subject set、user:* はワイルドカードを表す。ここに現れない形の RelationTuple は書き込み時も判定時も受け付けない。ほかの kind では指定しない。")
  direct_subject_types?: string[];
}


@doc("ResourceTypeDefinition が公開する 1 つの関係の定義。rewrites が空の関係は決して成立しない。")
model RelationDefinition {
  @doc("関係名。^[a-z][a-z0-9_]*$。同一 ResourceTypeDefinition 内で一意。")
  @minLength(1)
  @maxLength(64)
  name: string;
  rewrites: RelationRewrite[];
}


@doc("認可モデルが定めるリソース型。RelationTuple の resource_type と subject_type はここで宣言した型名だけを取りうる。")
model ResourceTypeDefinition {
  @doc("型名。^[a-z][a-z0-9_]*$。AuthorizationModel 内で一意。")
  @minLength(1)
  @maxLength(64)
  name: string;
  relations: RelationDefinition[];
}


@doc("テナントの認可モデル。版を追記のみで積み上げ、published された最新版が判定に使われる。未知の関係・型を参照する版は登録時に fail-closed で拒否する。")
model AuthorizationModel {
  @doc("生成識別子。UUID v4。不変。")
  id: string;
  @doc("所属テナント ID。cross-tenant 参照は禁止。")
  tenant_id: string = "default";
  @doc("テナント内で単調増加する版。1 から始まる。")
  version: int32;
  resource_types: ResourceTypeDefinition[];
  created_at: utcDateTime;
}


@doc("(resource_type:resource_id, relation, subject) の関係事実。テナント内で一意な組であり、同じ組の再書き込みは冪等。subject_relation を持つ主体は subject set (group:eng#member) を表す。subject_id が * のワイルドカードは、direct 規則が subject_type を許した場合にだけ受け付ける。")
model RelationTuple {
  @doc("関係を持たれる側のリソース型。認可モデルが宣言した型名。")
  @minLength(1)
  @maxLength(64)
  resource_type: string;
  @doc("リソース識別子。idmagic は採番せず、呼び出し側の資源空間の id をそのまま持つ。")
  @minLength(1)
  @maxLength(256)
  resource_id: string;
  @minLength(1)
  @maxLength(64)
  relation: string;
  @doc("主体型。認可モデルが宣言した型名。")
  @minLength(1)
  @maxLength(64)
  subject_type: string;
  @doc("主体識別子。* はワイルドカードを表す。")
  @minLength(1)
  @maxLength(256)
  subject_id: string;
  @doc("subject set を表す関係名。省略時は個別の主体を指す。")
  @maxLength(64)
  subject_relation?: string;
}


@doc("WriteRelationTuples が 1 トランザクションで適用する差分。writes と deletes は同じ組を同時に含んではならない。delete_objects に挙げたオブジェクトは、リソース側・主体側の双方の RelationTuple を取り除く (削除の波及)。")
model RelationTupleWriteRequest {
  writes?: RelationTuple[];
  deletes?: RelationTuple[];
  delete_objects?: FgaObjectReference[];
}


@doc("type:id 形式のオブジェクト参照。")
model FgaObjectReference {
  @minLength(1)
  @maxLength(64)
  type: string;
  @minLength(1)
  @maxLength(256)
  id: string;
}


@doc("判定時に主体を代行しているプリンシパルの 1 段。RFC 8693 の act チェーンを外側から内側の順に並べる。")
model FgaActor {
  @doc("認可モデルが宣言する主体型 (agent / user など)。判定でも有効性の解決でも同じ語彙を使う。")
  @minLength(1)
  type: string;
  @minLength(1)
  id: string;
}


@doc("CheckAccess の入力。subject は代行される主体 (通常はアクセストークンの sub)、actor_chain は代行しているエージェント列である。minimum_consistency を与えると、ストアの書き込み版がそれ以上であることを確認してから判定する。")
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
  @doc("この判定に必要な最小の整合トークン。WriteRelationTuples が返した値を渡す。")
  minimum_consistency?: string;
}


@doc("CheckAccess の結果。permitted が false のとき reasons が拒否した規則名を持つ。relation_path はたどった関係名だけの要約で、オブジェクト識別子や主体識別子を含まない。")
model FgaCheckResult {
  permitted: boolean;
  @doc("判定に用いた認可モデルの版。")
  model_version: int32;
  @doc("判定時点のストアの整合トークン。")
  consistency: string;
  relation_path: string[];
  reasons: string[];
}


@doc("ListAccessibleResources の結果。走査の上限に達した場合 truncated が true になり、呼び出し側はこれを完全な一覧として扱ってはならない。")
model FgaListAccessibleResourcesResult {
  resource_ids: string[];
  truncated: boolean;
  model_version: int32;
  consistency: string;
}


@doc("認可モデルの登録が検証で拒否された (未知の型・関係の参照、名前の書式違反、書き換え規則の循環、必須フィールドの欠落)。")
model AuthorizationModelInvalidError {

}


@doc("テナントに published 済みの認可モデルが存在しない。")
model AuthorizationModelNotFoundError {

}


@doc("RelationTuple が登録済み認可モデルに適合しない (未知の型・関係、許可されていない主体型、許可されていないワイルドカード、writes と deletes の重複)。")
model RelationTupleInvalidError {

}


@doc("提示された整合トークンがストアの書き込み版より新しい、または別テナントのトークンである。判定は fail-closed で拒否する。")
model ConsistencyNotSatisfiedError {

}


@doc("管理者がテナントの認可モデルの新しい版を published した。")
model AuthorizationModelPublished {
  occurredAt: utcDateTime;
  tenantId: string;
  modelId: string;
  version: int32;
  resourceTypeCount: int32;
}


@doc("RelationTuple を書き込んだ。個々のタプルは監査へ複製せず、件数と整合トークンだけを残す。")
model RelationTupleWritten {
  occurredAt: utcDateTime;
  tenantId: string;
  writtenCount: int32;
  consistency: string;
}


@doc("RelationTuple を削除した。オブジェクト単位の削除の波及もこのイベントで表す。")
model RelationTupleDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  deletedCount: int32;
  consistency: string;
}


@doc("CheckAccess が判定を下した。リソース識別子は SHA-256 の先頭 16 桁ダイジェストにし、主体識別子とタプルの内容は残さない。")
model FgaCheckEvaluated {
  occurredAt: utcDateTime;
  tenantId: string;
  resourceType: string;
  resourceIdDigest: string;
  relation: string;
  permitted: boolean;
  modelVersion: int32;
  @doc("たどった関係名だけの経路要約。")
  relationPath: string[];
  @doc("拒否した規則名。permitted が true のときは空。")
  reasons: string[];
  @doc("代行チェーンの段数。0 は直接アクセス。")
  actorChainDepth: int32;
}


@doc("ListAccessibleResources が走査を終えた。1 件ごとの判定は監査へ展開せず、この 1 件で件数と打ち切りだけを残す。リソース識別子は残さない。")
model FgaResourcesEnumerated {
  occurredAt: utcDateTime;
  tenantId: string;
  resourceType: string;
  relation: string;
  @doc("走査した候補の件数。")
  candidateCount: int32;
  @doc("許可された件数。")
  permittedCount: int32;
  @doc("走査の上限に達して打ち切ったかどうか。")
  truncated: boolean;
  modelVersion: int32;
  actorChainDepth: int32;
}

}
