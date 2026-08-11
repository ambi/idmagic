import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Tenancy")
namespace IdMagic.Tenancy {

namespace Operations {
model ListTenantsHttpResponse {
  response: IdMagic.Contract.TenantListResponse;
}

model ListTenantsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListTenantsHttpResponse;
}

union ListTenantsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantsError403Body;
}

@doc("system_admin がテナント一覧を取得する。テナント横断操作のため system_admin\nロールでゲートし、システムコンソール (/system) から呼ぶ。default テナントに解決する\nbare path と control-plane path (/realms/default 配下、cookie session 整合) の両方から\n同じ handler に入る。")
@TypeSpec.OpenAPI.operationId("ListTenants")
@route("/api/admin/v1/tenants")
@get
op ListTenants(

): ListTenantsSuccess_200 | ListTenantsError403;

model GetTenantHttpResponse {
  tenant: IdMagic.Contract.TenantSummaryResponse;
}

model GetTenantSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetTenantHttpResponse;
}

union GetTenantError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetTenantError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetTenantError400Body;
}

union GetTenantError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantError403Body;
}

@doc("system_admin が単一テナントを取得する。")
@TypeSpec.OpenAPI.operationId("GetTenant")
@route("/api/admin/v1/tenants/{tenant_id}")
@get
op GetTenant(
  @path tenant_id: string
): GetTenantSuccess_200 | GetTenantError400 | GetTenantError403;

model CreateTenantHttpRequest {
  request: IdMagic.Contract.TenantCreateRequest;
}

model CreateTenantHttpResponse {
  tenant: IdMagic.Contract.TenantSummaryResponse;
}

model CreateTenantSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateTenantHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model CreateTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateTenantError403Body;
}

@doc("system_admin が新規テナントを作成する。")
@TypeSpec.OpenAPI.operationId("CreateTenant")
@route("/api/admin/v1/tenants")
@post
op CreateTenant(
  @body body: CreateTenantHttpRequest
): CreateTenantSuccess_201 | CreateTenantError400 | CreateTenantError403;

model UpdateTenantHttpRequest {
  tenant_id: string;
  request: IdMagic.Contract.TenantUpdateRequest;
}

model UpdateTenantHttpResponse {
  tenant: IdMagic.Contract.TenantSummaryResponse;
}

model UpdateTenantSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateTenantHttpResponse;
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
@TypeSpec.OpenAPI.operationId("UpdateTenant")
@route("/api/admin/v1/tenants/{tenant_id}")
@patch
op UpdateTenant(
  @path tenant_id: string,
  @body body: UpdateTenantHttpRequest
): UpdateTenantSuccess_200 | UpdateTenantError400 | UpdateTenantError403 | UpdateTenantError422;

model DisableTenantHttpRequest {
  tenant_id: string;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model DisableTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableTenantError403Body;
}

@doc("system_admin が default 以外のテナントを無効化する。/authorize / /token / /login 等は一律 invalid_request になる。")
@TypeSpec.OpenAPI.operationId("DisableTenant")
@route("/api/admin/v1/tenants/{tenant_id}/disable")
@post
op DisableTenant(
  @path tenant_id: string,
  @body body: DisableTenantHttpRequest
): DisableTenantSuccess_204 | DisableTenantError400 | DisableTenantError403;

model EnableTenantHttpRequest {
  tenant_id: string;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model EnableTenantError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableTenantError403Body;
}

@doc("system_admin が無効化済みテナントを再有効化する。")
@TypeSpec.OpenAPI.operationId("EnableTenant")
@route("/api/admin/v1/tenants/{tenant_id}/enable")
@post
op EnableTenant(
  @path tenant_id: string,
  @body body: EnableTenantHttpRequest
): EnableTenantSuccess_204 | EnableTenantError400 | EnableTenantError403;

model SetTenantEndpointStyleHttpRequest {
  tenant_id: string;
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
  IdMagic.Contract.AccessDeniedError,
}

model SetTenantEndpointStyleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SetTenantEndpointStyleError403Body;
}

@doc("system_admin がテナントの正規ロケーションの形を切り替える。\n通常の属性更新 (UpdateTenant) から分離するのは、これがテナントの identity を\n作り替える破壊的操作だからである: issuer が変わるため既発行 token の `iss` 検証と\n全 RP の設定が壊れ、WebAuthn RP ID が変わるため既存 passkey がすべて無効になり、\ncookie scope が変わるため進行中のセッションが切れる。\nSubdomain は tenant_base_domain が設定された配備でのみ選択でき、未設定なら\nInvalidRequestError とする。realm は不変なので、切り替え後のホスト名は既存 realm から\n一意に決まる。")
@TypeSpec.OpenAPI.operationId("SetTenantEndpointStyle")
@route("/api/admin/v1/tenants/{tenant_id}/endpoint_style")
@put
op SetTenantEndpointStyle(
  @path tenant_id: string,
  @body body: SetTenantEndpointStyleHttpRequest
): SetTenantEndpointStyleSuccess_204 | SetTenantEndpointStyleError400 | SetTenantEndpointStyleError403;

model UpdateTenantQuotaHttpRequest {
  tenant_id: string;
  request: IdMagic.Contract.TenantQuotaUpdateRequest;
}

model UpdateTenantQuotaHttpResponse {
  tenant: IdMagic.Contract.TenantSummaryResponse;
}

model UpdateTenantQuotaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateTenantQuotaHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantQuotaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantQuotaError403Body;
}

@doc("system_admin が特定テナントの Quota を更新する。")
@TypeSpec.OpenAPI.operationId("UpdateTenantQuota")
@route("/api/admin/v1/tenants/{tenant_id}/quota")
@put
op UpdateTenantQuota(
  @path tenant_id: string,
  @body body: UpdateTenantQuotaHttpRequest
): UpdateTenantQuotaSuccess_200 | UpdateTenantQuotaError400 | UpdateTenantQuotaError403;

model GetAdminSettingsHttpResponse {
  settings: IdMagic.Contract.AdminSettingsResponse;
}

model GetAdminSettingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminSettingsHttpResponse;
}

union GetAdminSettingsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminSettingsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminSettingsError403Body;
}

@doc("テナント内 admin が自身のテナントの設定 (表示名 / 既定 locale / パスワードポリシー上書き) を取得する。\n対象テナントは context.tenant_id (解決済みテナント) に固定するため cross-tenant 読み出しは構造的に発生しない。\n`/api/admin/tenants` (system_admin 専用) とは別経路で、admin にも開放される。\nレスポンスには Authentication context の PasswordPolicy の現行値 (password_policy_defaults) と、\nカタログが同梱翻訳を持つ locale 一覧 (supported_locales) を含め、UI が「上書きしないときに\n実際に適用される値」と既定 locale の選択肢を具体的に示せるようにする。")
@TypeSpec.OpenAPI.operationId("GetAdminSettings")
@route("/api/admin/v1/settings")
@get
op GetAdminSettings(

): GetAdminSettingsSuccess_200 | GetAdminSettingsError403;

model GetAdminIntegrationEndpointsHttpResponse {
  catalog: IdMagic.Contract.AdminIntegrationEndpointCatalog;
}

model GetAdminIntegrationEndpointsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminIntegrationEndpointsHttpResponse;
}

union GetAdminIntegrationEndpointsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminIntegrationEndpointsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminIntegrationEndpointsError403Body;
}

@doc("テナント内 admin が、外部の OIDC RP / SAML SP / WS-Federation RP / SCIM client / management client に投入する IdMagic の正規連携情報を一括取得する。全 URL は解決済み request tenant の canonical issuer から server が導出し、frontend は realm や origin を連結しない。秘密情報は含めず Cache-Control: no-store で返す。")
@TypeSpec.OpenAPI.operationId("GetAdminIntegrationEndpoints")
@route("/api/admin/v1/integration-endpoints")
@get
op GetAdminIntegrationEndpoints(

): GetAdminIntegrationEndpointsSuccess_200 | GetAdminIntegrationEndpointsError403;

model UpdateAdminSettingsHttpRequest {
  request: IdMagic.Contract.TenantUpdateRequest;
}

model UpdateAdminSettingsHttpResponse {
  settings: IdMagic.Contract.AdminSettingsResponse;
}

model UpdateAdminSettingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAdminSettingsHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminSettingsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminSettingsError403Body;
}

@doc("テナント内 admin が自身のテナントの設定を更新する。display_name、default_locale、\npassword_policy_override のいずれかを任意フィールドとして受け取り、\ncontext.tenant_id に固定した上で UpdateTenant のロジックを実行する。\npassword_policy_override が Authentication context の PasswordPolicy の標準値より弱い場合\n(min_length 下回り / max_length 上回り / history_depth 下回り) は use case\n側で拒否する。default_locale はカタログが同梱翻訳を持つ locale のみ受け付け、\n空文字列でシステム既定へ戻す。")
@TypeSpec.OpenAPI.operationId("UpdateAdminSettings")
@route("/api/admin/v1/settings")
@patch
op UpdateAdminSettings(
  @body body: UpdateAdminSettingsHttpRequest
): UpdateAdminSettingsSuccess_200 | UpdateAdminSettingsError400 | UpdateAdminSettingsError403;

model GetTenantUserAttributeSchemaHttpResponse {
  schema: IdMagic.Contract.TenantUserAttributeSchemaResponse;
}

model GetTenantUserAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetTenantUserAttributeSchemaHttpResponse;
}

union GetTenantUserAttributeSchemaError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetTenantUserAttributeSchemaError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetTenantUserAttributeSchemaError403Body;
}

@doc("テナント内 admin が自身のテナントの custom 属性定義を取得する。\n対象テナントは context.tenant_id に固定し cross-tenant 読み出しを構造的に排除する。\nレスポンスには tenant 固有の custom 定義に加えて組み込みカタログ (builtin) も\n含め、UI が全属性を一覧しつつ key 衝突を避けられるようにする。")
@TypeSpec.OpenAPI.operationId("GetTenantUserAttributeSchema")
@route("/api/admin/v1/tenant/user_attribute_schema")
@get
op GetTenantUserAttributeSchema(

): GetTenantUserAttributeSchemaSuccess_200 | GetTenantUserAttributeSchemaError403;

model UpdateTenantUserAttributeSchemaHttpRequest {
  request: IdMagic.Contract.TenantUserAttributeSchemaUpdateRequest;
}

model UpdateTenantUserAttributeSchemaHttpResponse {
  schema: IdMagic.Contract.TenantUserAttributeSchemaResponse;
}

model UpdateTenantUserAttributeSchemaSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateTenantUserAttributeSchemaHttpResponse;
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

@doc("テナント内 admin が custom 属性定義を全置換する。context.tenant_id に\n固定した上で、各 UserAttributeDef を検証し、組み込み属性との key 衝突・重複 key を\n拒否する。DynamicGroupRule が参照中の key は削除または型変更できない。値そのものは保持せず定義のみを扱う。")
@TypeSpec.OpenAPI.operationId("UpdateTenantUserAttributeSchema")
@route("/api/admin/v1/tenant/user_attribute_schema")
@put
op UpdateTenantUserAttributeSchema(
  @body body: UpdateTenantUserAttributeSchemaHttpRequest
): UpdateTenantUserAttributeSchemaSuccess_200 | UpdateTenantUserAttributeSchemaError400 | UpdateTenantUserAttributeSchemaError403 | UpdateTenantUserAttributeSchemaError422;

model GetTenantBrandingHttpResponse {
  branding: IdMagic.Contract.TenantBrandingResponse;
}

model GetTenantBrandingSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetTenantBrandingHttpResponse;
}

@doc("解決済みテナントの hosted UI ブランディングを取得する公開 interface。認証を要求しない\n(login / consent / device 画面が未認証のうちに読む)。branding 未設定・値不正・アセット欠損の\nいずれでもシステム既定にフォールバックし、常に成功する。対象テナントは realm 解決 (ResolveTenant)\nで既に確定しているため入力を取らない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetTenantBranding")
@route("/api/branding")
@get
op GetTenantBranding(

): GetTenantBrandingSuccess_200;

model UpdateTenantBrandingHttpRequest {
  request: IdMagic.Contract.TenantBrandingUpdateRequest;
}

model UpdateTenantBrandingHttpResponse {
  branding: IdMagic.Contract.TenantBrandingResponse;
}

model UpdateTenantBrandingSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateTenantBrandingHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateTenantBrandingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateTenantBrandingError403Body;
}

@doc("テナント内 admin が自身のテナントの branding (製品名 / 色 / 順序固定の footer リンク /\nフッターテキスト) を更新する。画像アセットは別 interface で扱う。色は `#rrggbb` 形式のみを\n検証し、コントラスト比を理由に拒否しない。各 footer link は label と https URL の組で、どちらか\n一方だけの指定は拒否する。")
@TypeSpec.OpenAPI.operationId("UpdateTenantBranding")
@route("/api/admin/v1/tenant/branding")
@put
op UpdateTenantBranding(
  @body body: UpdateTenantBrandingHttpRequest
): UpdateTenantBrandingSuccess_200 | UpdateTenantBrandingError400 | UpdateTenantBrandingError403;

model UploadTenantBrandingAssetHttpRequest {
  kind: IdMagic.Contract.TenantBrandingAssetKind;
  file: string;
}

model UploadTenantBrandingAssetHttpResponse {
  response: IdMagic.Contract.TenantBrandingUploadResponse;
}

model UploadTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UploadTenantBrandingAssetHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UploadTenantBrandingAssetError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UploadTenantBrandingAssetError403Body;
}

@doc("管理者が branding のロゴまたは favicon 画像を multipart でアップロードする。受理形式は PNG / JPEG / WebP / GIF、最大 256KiB とし、magic byte で検証する。")
@TypeSpec.OpenAPI.operationId("UploadTenantBrandingAsset")
@route("/api/admin/v1/tenant/branding/assets/{kind}")
@post
op UploadTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind,
  @body body: UploadTenantBrandingAssetHttpRequest
): UploadTenantBrandingAssetSuccess_200 | UploadTenantBrandingAssetError400 | UploadTenantBrandingAssetError403;

model DeleteTenantBrandingAssetHttpResponse {
  response: IdMagic.Contract.TenantBrandingUploadResponse;
}

model DeleteTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DeleteTenantBrandingAssetHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model DeleteTenantBrandingAssetError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteTenantBrandingAssetError403Body;
}

@doc("管理者が branding の保存済みロゴまたは favicon を削除する。削除後は該当アセットの URL を返さない。")
@TypeSpec.OpenAPI.operationId("DeleteTenantBrandingAsset")
@route("/api/admin/v1/tenant/branding/assets/{kind}")
@delete
op DeleteTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind
): DeleteTenantBrandingAssetSuccess_200 | DeleteTenantBrandingAssetError400 | DeleteTenantBrandingAssetError403;

model ListNotificationTemplatesHttpResponse {
  response: IdMagic.Contract.NotificationTemplateListResponse;
}

model ListNotificationTemplatesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListNotificationTemplatesHttpResponse;
}

union ListNotificationTemplatesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListNotificationTemplatesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListNotificationTemplatesError403Body;
}

@doc("テナント内 admin が通知テンプレートの一覧を取得する。カタログの全 template_key × 全サポート locale を、上書きの有無 (customized) と現在有効な件名つきで返す。対象テナントは context.tenant_id に固定するため cross-tenant 読み出しは構造的に発生しない。")
@TypeSpec.OpenAPI.operationId("ListNotificationTemplates")
@route("/api/admin/v1/tenant/notification_templates")
@get
op ListNotificationTemplates(

): ListNotificationTemplatesSuccess_200 | ListNotificationTemplatesError403;

model GetNotificationTemplateHttpResponse {
  template: IdMagic.Contract.NotificationTemplateDetail;
}

model GetNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetNotificationTemplateHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model GetNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetNotificationTemplateError403Body;
}

@doc("テナント内 admin がテンプレート 1 件を編集用に取得する。現在有効な文面、組込み既定の文面、および使用できる差し込み変数の許可集合を同時に返す。カタログに無い template_key / locale は InvalidRequestError で拒否する。")
@TypeSpec.OpenAPI.operationId("GetNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}")
@get
op GetNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string
): GetNotificationTemplateSuccess_200 | GetNotificationTemplateError400 | GetNotificationTemplateError403;

model UpdateNotificationTemplateHttpRequest {
  template_key: IdMagic.Contract.NotificationTemplateKey;
  locale: string;
  request: IdMagic.Contract.NotificationTemplateUpdateRequest;
}

model UpdateNotificationTemplateHttpResponse {
  template: IdMagic.Contract.NotificationTemplateDetail;
}

model UpdateNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateNotificationTemplateHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateNotificationTemplateError403Body;
}

@doc("テナント内 admin がテンプレート上書きを保存する。件名 / テキスト本文 / HTML 本文を必須の組として全置換し、テキストだけ・HTML だけの上書きは作れない。本文と件名に含まれる差し込み変数が template_key の許可集合に収まらない場合は、実行時に空文字列へ潰れる前に保存時点で拒否する (fail-closed)。差出人メールアドレスはサーバ設定であり上書きできず、表示名だけを上書きできる。")
@TypeSpec.OpenAPI.operationId("UpdateNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}")
@put
op UpdateNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,
  @body body: UpdateNotificationTemplateHttpRequest
): UpdateNotificationTemplateSuccess_200 | UpdateNotificationTemplateError400 | UpdateNotificationTemplateError403;

model ResetNotificationTemplateHttpResponse {
  template: IdMagic.Contract.NotificationTemplateDetail;
}

model ResetNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ResetNotificationTemplateHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model ResetNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResetNotificationTemplateError403Body;
}

@doc("テナント内 admin がテンプレート上書きを削除し組込み既定へ戻す。上書きが無いテンプレートに対しても成功する冪等操作。版管理は持たないため、戻り先は直前の上書きではなく常に組込み既定である。")
@TypeSpec.OpenAPI.operationId("ResetNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}")
@delete
op ResetNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string
): ResetNotificationTemplateSuccess_200 | ResetNotificationTemplateError400 | ResetNotificationTemplateError403;

model PreviewNotificationTemplateHttpRequest {
  template_key: IdMagic.Contract.NotificationTemplateKey;
  locale: string;
  request: IdMagic.Contract.NotificationTemplatePreviewRequest;
}

model PreviewNotificationTemplateHttpResponse {
  preview: IdMagic.Contract.NotificationTemplatePreviewResponse;
}

model PreviewNotificationTemplateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PreviewNotificationTemplateHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model PreviewNotificationTemplateError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PreviewNotificationTemplateError403Body;
}

@doc("テナント内 admin が保存前の文面をサンプル値で描画する。省略したフィールドは現在有効な文面を使う。HTML 側の差し込み値は必ずエスケープして展開し、リンク URL はサンプルの発行元 URL から組み立てる。メールは送信せず、上書きも保存しない。")
@TypeSpec.OpenAPI.operationId("PreviewNotificationTemplate")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}/preview")
@post
op PreviewNotificationTemplate(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,
  @body body: PreviewNotificationTemplateHttpRequest
): PreviewNotificationTemplateSuccess_200 | PreviewNotificationTemplateError400 | PreviewNotificationTemplateError403;

model SendTestNotificationHttpRequest {
  template_key: IdMagic.Contract.NotificationTemplateKey;
  locale: string;
}

model SendTestNotificationHttpResponse {
  result: IdMagic.Contract.NotificationTemplateTestSendResponse;
}

model SendTestNotificationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SendTestNotificationHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model SendTestNotificationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SendTestNotificationError403Body;
}

@doc("テナント内 admin が現在有効な文面のテストメールを受け取る。宛先は操作した管理者本人の検証済みメールアドレスに固定し、リクエストでは指定できない。任意宛先を許すと管理者権限がメール送信の踏み台になるため、この固定は仕様である。操作者に検証済みアドレスが無い場合は InvalidRequestError で拒否する。")
@TypeSpec.OpenAPI.operationId("SendTestNotification")
@route("/api/admin/v1/tenant/notification_templates/{template_key}/{locale}/test")
@post
op SendTestNotification(
  @path template_key: IdMagic.Contract.NotificationTemplateKey,
  @path locale: string,
  @body body: SendTestNotificationHttpRequest
): SendTestNotificationSuccess_200 | SendTestNotificationError400 | SendTestNotificationError403;

model GetTenantBrandingAssetHttpResponse {
  file: string;
}

model GetTenantBrandingAssetSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetTenantBrandingAssetHttpResponse;
}

union GetTenantBrandingAssetError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetTenantBrandingAssetError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetTenantBrandingAssetError400Body;
}

@doc("保存済み branding アセット (ロゴ / favicon) を配信する公開 interface。アップロード後に返す URL は、\n解決済み realm prefix を含むこの interface へ同一 origin で到達可能であり、管理画面プレビューと\nhosted UI はその URL を画像として取得・描画できる。Content-Type は検証済み形式に固定し、\nX-Content-Type-Options: nosniff を付ける。別テナントまたは削除済み object は未存在として扱う。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetTenantBrandingAsset")
@route("/tenant-branding-assets/{kind}/{id}")
@get
op GetTenantBrandingAsset(
  @path kind: IdMagic.Contract.TenantBrandingAssetKind,
  @path id: string
): GetTenantBrandingAssetSuccess_200 | GetTenantBrandingAssetError400;
}

}
