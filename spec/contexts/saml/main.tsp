import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace IdMagic.Saml {

namespace Operations {
model SamlSingleSignOn1Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn1Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML 2.0 Web Browser SSO の sign-in。SP-initiated は AuthnRequest を HTTP-Redirect / HTTP-POST binding で受信し、Version=2.0、IssueInstant の許容窓、Issuer、Destination、HTTP-POST response ProtocolBinding、ACS URL/index の排他と対応可否、NameIDPolicy format を検証してから service provider を解決する。routeのIdP profile、SPのprofile binding、Destinationが一致しなければfail-closedで拒否する。署名必須 SP では Redirect binding 署名または XML 署名を検証する。未対応または矛盾する要求は fail-closed に拒否し、返信可能な検証済み ACS がある場合だけ HTTP-POST の SAML protocol error を返す。ForceAuthn=true は直近再認証済みでなければログインへ誘導する。IsPassive=true で既存セッションを使えない場合はログインへ遷移せず NoPassive response を返す。IdP-initiated は entityID クエリで対象 SP を選ぶ。未認証はログインへ誘導し、認証往復をまたいで要求を保つ。認証済みなら application 割当ゲートを fail-closed で適用し、未割当 subject には発行しない。SP の claim policy で発行した NameID と attribute を bearer SAML assertion に載せ、割当profileのXML署名資格情報とSP設定に従って assertion / response を署名し、SAMLResponse を ACS へ HTTP-POST で自動送信する。tenant / SP / AuthnRequest ID は assertion 発行直前に TTL replay store へ原子的に一度だけ記録する。InResponseTo・AudienceRestriction・Recipient・Destination を整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleSignOn")
@route("/saml/sso")
@get
op SamlSingleSignOn1(
  @query saml_request?: string,
  @query entity_id?: string,
  @query relay_state?: string,
  @query profile_id: string = "default"
): SamlSingleSignOn1Success_200 | SamlSingleSignOn1Success_303;

model SamlSingleSignOn2Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn2Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML 2.0 Web Browser SSO の sign-in。SP-initiated は AuthnRequest を HTTP-Redirect / HTTP-POST binding で受信し、Version=2.0、IssueInstant の許容窓、Issuer、Destination、HTTP-POST response ProtocolBinding、ACS URL/index の排他と対応可否、NameIDPolicy format を検証してから service provider を解決する。routeのIdP profile、SPのprofile binding、Destinationが一致しなければfail-closedで拒否する。署名必須 SP では Redirect binding 署名または XML 署名を検証する。未対応または矛盾する要求は fail-closed に拒否し、返信可能な検証済み ACS がある場合だけ HTTP-POST の SAML protocol error を返す。ForceAuthn=true は直近再認証済みでなければログインへ誘導する。IsPassive=true で既存セッションを使えない場合はログインへ遷移せず NoPassive response を返す。IdP-initiated は entityID クエリで対象 SP を選ぶ。未認証はログインへ誘導し、認証往復をまたいで要求を保つ。認証済みなら application 割当ゲートを fail-closed で適用し、未割当 subject には発行しない。SP の claim policy で発行した NameID と attribute を bearer SAML assertion に載せ、割当profileのXML署名資格情報とSP設定に従って assertion / response を署名し、SAMLResponse を ACS へ HTTP-POST で自動送信する。tenant / SP / AuthnRequest ID は assertion 発行直前に TTL replay store へ原子的に一度だけ記録する。InResponseTo・AudienceRestriction・Recipient・Destination を整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleSignOn")
@route("/saml/idp/{profile_id}/sso")
@get
op SamlSingleSignOn2(
  @query saml_request?: string,
  @query entity_id?: string,
  @query relay_state?: string,
  @path profile_id: string
): SamlSingleSignOn2Success_200 | SamlSingleSignOn2Success_303;

model SamlSingleSignOn3HttpRequest {
  @doc("SP-initiated の AuthnRequest (binding に応じ deflate+base64 / base64)。IdP-initiated では空。")
  saml_request?: string;
  @doc("IdP-initiated で対象 SP を指定する entityID。")
  entity_id?: string;
  @doc("SP が往復させる不透明コンテキスト。SAMLResponse に同値で返す。")
  relay_state?: string;
  @doc("request routeが選択したIdP profile ID。")
  profile_id: string = "default";
}

model SamlSingleSignOn3Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn3Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML 2.0 Web Browser SSO の sign-in。SP-initiated は AuthnRequest を HTTP-Redirect / HTTP-POST binding で受信し、Version=2.0、IssueInstant の許容窓、Issuer、Destination、HTTP-POST response ProtocolBinding、ACS URL/index の排他と対応可否、NameIDPolicy format を検証してから service provider を解決する。routeのIdP profile、SPのprofile binding、Destinationが一致しなければfail-closedで拒否する。署名必須 SP では Redirect binding 署名または XML 署名を検証する。未対応または矛盾する要求は fail-closed に拒否し、返信可能な検証済み ACS がある場合だけ HTTP-POST の SAML protocol error を返す。ForceAuthn=true は直近再認証済みでなければログインへ誘導する。IsPassive=true で既存セッションを使えない場合はログインへ遷移せず NoPassive response を返す。IdP-initiated は entityID クエリで対象 SP を選ぶ。未認証はログインへ誘導し、認証往復をまたいで要求を保つ。認証済みなら application 割当ゲートを fail-closed で適用し、未割当 subject には発行しない。SP の claim policy で発行した NameID と attribute を bearer SAML assertion に載せ、割当profileのXML署名資格情報とSP設定に従って assertion / response を署名し、SAMLResponse を ACS へ HTTP-POST で自動送信する。tenant / SP / AuthnRequest ID は assertion 発行直前に TTL replay store へ原子的に一度だけ記録する。InResponseTo・AudienceRestriction・Recipient・Destination を整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleSignOn")
@route("/saml/idp/{profile_id}/sso")
@post
op SamlSingleSignOn3(
  @path profile_id: string,
  @body body: SamlSingleSignOn3HttpRequest
): SamlSingleSignOn3Success_200 | SamlSingleSignOn3Success_303;

model SamlSingleSignOn4HttpRequest {
  @doc("SP-initiated の AuthnRequest (binding に応じ deflate+base64 / base64)。IdP-initiated では空。")
  saml_request?: string;
  @doc("IdP-initiated で対象 SP を指定する entityID。")
  entity_id?: string;
  @doc("SP が往復させる不透明コンテキスト。SAMLResponse に同値で返す。")
  relay_state?: string;
  @doc("request routeが選択したIdP profile ID。")
  profile_id: string = "default";
}

model SamlSingleSignOn4Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn4Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML 2.0 Web Browser SSO の sign-in。SP-initiated は AuthnRequest を HTTP-Redirect / HTTP-POST binding で受信し、Version=2.0、IssueInstant の許容窓、Issuer、Destination、HTTP-POST response ProtocolBinding、ACS URL/index の排他と対応可否、NameIDPolicy format を検証してから service provider を解決する。routeのIdP profile、SPのprofile binding、Destinationが一致しなければfail-closedで拒否する。署名必須 SP では Redirect binding 署名または XML 署名を検証する。未対応または矛盾する要求は fail-closed に拒否し、返信可能な検証済み ACS がある場合だけ HTTP-POST の SAML protocol error を返す。ForceAuthn=true は直近再認証済みでなければログインへ誘導する。IsPassive=true で既存セッションを使えない場合はログインへ遷移せず NoPassive response を返す。IdP-initiated は entityID クエリで対象 SP を選ぶ。未認証はログインへ誘導し、認証往復をまたいで要求を保つ。認証済みなら application 割当ゲートを fail-closed で適用し、未割当 subject には発行しない。SP の claim policy で発行した NameID と attribute を bearer SAML assertion に載せ、割当profileのXML署名資格情報とSP設定に従って assertion / response を署名し、SAMLResponse を ACS へ HTTP-POST で自動送信する。tenant / SP / AuthnRequest ID は assertion 発行直前に TTL replay store へ原子的に一度だけ記録する。InResponseTo・AudienceRestriction・Recipient・Destination を整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleSignOn")
@route("/saml/sso")
@post
op SamlSingleSignOn4(
  @body body: SamlSingleSignOn4HttpRequest
): SamlSingleSignOn4Success_200 | SamlSingleSignOn4Success_303;

model SamlSingleLogout1Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout1Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML Single Logout。LogoutRequest を HTTP-Redirect / HTTP-POST binding で受信し、Issuer で SP を解決し、routeのIdP profileとSPのprofile bindingを照合し、署名必須 SP では署名を検証する。Destination は選択profileの SLO endpoint、返送先は登録済み SingleLogoutService に限定する。ローカルセッションを破棄し、割当profileの鍵で署名した LogoutResponse を登録済み SLO URL へ返す。entityID / sp 指定のローカル logout も受理するが、判定不能・未登録の返送先へはリダイレクトしない (open redirect 防止)。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleLogout")
@route("/saml/slo")
@get
op SamlSingleLogout1(
  @query entity_id?: string,
  @query relay_state?: string,
  @query profile_id: string = "default"
): SamlSingleLogout1Success_200 | SamlSingleLogout1Success_303;

model SamlSingleLogout2HttpRequest {
  @doc("logout を要求した SP の entityID。")
  entity_id?: string;
  relay_state?: string;
  @doc("request routeが選択したIdP profile ID。")
  profile_id: string = "default";
}

model SamlSingleLogout2Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout2Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML Single Logout。LogoutRequest を HTTP-Redirect / HTTP-POST binding で受信し、Issuer で SP を解決し、routeのIdP profileとSPのprofile bindingを照合し、署名必須 SP では署名を検証する。Destination は選択profileの SLO endpoint、返送先は登録済み SingleLogoutService に限定する。ローカルセッションを破棄し、割当profileの鍵で署名した LogoutResponse を登録済み SLO URL へ返す。entityID / sp 指定のローカル logout も受理するが、判定不能・未登録の返送先へはリダイレクトしない (open redirect 防止)。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleLogout")
@route("/saml/slo")
@post
op SamlSingleLogout2(
  @body body: SamlSingleLogout2HttpRequest
): SamlSingleLogout2Success_200 | SamlSingleLogout2Success_303;

model SamlSingleLogout3Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout3Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML Single Logout。LogoutRequest を HTTP-Redirect / HTTP-POST binding で受信し、Issuer で SP を解決し、routeのIdP profileとSPのprofile bindingを照合し、署名必須 SP では署名を検証する。Destination は選択profileの SLO endpoint、返送先は登録済み SingleLogoutService に限定する。ローカルセッションを破棄し、割当profileの鍵で署名した LogoutResponse を登録済み SLO URL へ返す。entityID / sp 指定のローカル logout も受理するが、判定不能・未登録の返送先へはリダイレクトしない (open redirect 防止)。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleLogout")
@route("/saml/idp/{profile_id}/slo")
@get
op SamlSingleLogout3(
  @query entity_id?: string,
  @query relay_state?: string,
  @path profile_id: string
): SamlSingleLogout3Success_200 | SamlSingleLogout3Success_303;

model SamlSingleLogout4HttpRequest {
  @doc("logout を要求した SP の entityID。")
  entity_id?: string;
  relay_state?: string;
  @doc("request routeが選択したIdP profile ID。")
  profile_id: string = "default";
}

model SamlSingleLogout4Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout4Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML Single Logout。LogoutRequest を HTTP-Redirect / HTTP-POST binding で受信し、Issuer で SP を解決し、routeのIdP profileとSPのprofile bindingを照合し、署名必須 SP では署名を検証する。Destination は選択profileの SLO endpoint、返送先は登録済み SingleLogoutService に限定する。ローカルセッションを破棄し、割当profileの鍵で署名した LogoutResponse を登録済み SLO URL へ返す。entityID / sp 指定のローカル logout も受理するが、判定不能・未登録の返送先へはリダイレクトしない (open redirect 防止)。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleLogout")
@route("/saml/idp/{profile_id}/slo")
@post
op SamlSingleLogout4(
  @path profile_id: string,
  @body body: SamlSingleLogout4HttpRequest
): SamlSingleLogout4Success_200 | SamlSingleLogout4Success_303;

model PublishSamlMetadata1Success_200 {
  @statusCode statusCode: 200;
}

@doc("SAML 2.0 IdP metadata の公開。default routeはrealmのdefault profile、profile routeは指定profileのentityID、SSO / SLO endpoint、対応 NameID format、profile scopeのXmlFederationSigning証明書を <EntityDescriptor><IDPSSODescriptor> として公開する。active 証明書に加え rotation grace 中の verifying 証明書も掲載する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishSamlMetadata")
@route("/saml/metadata")
@get
op PublishSamlMetadata1(

): PublishSamlMetadata1Success_200;

model PublishSamlMetadata2Success_200 {
  @statusCode statusCode: 200;
}

@doc("SAML 2.0 IdP metadata の公開。default routeはrealmのdefault profile、profile routeは指定profileのentityID、SSO / SLO endpoint、対応 NameID format、profile scopeのXmlFederationSigning証明書を <EntityDescriptor><IDPSSODescriptor> として公開する。active 証明書に加え rotation grace 中の verifying 証明書も掲載する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishSamlMetadata")
@route("/saml/idp/{profile_id}/metadata")
@get
op PublishSamlMetadata2(
  @path profile_id: string
): PublishSamlMetadata2Success_200;

model DownloadSamlSigningCertificate1Success_200 {
  @statusCode statusCode: 200;
}

@doc("SAML SP への手動 trust 設定用に、request tenant の active XmlFederationSigning X.509 証明書を PEM で公開する。rotation overlap 中の verifying 証明書を含む trust 正本は PublishSamlMetadata とし、このエンドポイントは新規設定用 active 証明書だけを返す。秘密鍵は返さない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DownloadSamlSigningCertificate")
@route("/saml/signing-certificate.pem")
@get
op DownloadSamlSigningCertificate1(

): DownloadSamlSigningCertificate1Success_200;

model DownloadSamlSigningCertificate2Success_200 {
  @statusCode statusCode: 200;
}

@doc("SAML SP への手動 trust 設定用に、request tenant の active XmlFederationSigning X.509 証明書を PEM で公開する。rotation overlap 中の verifying 証明書を含む trust 正本は PublishSamlMetadata とし、このエンドポイントは新規設定用 active 証明書だけを返す。秘密鍵は返さない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DownloadSamlSigningCertificate")
@route("/saml/idp/{profile_id}/signing-certificate.pem")
@get
op DownloadSamlSigningCertificate2(
  @path profile_id: string
): DownloadSamlSigningCertificate2Success_200;

model CreateSamlIdentityProviderProfileHttpRequest {
  profile: IdMagic.Contract.SamlIdentityProviderProfile;
}

model CreateSamlIdentityProviderProfileHttpResponse {
  profile: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse;
}

model CreateSamlIdentityProviderProfileSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateSamlIdentityProviderProfileHttpResponse;
}

union CreateSamlIdentityProviderProfileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateSamlIdentityProviderProfileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateSamlIdentityProviderProfileError400Body;
}

union CreateSamlIdentityProviderProfileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateSamlIdentityProviderProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateSamlIdentityProviderProfileError403Body;
}

@doc("テナント管理者が追加のsharedまたはdedicated SAML IdP profileを作成する。default profileはこの操作では作成できない。")
@TypeSpec.OpenAPI.operationId("CreateSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles")
@post
op CreateSamlIdentityProviderProfile(
  @body body: CreateSamlIdentityProviderProfileHttpRequest
): CreateSamlIdentityProviderProfileSuccess_201 | CreateSamlIdentityProviderProfileError400 | CreateSamlIdentityProviderProfileError403;

model ListSamlIdentityProviderProfilesHttpResponse {
  profiles: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse[];
}

model ListSamlIdentityProviderProfilesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSamlIdentityProviderProfilesHttpResponse;
}

union ListSamlIdentityProviderProfilesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListSamlIdentityProviderProfilesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSamlIdentityProviderProfilesError403Body;
}

@doc("テナント内のSAML IdP profileをdefault先頭、name昇順で一覧する。")
@TypeSpec.OpenAPI.operationId("ListSamlIdentityProviderProfiles")
@route("/api/admin/v1/saml/idp-profiles")
@get
op ListSamlIdentityProviderProfiles(

): ListSamlIdentityProviderProfilesSuccess_200 | ListSamlIdentityProviderProfilesError403;

model UpdateSamlIdentityProviderProfileHttpRequest {
  profile: IdMagic.Contract.SamlIdentityProviderProfile;
}

model UpdateSamlIdentityProviderProfileHttpResponse {
  profile: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse;
}

model UpdateSamlIdentityProviderProfileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateSamlIdentityProviderProfileHttpResponse;
}

union UpdateSamlIdentityProviderProfileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateSamlIdentityProviderProfileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateSamlIdentityProviderProfileError400Body;
}

union UpdateSamlIdentityProviderProfileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateSamlIdentityProviderProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateSamlIdentityProviderProfileError403Body;
}

union UpdateSamlIdentityProviderProfileError409Body {
  IdMagic.Contract.DefaultIdPProfileError,
}

model UpdateSamlIdentityProviderProfileError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: UpdateSamlIdentityProviderProfileError409Body;
}

@doc("テナント管理者が追加IdP profileのnameまたはmodeを変更する。default profileは変更できず、dedicatedへの変更時に複数SPから参照されていれば拒否する。")
@TypeSpec.OpenAPI.operationId("UpdateSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles/{profile_id}")
@put
op UpdateSamlIdentityProviderProfile(
  @path profile_id: string,
  @body body: UpdateSamlIdentityProviderProfileHttpRequest
): UpdateSamlIdentityProviderProfileSuccess_200 | UpdateSamlIdentityProviderProfileError400 | UpdateSamlIdentityProviderProfileError403 | UpdateSamlIdentityProviderProfileError409;

model DeleteSamlIdentityProviderProfileSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteSamlIdentityProviderProfileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteSamlIdentityProviderProfileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlIdentityProviderProfileError400Body;
}

union DeleteSamlIdentityProviderProfileError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteSamlIdentityProviderProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlIdentityProviderProfileError403Body;
}

union DeleteSamlIdentityProviderProfileError409Body {
  IdMagic.Contract.DefaultIdPProfileError,
  IdMagic.Contract.IdPProfileInUseError,
}

model DeleteSamlIdentityProviderProfileError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlIdentityProviderProfileError409Body;
}

@doc("参照されていない追加IdP profileを削除する。default profileとSPが参照中のprofileは拒否する。")
@TypeSpec.OpenAPI.operationId("DeleteSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles/{profile_id}")
@delete
op DeleteSamlIdentityProviderProfile(
  @path profile_id: string
): DeleteSamlIdentityProviderProfileSuccess_204 | DeleteSamlIdentityProviderProfileError400 | DeleteSamlIdentityProviderProfileError403 | DeleteSamlIdentityProviderProfileError409;

model RegisterSamlServiceProviderHttpRequest {
  service_provider: IdMagic.Contract.SamlServiceProvider;
}

model RegisterSamlServiceProviderHttpResponse {
  service_provider: IdMagic.Contract.SamlServiceProvider;
}

model RegisterSamlServiceProviderSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RegisterSamlServiceProviderHttpResponse;
}

model RegisterSamlServiceProviderSuccess_201 {
  @statusCode statusCode: 201;
  @body body: RegisterSamlServiceProviderHttpResponse;
}

union RegisterSamlServiceProviderError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterSamlServiceProviderError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterSamlServiceProviderError400Body;
}

union RegisterSamlServiceProviderError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RegisterSamlServiceProviderError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterSamlServiceProviderError403Body;
}

@doc("SAML service provider の登録 / 更新 (upsert)。entityID、許可 ACS URL、NameID format、audience、署名方針、claim policy を受け取り、テナント境界に閉じて保存する。entityID と ACS URL の閉集合は必須。")
@TypeSpec.OpenAPI.operationId("RegisterSamlServiceProvider")
@route("/api/admin/v1/saml/service-providers")
@post
op RegisterSamlServiceProvider(
  @body body: RegisterSamlServiceProviderHttpRequest
): RegisterSamlServiceProviderSuccess_200 | RegisterSamlServiceProviderSuccess_201 | RegisterSamlServiceProviderError400 | RegisterSamlServiceProviderError403;

model ListSamlServiceProvidersHttpResponse {
  service_providers: IdMagic.Contract.SamlServiceProvider[];
}

model ListSamlServiceProvidersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSamlServiceProvidersHttpResponse;
}

union ListSamlServiceProvidersError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListSamlServiceProvidersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSamlServiceProvidersError403Body;
}

@doc("テナント内の SAML service provider を entityID 昇順で一覧する。")
@TypeSpec.OpenAPI.operationId("ListSamlServiceProviders")
@route("/api/admin/v1/saml/service-providers")
@get
op ListSamlServiceProviders(

): ListSamlServiceProvidersSuccess_200 | ListSamlServiceProvidersError403;

model DeleteSamlServiceProviderSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteSamlServiceProviderError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteSamlServiceProviderError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlServiceProviderError400Body;
}

union DeleteSamlServiceProviderError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteSamlServiceProviderError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlServiceProviderError403Body;
}

union DeleteSamlServiceProviderError409Body {
  IdMagic.Contract.ApplicationOwnedProtocolError,
}

model DeleteSamlServiceProviderError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlServiceProviderError409Body;
}

@doc("entityID に一致する catalog 外 SAML service provider を削除する (冪等)。application_id が非 NULL の SP は ApplicationOwnedProtocolError (HTTP 409) で拒否する。")
@TypeSpec.OpenAPI.operationId("DeleteSamlServiceProvider")
@route("/api/admin/v1/saml/service-providers")
@delete
op DeleteSamlServiceProvider(
  @query entity_id: string
): DeleteSamlServiceProviderSuccess_204 | DeleteSamlServiceProviderError400 | DeleteSamlServiceProviderError403 | DeleteSamlServiceProviderError409;
}

}
