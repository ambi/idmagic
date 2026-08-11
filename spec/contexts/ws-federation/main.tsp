import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace IdMagic.WsFederation {

namespace Operations {
model WsFederationSignInSuccess_200 {
  @statusCode statusCode: 200;
}

model WsFederationSignInSuccess_303 {
  @statusCode statusCode: 303;
}

@doc("WS-Federation passive requestor profile の単一 HTTP endpoint。wa で sign-in と sign-out を dispatch する。sign-in は wtrealm で relying party を解決し、wreply を許可集合に限定する (open redirect 防止)。未認証はログインへ誘導し、認証済みなら relying party の claim policy で発行した claim を署名済み SAML assertion (既定 SAML 1.1 / Entra 互換、RP 設定で SAML 2.0) に載せ、RSTR に包んで自動 POST する。wfresh が示す最大経過時間より認証が古ければ再認証へ誘導し、wauth が要求する認証方式を尊重する。sign-out の意味契約は WsFederationSignOut が所有する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("WsFederationSignIn")
@route("/wsfed")
@get
op WsFederationSignIn(
  @query wa: string,
  @query wtrealm: string,
  @query wreply?: string,
  @query wctx?: string,
  @query wfresh?: string,
  @query wauth?: string,
  @query whr?: string
): WsFederationSignInSuccess_200 | WsFederationSignInSuccess_303;

model WsTrustIssueHttpRequest {
  request: IdMagic.Contract.WsTrustRequestSecurityToken;
}

model WsTrustIssueHttpResponse {
  response: IdMagic.Contract.WsTrustResponse;
}

model WsTrustIssueSuccess_200 {
  @statusCode statusCode: 200;
  @body body: WsTrustIssueHttpResponse;
}

union WsTrustIssueError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model WsTrustIssueError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: WsTrustIssueError400Body;
}

union WsTrustIssueError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model WsTrustIssueError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: WsTrustIssueError403Body;
}

@doc("WS-Trust 1.3 active STS Issue endpoint。UsernameToken を検証し、 To を現在 realm の active STS endpoint、Action / RequestType を Issue、KeyType を Bearer に限定し、AppliesTo を登録済み WsFedRelyingParty に解決する。RP の claim policy で組み立てた claim を署名済み SAML assertion に載せ、RSTR として返す。未登録 AppliesTo、期限切れ Timestamp、不正な To / Action / RequestType / KeyType / TokenType は fail-closed で拒否する。")
@TypeSpec.OpenAPI.operationId("WsTrustIssue")
@route("/trust/usernamemixed")
@post
op WsTrustIssue(
  @body body: WsTrustIssueHttpRequest
): WsTrustIssueSuccess_200 | WsTrustIssueError400 | WsTrustIssueError403;

model WsTrustMetadataExchangeHttpResponse {
  endpoint: IdMagic.Contract.SecurityTokenEndpoint;
}

model WsTrustMetadataExchangeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: WsTrustMetadataExchangeHttpResponse;
}

@doc("WS-Trust MEX endpoint。active STS の usernamemixed endpoint、UsernameToken policy、WS-Addressing / SOAP binding を realm issuer から派生して広告する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("WsTrustMetadataExchange")
@route("/trust/mex")
@get
op WsTrustMetadataExchange(

): WsTrustMetadataExchangeSuccess_200;

model PublishWsFederationMetadataHttpResponse {
  endpoint: IdMagic.Contract.SecurityTokenEndpoint;
}

model PublishWsFederationMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PublishWsFederationMetadataHttpResponse;
}

@doc("WS-Federation / WS-Trust metadata publication。realm entityID、passive endpoint、 active STS endpoint、MEX endpoint、tenant の XmlFederationSigning 証明書を AD FS 互換 federationmetadata.xml として公開する。active 証明書に加え rotation grace 中の verifying 証明書も掲載する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishWsFederationMetadata")
@route("/federationmetadata/2007-06/federationmetadata.xml")
@get
op PublishWsFederationMetadata(

): PublishWsFederationMetadataSuccess_200;

model ConfigureEntraFederationHttpRequest {
  @doc("Entra で検証済みの DNS domain。")
  domain: string;
  @doc("空なら domain から安定生成する IssuerUri。")
  issuer_uri?: string;
  @doc("ImmutableID の元になる不変 user 属性。")
  source_anchor_attribute: string;
  @doc("Entra が wreply に使う URL。空なら Microsoft Online の既定 URL。")
  reply_url?: string;
}

model ConfigureEntraFederationHttpResponse {
  profile: IdMagic.Contract.EntraFederationProfile;
  relying_party: IdMagic.Contract.WsFedRelyingParty;
}

model ConfigureEntraFederationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ConfigureEntraFederationHttpResponse;
}

union ConfigureEntraFederationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ConfigureEntraFederationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ConfigureEntraFederationError400Body;
}

union ConfigureEntraFederationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ConfigureEntraFederationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfigureEntraFederationError403Body;
}

@doc("Microsoft Entra domain federation preset を設定する。検証済み domain、 IssuerUri、sourceAnchor 属性を受け取り、sourceAnchor が既存 user で欠落・重複しないこと、 GUID または base64 ImmutableID として正規化できることを検証する。成功時は UPN / ImmutableID / persistent NameID の claim preset を持つ WsFedRelyingParty を upsert し、Entra に登録する PassiveLogOnUri / ActiveLogOnUri / MetadataExchangeUri を返す。 Hybrid Azure AD Join device registration は windowstransport + コンピュータアカウント Kerberos を要するため未提供として診断に含める。")
@TypeSpec.OpenAPI.operationId("ConfigureEntraFederation")
@route("/api/admin/v1/wsfed/entra-federation")
@post
op ConfigureEntraFederation(
  @body body: ConfigureEntraFederationHttpRequest
): ConfigureEntraFederationSuccess_200 | ConfigureEntraFederationError400 | ConfigureEntraFederationError403;

model RegisterWsFedRelyingPartyHttpRequest {
  relying_party: IdMagic.Contract.WsFedRelyingParty;
}

model RegisterWsFedRelyingPartyHttpResponse {
  relying_party: IdMagic.Contract.WsFedRelyingParty;
}

model RegisterWsFedRelyingPartySuccess_201 {
  @statusCode statusCode: 201;
  @body body: RegisterWsFedRelyingPartyHttpResponse;
}

model RegisterWsFedRelyingPartySuccess_200 {
  @statusCode statusCode: 200;
  @body body: RegisterWsFedRelyingPartyHttpResponse;
}

union RegisterWsFedRelyingPartyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterWsFedRelyingPartyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterWsFedRelyingPartyError400Body;
}

union RegisterWsFedRelyingPartyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RegisterWsFedRelyingPartyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterWsFedRelyingPartyError403Body;
}

@doc("管理者が WS-Federation relying party trust を登録または更新する。wtrealm、reply_urls、audience、token_type、claim_policy を契約として保存し、reply_urls は空でない閉集合にする。管理操作は AdminFederationTrustsManage で保護する。")
@TypeSpec.OpenAPI.operationId("RegisterWsFedRelyingParty")
@route("/api/admin/v1/wsfed/relying-parties")
@post
op RegisterWsFedRelyingParty(
  @body body: RegisterWsFedRelyingPartyHttpRequest
): RegisterWsFedRelyingPartySuccess_201 | RegisterWsFedRelyingPartySuccess_200 | RegisterWsFedRelyingPartyError400 | RegisterWsFedRelyingPartyError403;

model ListWsFedRelyingPartiesHttpResponse {
  relying_parties: IdMagic.Contract.WsFedRelyingParty[];
}

model ListWsFedRelyingPartiesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListWsFedRelyingPartiesHttpResponse;
}

union ListWsFedRelyingPartiesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListWsFedRelyingPartiesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListWsFedRelyingPartiesError403Body;
}

@doc("管理者が所属テナントの WS-Federation relying party trust を一覧する。管理操作は AdminFederationTrustsManage で保護する。")
@TypeSpec.OpenAPI.operationId("ListWsFedRelyingParties")
@route("/api/admin/v1/wsfed/relying-parties")
@get
op ListWsFedRelyingParties(

): ListWsFedRelyingPartiesSuccess_200 | ListWsFedRelyingPartiesError403;

model DeleteWsFedRelyingPartySuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteWsFedRelyingPartyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteWsFedRelyingPartyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteWsFedRelyingPartyError400Body;
}

union DeleteWsFedRelyingPartyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteWsFedRelyingPartyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteWsFedRelyingPartyError403Body;
}

union DeleteWsFedRelyingPartyError409Body {
  IdMagic.Contract.ApplicationOwnedProtocolError,
}

model DeleteWsFedRelyingPartyError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteWsFedRelyingPartyError409Body;
}

@doc("管理者が所属テナントの catalog 外 WS-Federation relying party trust を wtrealm で削除する。application_id が非 NULL の RP は ApplicationOwnedProtocolError (HTTP 409) で拒否する。")
@TypeSpec.OpenAPI.operationId("DeleteWsFedRelyingParty")
@route("/api/admin/v1/wsfed/relying-parties")
@delete
op DeleteWsFedRelyingParty(
  @query wtrealm: string
): DeleteWsFedRelyingPartySuccess_204 | DeleteWsFedRelyingPartyError400 | DeleteWsFedRelyingPartyError403 | DeleteWsFedRelyingPartyError409;
}

}
