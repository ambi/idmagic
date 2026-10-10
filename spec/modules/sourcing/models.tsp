namespace IdMagic.Contract {

@doc("RFC 7644 の 3.12 節の SCIM のプロトコルエラーのレスポンス。SCIM は形を標準が定める接点なので Problem Details を返さず、ボディは `application/json` である。")
model ScimProtocolError {
  @doc("常に `urn:ietf:params:scim:api:messages:2.0:Error` の一要素だけである。")
  schemas: string[];
  @doc("HTTP のステータスコード。10 進数の文字列で表す。")
  status: string;
  @doc("発生した事象の、人が読む説明。常に英語で、ローカライズしない。")
  detail?: string;
  @doc("RFC 7644 の 3.12 節の表 9 にある SCIM の詳細エラーのキーワード。`invalidFilter`、`invalidValue`、`uniqueness` など。当てはまるキーワードがないときは省く。")
  scimType?: string;
}


@doc("SCIM 2.0 の id を、それが表す内部の User（`sub`）に対応付ける。")
model ScimUserRef {
  tenant_id: string;
  scim_id: string;
  user_id: string;
}


@doc("SCIM 2.0 の id を、それが表す内部の Group（グループの id）に対応付ける。")
model ScimGroupRef {
  tenant_id: string;
  scim_id: string;
  group_id: string;
}


@doc("/Users と /Groups のエンドポイントへ送り、そこから返る SCIM のリソース（RFC 7643）。含まれる属性はリソースが `schemas` で宣言するスキーマによって決まるので、契約は一つの属性の集合に固定せず、開いた JSON オブジェクトとして記録する。サーバーが従う属性の定義は GetScimSchemas が報告する。IdMagic は User と Group のコアスキーマと、User の enterprise 拡張に対応する。")
model ScimResource is Record<unknown>;


@doc("RFC 7644 の 3.4.2 節の SCIM の検索結果のエンベロープ。")
model ScimListResponse {
  @doc("常に `urn:ietf:params:scim:api:messages:2.0:ListResponse` の一要素だけである。")
  schemas: string[];
  @doc("このページに含まれる数ではなく、検索に一致したリソースの総数。")
  totalResults: int32;
  @doc("このページの最初のリソースの、1 から数えた位置。")
  startIndex: int32;
  @doc("このページに含まれるリソースの数。")
  itemsPerPage: int32;
  @doc("一致したリソースのページ。RFC 7644 の 3.4.2 節の綴りに合わせて大文字で始める。")
  Resources: ScimResource[];
}


@doc("SCIM の任意の機能の一つに対応しているかどうか（RFC 7643 の 5 節）。")
model ScimFeatureSupport {
  supported: boolean;
}


@doc("ServiceProviderConfig が宣言する、一括操作への対応（RFC 7643 の 5 節）。")
model ScimBulkConfig {
  supported: boolean;
  maxOperations: int32;
  maxPayloadSize: int32;
}


@doc("ServiceProviderConfig が宣言する、絞り込みへの対応（RFC 7643 の 5 節）。")
model ScimFilterConfig {
  supported: boolean;
  @doc("検索の `count` にかかわらず、サーバーが返すページの最大の大きさ。")
  maxResults: int32;
}


@doc("サービスプロバイダーが受け付ける認証方式の一つ（RFC 7643 の 5 節）。")
model ScimAuthenticationScheme {
  name: string;
  @doc("方式の説明。常に英語で、ローカライズしない。")
  description: string;
  @doc("方式を定める仕様の場所。")
  specUri?: string;
  @doc("方式の種類。`oauthbearertoken` など。")
  type: string;
}


@doc("RFC 7643 の 5 節の SCIM のサービスプロバイダーの設定。このデプロイメントが対応する任意の機能を報告する。")
model ScimServiceProviderConfig {
  @doc("常に `urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig` の一要素だけである。")
  schemas: string[];
  documentationUri?: string;
  patch: ScimFeatureSupport;
  bulk: ScimBulkConfig;
  filter: ScimFilterConfig;
  changePassword: ScimFeatureSupport;
  sort: ScimFeatureSupport;
  etag: ScimFeatureSupport;
  authenticationSchemes: ScimAuthenticationScheme[];
}


@doc("リソースタイプがコアスキーマの外に拡張するスキーマ（RFC 7643 の 6 節）。User の enterprise 拡張など。")
model ScimSchemaExtension {
  @doc("拡張するスキーマの URN。")
  schema: string;
  @doc("このリソースタイプのリソースがこの拡張を含まなければならないかどうか。")
  required: boolean;
}


@doc("サービスプロバイダーが公開するリソースタイプの一つ（RFC 7643 の 6 節）。")
model ScimResourceType {
  @doc("常に `urn:ietf:params:scim:schemas:core:2.0:ResourceType` の一要素だけである。")
  schemas: string[];
  id: string;
  name: string;
  @doc("リソースタイプを提供するパス。SCIM のベース URL からの相対パスである。")
  endpoint: string;
  @doc("リソースタイプの説明。常に英語で、ローカライズしない。")
  description?: string;
  @doc("このリソースタイプのコアスキーマの URN。")
  schema: string;
  schemaExtensions?: ScimSchemaExtension[];
}


@doc("SCIM のスキーマの中の属性の定義の一つ（RFC 7643 の 7 節）。")
model ScimSchemaAttribute {
  name: string;
  @doc("属性の型。`string`、`boolean`、`reference`、`complex` など。")
  type: string;
  multiValued: boolean;
  @doc("属性が保持するものの説明。常に英語で、ローカライズしない。")
  description?: string;
  required: boolean;
  caseExact: boolean;
  @doc("`readOnly`、`readWrite`、`immutable`、`writeOnly` のいずれか。")
  mutability: string;
  @doc("`always`、`never`、`default`、`request` のいずれか。")
  returned: string;
  @doc("`none`、`server`、`global` のいずれか。")
  uniqueness: string;
  @doc("`reference` の属性で、参照が指してよいリソースタイプ。")
  referenceTypes?: string[];
  @doc("`complex` の属性で、副属性の定義。")
  subAttributes?: ScimSchemaAttribute[];
}


@doc("サービスプロバイダーが従うスキーマの一つ（RFC 7643 の 7 節）。IdMagic は User のコアスキーマ、User の enterprise 拡張、Group のコアスキーマを報告する。")
model ScimSchema {
  @doc("常に `urn:ietf:params:scim:schemas:core:2.0:Schema` の一要素だけである。")
  schemas: string[];
  @doc("この文書が定義するスキーマの URN。")
  id: string;
  name: string;
  @doc("スキーマの用途の説明。常に英語で、ローカライズしない。")
  description?: string;
  attributes: ScimSchemaAttribute[];
}

}
