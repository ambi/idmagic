import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Tenancy")
namespace IdMagic.Tenancy {

namespace Operations {
model ListTenantsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantListResponse;
}

union ListTenantsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantsError403Body;
}

@doc("system_admin がテナントの一覧を読む。テナントを横断するので system_admin のロールで制限し、システムコンソール（/system）から呼び出す。デフォルトのテナントへ解決する接頭辞なしのパスと、/realms/default の下の制御面のパスの両方から同じハンドラーに届き、Cookie のセッションの一貫性を保つ。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:read"])
@TypeSpec.OpenAPI.operationId("ListTenants")
@route("/api/admin/v1/tenants")
@get
op ListTenants(

): ListTenantsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListTenantsError403;

model GetTenantSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantSummaryResponse;
}

union GetTenantError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantError403Body;
}

union GetTenantError404Body {
  IdMagic.Contract.TenantNotFoundError,
}

model GetTenantError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetTenantError404Body;
}

@doc("system_admin がテナントを一つ読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:read"])
@TypeSpec.OpenAPI.operationId("GetTenant")
@route("/api/admin/v1/tenants/{tenant_id}")
@get
op GetTenant(
  @path tenant_id: string
): GetTenantSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetTenantError403 | GetTenantError404;

model CreateTenantSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.TenantSummaryResponse;
}

union CreateTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateTenantError400Body;
}

union CreateTenantError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateTenantError403Body;
}

@doc("system_admin がテナントを作成する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("CreateTenant")
@route("/api/admin/v1/tenants")
@post
op CreateTenant(
  @body body: IdMagic.Contract.TenantCreateRequest
): CreateTenantSuccess_201 | CreateTenantError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateTenantError403;

model UpdateTenantSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantSummaryResponse;
}

union UpdateTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantError400Body;
}

union UpdateTenantError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantError403Body;
}

union UpdateTenantError422Body {
  IdMagic.Contract.PolicyOverrideWeakerError,
}

model UpdateTenantError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantError422Body;
}

@doc("system_admin がテナントの表示名を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenant")
@route("/api/admin/v1/tenants/{tenant_id}")
@patch
op UpdateTenant(
  @path tenant_id: string,
  @body body: IdMagic.Contract.TenantUpdateRequest
): UpdateTenantSuccess_200 | UpdateTenantError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateTenantError403 | UpdateTenantError422;

model DisableTenantSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableTenantError400Body;
}

union DisableTenantError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DisableTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableTenantError403Body;
}

@doc("system_admin が default 以外のテナントを無効にする。以降は /authorize、/token、/login などがすべて invalid_request になる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("DisableTenant")
@route("/api/admin/v1/tenants/{tenant_id}/disable")
@post
op DisableTenant(
  @path tenant_id: string,): DisableTenantSuccess_204 | DisableTenantError400 | IdMagic.Contract.AuthenticationRequiredResponse | DisableTenantError403;

model EnableTenantSuccess_204 {
  @statusCode statusCode: 204;
}

union EnableTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableTenantError400Body;
}

union EnableTenantError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model EnableTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableTenantError403Body;
}

@doc("system_admin が、無効にしたテナントを再び有効にする。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("EnableTenant")
@route("/api/admin/v1/tenants/{tenant_id}/enable")
@post
op EnableTenant(
  @path tenant_id: string,): EnableTenantSuccess_204 | EnableTenantError400 | IdMagic.Contract.AuthenticationRequiredResponse | EnableTenantError403;

model SetTenantEndpointStyleHttpRequest {
  endpoint_style: IdMagic.Contract.TenantEndpointStyle;
}

model SetTenantEndpointStyleSuccess_204 {
  @statusCode statusCode: 204;
}

union SetTenantEndpointStyleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SetTenantEndpointStyleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SetTenantEndpointStyleError400Body;
}

union SetTenantEndpointStyleError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model SetTenantEndpointStyleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SetTenantEndpointStyleError403Body;
}

@doc("system_admin が、テナントの正規の所在の形を切り替える。テナントの同一性を作り替える破壊的な操作なので、通常の属性の更新（UpdateTenant）とは分けている。発行者が変わるので、発行済みのトークンの `iss` の検証とすべての Relying Party の設定が壊れる。WebAuthn の RP ID が変わるので、既存のパスキーがすべて無効になる。Cookie のスコープが変わるので、進行中のセッションが切れる。`Subdomain` を選べるのは `tenant_base_domain` を設定したデプロイメントだけで、設定していなければリクエストは InvalidRequestError になる。レルムは変わらないので、切り替え後のホスト名は既存のレルムから決まる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("SetTenantEndpointStyle")
@route("/api/admin/v1/tenants/{tenant_id}/endpoint-style")
@put
op SetTenantEndpointStyle(
  @path tenant_id: string,
  @body body: SetTenantEndpointStyleHttpRequest
): SetTenantEndpointStyleSuccess_204 | SetTenantEndpointStyleError400 | IdMagic.Contract.AuthenticationRequiredResponse | SetTenantEndpointStyleError403;

model UpdateTenantQuotaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantQuota;
}

union UpdateTenantQuotaError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantQuotaError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantQuotaError400Body;
}

union UpdateTenantQuotaError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantQuotaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantQuotaError403Body;
}

@doc("system_admin が、一つのテナントの Quota を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["tenants:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantQuota")
@route("/api/admin/v1/tenants/{tenant_id}/quota")
@put
op UpdateTenantQuota(
  @path tenant_id: string,
  @body body: IdMagic.Contract.TenantQuotaUpdateRequest
): UpdateTenantQuotaSuccess_200 | UpdateTenantQuotaError400 | UpdateTenantQuotaError403;

model GetAdminSettingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminSettingsResponse;
}

union GetAdminSettingsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminSettingsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminSettingsError403Body;
}

union GetAdminSettingsError404Body {
  IdMagic.Contract.TenantNotFoundError,
}

model GetAdminSettingsError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminSettingsError404Body;
}

@doc("テナントの管理者が、自身のテナントの設定（表示名、デフォルトのロケール、パスワードポリシーの上書き）を読む。対象は解決したテナントである `context.tenant_id` に固定されるので、テナントを横断した読み取りは構造上起こらない。system_admin だけが使える `/api/admin/tenants` とは別の経路で、admin にも開いている。レスポンスは、Authentication モジュールの PasswordPolicy の現在の値（`password_policy_defaults`）と、カタログが翻訳を提供するロケール（`supported_locales`）を含むので、UI は何も上書きしないときに何が適用されるか、どのデフォルトのロケールを選べるかを具体的に示せる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetAdminSettings")
@route("/api/admin/v1/settings")
@get
op GetAdminSettings(

): GetAdminSettingsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminSettingsError403 | GetAdminSettingsError404;

model GetAdminIntegrationEndpointsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminIntegrationEndpointCatalog;
}

union GetAdminIntegrationEndpointsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminIntegrationEndpointsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminIntegrationEndpointsError403Body;
}

union GetAdminIntegrationEndpointsError503Body {
  IdMagic.Contract.FederationCredentialsUnavailableError,
}

model GetAdminIntegrationEndpointsError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GetAdminIntegrationEndpointsError503Body;
}

@doc("テナントの管理者が、外部の OIDC の RP、SAML の SP、WS-Federation の RP、SCIM のクライアント、管理用のクライアントに設定する IdMagic の正規の連携情報を、一回の呼び出しで読む。サーバーはすべての URL を、解決したリクエストのテナントの正規の発行者から導くので、フロントエンドはレルムもオリジンも連結しない。シークレットは含まず、`Cache-Control: no-store` で返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetAdminIntegrationEndpoints")
@route("/api/admin/v1/integration-endpoints")
@get
op GetAdminIntegrationEndpoints(

): GetAdminIntegrationEndpointsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminIntegrationEndpointsError403 | GetAdminIntegrationEndpointsError503;

model UpdateAdminSettingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminSettingsResponse;
}

union UpdateAdminSettingsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAdminSettingsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminSettingsError400Body;
}

union UpdateAdminSettingsError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminSettingsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminSettingsError403Body;
}

@doc("テナントの管理者が、自身のテナントの設定を更新する。`display_name`、`default_locale`、`password_policy_override` を任意のフィールドとして受け取り、対象を `context.tenant_id` に固定してから UpdateTenant の処理を実行する。Authentication モジュールの PasswordPolicy の標準値より弱い `password_policy_override`（より小さい `min_length`、より大きい `max_length`、より小さい `history_depth`）と、システムの上限である 30 から 3650 日の範囲の外の `max_age_days` は、どちらもユースケースで拒否する。`default_locale` はカタログが翻訳を提供するロケールだけを受け付け、空文字列はシステムのデフォルトへ戻す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminSettings")
@route("/api/admin/v1/settings")
@patch
op UpdateAdminSettings(
  @body body: IdMagic.Contract.AdminSettingsUpdateRequest
): UpdateAdminSettingsSuccess_200 | UpdateAdminSettingsError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAdminSettingsError403;

model GetTenantUserAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantUserAttributeSchemaResponse;
}

union GetTenantUserAttributeSchemaError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantUserAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantUserAttributeSchemaError403Body;
}

@doc("テナントの管理者が、自身のテナントのカスタム属性の定義を読む。対象は `context.tenant_id` に固定され、テナントを横断した読み取りは構造上起こらない。レスポンスはテナント自身のカスタムの定義に加えて組み込みの一覧（`builtin`）を含むので、UI はすべての属性を並べ、キーの衝突を避けられる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetTenantUserAttributeSchema")
@route("/api/admin/v1/tenant/user-attribute-schema")
@get
op GetTenantUserAttributeSchema(

): GetTenantUserAttributeSchemaSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetTenantUserAttributeSchemaError403;

model UpdateTenantUserAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantUserAttributeSchemaResponse;
}

union UpdateTenantUserAttributeSchemaError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantUserAttributeSchemaError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantUserAttributeSchemaError400Body;
}

union UpdateTenantUserAttributeSchemaError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantUserAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantUserAttributeSchemaError403Body;
}

union UpdateTenantUserAttributeSchemaError422Body {
  IdMagic.Contract.InvalidUserAttributeSchemaError,
}

model UpdateTenantUserAttributeSchemaError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantUserAttributeSchemaError422Body;
}

union UpdateTenantUserAttributeSchemaError409Body {
  IdMagic.Contract.AttributeReferencedByDynamicGroupError,
}

model UpdateTenantUserAttributeSchemaError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantUserAttributeSchemaError409Body;
}

@doc("テナントの管理者が、カスタム属性の定義を丸ごと置き換える。対象を `context.tenant_id` に固定して各 UserAttributeDef を検証し、組み込みの属性と衝突するキーと重複するキーを拒否する。DynamicGroupRule がまだ参照しているキーは、削除も型の変更もできない。扱うのは定義だけで、値は保持しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantUserAttributeSchema")
@route("/api/admin/v1/tenant/user-attribute-schema")
@put
op UpdateTenantUserAttributeSchema(
  @body body: IdMagic.Contract.TenantUserAttributeSchemaUpdateRequest
): UpdateTenantUserAttributeSchemaSuccess_200 | UpdateTenantUserAttributeSchemaError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateTenantUserAttributeSchemaError403 | UpdateTenantUserAttributeSchemaError409 | UpdateTenantUserAttributeSchemaError422;

model GetTenantGroupAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantGroupAttributeSchemaResponse;
}

union GetTenantGroupAttributeSchemaError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantGroupAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantGroupAttributeSchemaError403Body;
}

@doc("テナントの管理者が、自身のテナントの Group のカスタム属性の定義を読む。対象は `context.tenant_id` に固定され、テナントを横断した読み取りは構造上起こらない。Group には組み込みの一覧がないので、レスポンスはテナント自身のカスタムの定義だけを含む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetTenantGroupAttributeSchema")
@route("/api/admin/v1/tenant/group-attribute-schema")
@get
op GetTenantGroupAttributeSchema(

): GetTenantGroupAttributeSchemaSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetTenantGroupAttributeSchemaError403;

model UpdateTenantGroupAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantGroupAttributeSchemaResponse;
}

union UpdateTenantGroupAttributeSchemaError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantGroupAttributeSchemaError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantGroupAttributeSchemaError400Body;
}

union UpdateTenantGroupAttributeSchemaError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantGroupAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantGroupAttributeSchemaError403Body;
}

union UpdateTenantGroupAttributeSchemaError422Body {
  IdMagic.Contract.InvalidGroupAttributeSchemaError,
}

model UpdateTenantGroupAttributeSchemaError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantGroupAttributeSchemaError422Body;
}

@doc("テナントの管理者が、Group のカスタム属性の定義を丸ごと置き換える。対象を `context.tenant_id` に固定して各 GroupAttributeDef を検証し、重複するキーを拒否する。扱うのは定義だけで、値は保持しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantGroupAttributeSchema")
@route("/api/admin/v1/tenant/group-attribute-schema")
@put
op UpdateTenantGroupAttributeSchema(
  @body body: IdMagic.Contract.TenantGroupAttributeSchemaUpdateRequest
): UpdateTenantGroupAttributeSchemaSuccess_200 | UpdateTenantGroupAttributeSchemaError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateTenantGroupAttributeSchemaError403 | UpdateTenantGroupAttributeSchemaError422;

model GetTenantBrandingSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingResponse;
}

model GetTenantBrandingSuccess_304 {
  @statusCode statusCode: 304;
}

@doc("解決したテナントの、ホストする UI のブランディングを返す公開インターフェース。サインイン、同意、デバイスの画面は誰も認証していない時点でこれを読むので、認証を求めない。未設定、不正、またはアセットのないブランディングは、それぞれシステムのデフォルトへ戻すので、呼び出しは常に成功する。レルムの解決（ResolveTenant）で対象のテナントがすでに決まっているので、入力はとらない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetTenantBranding")
@route("/api/branding")
@get
op GetTenantBranding(

): GetTenantBrandingSuccess_200 | GetTenantBrandingSuccess_304;

model UpdateTenantBrandingSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingResponse;
}

union UpdateTenantBrandingError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateTenantBrandingError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantBrandingError400Body;
}

union UpdateTenantBrandingError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantBrandingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantBrandingError403Body;
}

@doc("テナントの管理者が、自身のテナントのブランディング（製品名、色、決まった順序のフッターのリンク、フッターのテキスト）を更新する。画像のアセットは別のインターフェースで扱う。色は `#rrggbb` の形式だけを検証し、コントラスト比を理由に拒否することはない。フッターのリンクはそれぞれラベルと https の URL の組で、片方だけを与えると拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateTenantBranding")
@route("/api/admin/v1/tenant/branding")
@put
op UpdateTenantBranding(
  @body body: IdMagic.Contract.TenantBrandingUpdateRequest
): UpdateTenantBrandingSuccess_200 | UpdateTenantBrandingError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateTenantBrandingError403;

model UploadTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingUploadResponse;
}

union UploadTenantBrandingAssetError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UploadTenantBrandingAssetError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UploadTenantBrandingAssetError400Body;
}

union UploadTenantBrandingAssetError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UploadTenantBrandingAssetError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UploadTenantBrandingAssetError403Body;
}

@doc("管理者が、ブランディングのロゴまたはファビコンを multipart でアップロードする。PNG、JPEG、WebP、GIF を 256 KiB まで受け付け、形式はマジックバイトで確かめる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UploadTenantBrandingAsset")
@route("/api/admin/v1/tenant/branding/assets/{kind}")
@post
op UploadTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind,
  @header contentType: "multipart/form-data",
  @multipartBody body: {
    @doc("保存する画像。フォームのフィールド名は `file` である。")
    file: HttpPart<File>,
  }
): UploadTenantBrandingAssetSuccess_200 | UploadTenantBrandingAssetError400 | IdMagic.Contract.AuthenticationRequiredResponse | UploadTenantBrandingAssetError403;

model DeleteTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TenantBrandingUploadResponse;
}

union DeleteTenantBrandingAssetError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteTenantBrandingAssetError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteTenantBrandingAssetError400Body;
}

union DeleteTenantBrandingAssetError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteTenantBrandingAssetError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteTenantBrandingAssetError403Body;
}

@doc("管理者が、保存したブランディングのロゴまたはファビコンを削除する。以降はそのアセットの URL を返さない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("DeleteTenantBrandingAsset")
@route("/api/admin/v1/tenant/branding/assets/{kind}")
@delete
op DeleteTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind
): DeleteTenantBrandingAssetSuccess_200 | DeleteTenantBrandingAssetError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteTenantBrandingAssetError403;

model ListNotificationTemplatesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateListResponse;
}

union ListNotificationTemplatesError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListNotificationTemplatesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListNotificationTemplatesError403Body;
}

@doc("テナントの管理者が通知のテンプレートを一覧する。カタログのすべての template_key を、対応するすべてのロケールについて、カスタマイズ済みかどうかと現在有効な件名とともに返す。対象は `context.tenant_id` に固定されるので、テナントを横断した読み取りは構造上起こらない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("ListNotificationTemplates")
@route("/api/admin/v1/tenant/notification-templates")
@get
op ListNotificationTemplates(

): ListNotificationTemplatesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListNotificationTemplatesError403;

model GetNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateDetail;
}

union GetNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetNotificationTemplateError400Body;
}

union GetNotificationTemplateError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetNotificationTemplateError403Body;
}

@doc("テナントの管理者が、編集のためにテンプレートを一つ読む。現在有効な文面、組み込みのデフォルトの文面、許可する置換変数の集合をまとめて返す。カタログが知らない template_key またはロケールは InvalidRequestError で拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("GetNotificationTemplate")
@route("/api/admin/v1/tenant/notification-templates/{template_key}/{locale}")
@get
op GetNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string
): GetNotificationTemplateSuccess_200 | GetNotificationTemplateError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetNotificationTemplateError403;

model UpdateNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateDetail;
}

union UpdateNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateNotificationTemplateError400Body;
}

union UpdateNotificationTemplateError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateNotificationTemplateError403Body;
}

@doc("テナントの管理者が、テンプレートの上書きを保存する。件名、テキストの本文、HTML の本文を一組の必須の集合として置き換えるので、テキストだけ、HTML だけの上書きは作れない。本文または件名にある、template_key の許可する集合の外の置換変数は、実行時に空文字列へ潰れる前に、保存時に拒否する（フェイルクローズ）。送信者のメールアドレスはサーバーの設定で上書きできず、上書きできるのは表示名だけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("UpdateNotificationTemplate")
@route("/api/admin/v1/tenant/notification-templates/{template_key}/{locale}")
@put
op UpdateNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,
  @body body: IdMagic.Contract.NotificationTemplateUpdateRequest
): UpdateNotificationTemplateSuccess_200 | UpdateNotificationTemplateError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateNotificationTemplateError403;

model ResetNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateDetail;
}

union ResetNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ResetNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ResetNotificationTemplateError400Body;
}

union ResetNotificationTemplateError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ResetNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResetNotificationTemplateError403Body;
}

@doc("テナントの管理者が、テンプレートの上書きを削除し、組み込みのデフォルトへ戻す。上書きのないテンプレートに対しても成功するので冪等である。バージョンの履歴はないので、戻る先は常に組み込みのデフォルトで、以前の上書きではない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("ResetNotificationTemplate")
@route("/api/admin/v1/tenant/notification-templates/{template_key}/{locale}")
@delete
op ResetNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string
): ResetNotificationTemplateSuccess_200 | ResetNotificationTemplateError400 | IdMagic.Contract.AuthenticationRequiredResponse | ResetNotificationTemplateError403;

model PreviewNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplatePreviewResponse;
}

union PreviewNotificationTemplateError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model PreviewNotificationTemplateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: PreviewNotificationTemplateError400Body;
}

union PreviewNotificationTemplateError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model PreviewNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PreviewNotificationTemplateError403Body;
}

@doc("テナントの管理者が、保存していない文面をサンプルの値で描画する。省いたフィールドは現在有効な文面を使う。HTML 側の置換値は展開の前に必ずエスケープし、リンクの URL はサンプルの発行者の URL から組み立てる。メールは送らず、上書きも保存しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("PreviewNotificationTemplate")
@route("/api/admin/v1/tenant/notification-templates/{template_key}/{locale}/preview")
@post
op PreviewNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,
  @body body: IdMagic.Contract.NotificationTemplatePreviewRequest
): PreviewNotificationTemplateSuccess_200 | PreviewNotificationTemplateError400 | IdMagic.Contract.AuthenticationRequiredResponse | PreviewNotificationTemplateError403;

model SendTestNotificationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.NotificationTemplateTestSendResponse;
}

union SendTestNotificationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SendTestNotificationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SendTestNotificationError400Body;
}

union SendTestNotificationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model SendTestNotificationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SendTestNotificationError403Body;
}

@doc("テナントの管理者が、現在有効な文面でテストメッセージを受け取る。宛先は管理者自身の確認済みのメールアドレスに固定し、リクエストで指定することはできない。任意の宛先を許すと、管理者の権限がメール送信の中継になるので、意図して固定している。呼び出し元に確認済みのアドレスがないときは、InvalidRequestError でリクエストを拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:write"])
@TypeSpec.OpenAPI.operationId("SendTestNotification")
@route("/api/admin/v1/tenant/notification-templates/{template_key}/{locale}/test")
@post
op SendTestNotification(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,): SendTestNotificationSuccess_200 | SendTestNotificationError400 | IdMagic.Contract.AuthenticationRequiredResponse | SendTestNotificationError403;

model GetTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @doc("保存したアセットの、検証済みの形式に固定する。`X-Content-Type-Options: nosniff` を付けて返す。")
  @header contentType: "image/png" | "image/jpeg" | "image/webp" | "image/gif";
  @doc("アセットのバイト列。")
  @body body: bytes;
}

union GetTenantBrandingAssetError404Body {
  IdMagic.Contract.TenantBrandingAssetNotFoundError,
}

model GetTenantBrandingAssetError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetTenantBrandingAssetError404Body;
}

@doc("保存したブランディングのアセット（ロゴまたはファビコン）を返す公開インターフェース。アップロード後に返す URL は、解決したレルムの接頭辞を含めて同じオリジンでこのインターフェースに届くので、管理コンソールのプレビューとホストする UI のどちらも画像として取得して描画できる。Content-Type は検証済みの形式に固定し、`X-Content-Type-Options: nosniff` を設定する。他のテナントのオブジェクトと削除済みのオブジェクトは、どちらも存在しないものとして扱う。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetTenantBrandingAsset")
@route("/tenant-branding-assets/{kind}/{id}")
@get
op GetTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind,
  @path id: string
): GetTenantBrandingAssetSuccess_200 | GetTenantBrandingAssetError404;
}

}
