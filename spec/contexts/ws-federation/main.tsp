import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("WS-Federation")
namespace IdMagic.WsFederation {

namespace Operations {
model WsFederationSignInSuccess_200 {
  @statusCode statusCode: 200;
}

model WsFederationSignInSuccess_303 {
  @statusCode statusCode: 303;
}

model WsFederationSignInError400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

model WsFederationSignInError403 {
  @statusCode statusCode: 403;
  @header contentType: "text/plain";
  @body body: string;
}

@doc("WS-Federation のパッシブリクエスターのプロファイルの唯一の HTTP エンドポイント。`wa` でサインインとサインアウトを振り分ける。サインインは `wtrealm` から Relying Party を解決し、`wreply` を許可した集合に限るので、オープンリダイレクトにはならない。未認証の呼び出し元はサインインへ送る。認証済みの呼び出し元には、Relying Party のクレームのポリシーが組み立てたクレームを、署名した SAML アサーション（デフォルトは Entra が求める SAML 1.1、RP が設定していれば SAML 2.0）に収め、RSTR で包んで自動 POST する。認証が `wfresh` の示す最大の経過時間より古いときは呼び出し元を再認証へ送り、`wauth` が求める認証方式に従う。サインアウトの意味は WsFederationSignOut が定める。")
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
): WsFederationSignInSuccess_200 | WsFederationSignInSuccess_303 | WsFederationSignInError400 | WsFederationSignInError403;

model WsTrustIssueSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WsTrustResponse;
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

model WsTrustIssueError401 {
  @statusCode statusCode: 401;
  @header contentType: "text/plain";
  @body body: string;
}

@doc("WS-Trust 1.3 のアクティブ STS の Issue エンドポイント。UsernameToken を検証し、`To` を現在のレルムのアクティブ STS のエンドポイントに、`Action` と `RequestType` を Issue に、`KeyType` を Bearer に限り、`AppliesTo` を登録済みの WsFedRelyingParty へ解決する。RP のクレームのポリシーが組み立てたクレームを署名した SAML アサーションに収め、RSTR として返す。未登録の `AppliesTo`、期限切れの Timestamp、形式の誤った `To`、`Action`、`RequestType`、`KeyType`、`TokenType` は、いずれもフェイルクローズで拒否する。")
@TypeSpec.OpenAPI.operationId("WsTrustIssue")
@route("/trust/usernamemixed")
@post
op WsTrustIssue(
  @body body: IdMagic.Contract.WsTrustRequestSecurityToken
): WsTrustIssueSuccess_200 | WsTrustIssueError400 | IdMagic.Contract.AuthenticationRequiredResponse | WsTrustIssueError401 | WsTrustIssueError403;

model WsTrustMetadataExchangeSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/xml; charset=utf-8";
  @doc("WS-MetadataExchange の文書。`mex:Metadata` の XML で返す。")
  @body body: string;
}

@doc("WS-Trust の MEX エンドポイント。アクティブ STS の usernamemixed のエンドポイント、UsernameToken のポリシー、WS-Addressing と SOAP のバインディングを示し、いずれもレルムの発行者から導く。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("WsTrustMetadataExchange")
@route("/trust/mex")
@get
op WsTrustMetadataExchange(

): WsTrustMetadataExchangeSuccess_200;

model PublishWsFederationMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/xml; charset=utf-8";
  @doc("AD FS 互換の `federationmetadata.xml` の文書。`EntityDescriptor` の XML で返す。")
  @body body: string;
}

@doc("WS-Federation と WS-Trust のメタデータの公開。レルムの entityID、パッシブのエンドポイント、アクティブ STS のエンドポイント、MEX のエンドポイント、テナントの XmlFederationSigning の証明書を、AD FS 互換の `federationmetadata.xml` として公開する。有効な証明書に加えて、ローテーションの猶予の内にある verifying の証明書も載せる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishWsFederationMetadata")
@route("/federationmetadata/2007-06/federationmetadata.xml")
@get
op PublishWsFederationMetadata(

): PublishWsFederationMetadataSuccess_200;

model ConfigureEntraFederationHttpRequest {
  @doc("Entra で確認済みの DNS ドメイン。")
  domain: string;
  @doc("IssuerUri。空のままにすると、ドメインから安定して導く。")
  issuer_uri?: string;
  @doc("ImmutableID の元になる、変わらないユーザーの属性。")
  source_anchor_attribute: string;
  @doc("Entra が `wreply` として使う URL。空のままにすると、Microsoft Online のデフォルトの URL を使う。")
  reply_url?: string;
}

model ConfigureEntraFederationHttpResponse {
  profile: IdMagic.Contract.EntraFederationProfile;
  relying_party: IdMagic.Contract.WsFedRelyingParty;
  @doc("`Set-MsolDomainAuthentication` に渡す値。そのパラメーター名（IssuerUri、PassiveLogOnUri、ActiveLogOnUri、MetadataExchangeUri、SigningCertificate）をキーとする。")
  powershell: Record<string>;
  @doc("このプリセットが提供しないもの。運用者が設定した後ではなく前に気付けるように示す。Hybrid Azure AD Join のデバイス登録はここに載る。")
  known_limitations: string[];
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ConfigureEntraFederationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfigureEntraFederationError403Body;
}

@doc("Microsoft Entra のドメインフェデレーションのプリセットを設定する。確認済みのドメイン、IssuerUri、sourceAnchor の属性を受け取り、sourceAnchor が既存のユーザーで欠けても重複してもいないこと、GUID または base64 の ImmutableID へ正規化できることを確かめる。成功すると、UPN、ImmutableID、永続的な NameID のクレームのプリセットを含む WsFedRelyingParty を作成または更新し、Entra に登録する PassiveLogOnUri、ActiveLogOnUri、MetadataExchangeUri を返す。Hybrid Azure AD Join のデバイス登録には windowstransport とコンピューターアカウントの Kerberos が必要なので、診断はそれを提供しないものとして報告する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("ConfigureEntraFederation")
@route("/api/admin/v1/wsfed/entra-federation")
@post
op ConfigureEntraFederation(
  @body body: ConfigureEntraFederationHttpRequest
): ConfigureEntraFederationSuccess_200 | ConfigureEntraFederationError400 | IdMagic.Contract.AuthenticationRequiredResponse | ConfigureEntraFederationError403;

model RegisterWsFedRelyingPartySuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.WsFedRelyingPartyRequest;
}

model RegisterWsFedRelyingPartySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WsFedRelyingParty;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model RegisterWsFedRelyingPartyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterWsFedRelyingPartyError403Body;
}

@doc("管理者が、WS-Federation の Relying Party の信頼を登録または更新する。`wtrealm`、`reply_urls`、`audience`、`token_type`、`claim_policy` を契約として保存し、`reply_urls` は空でない閉じた集合である。この操作は AdminFederationTrustsManage が守る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("RegisterWsFedRelyingParty")
@route("/api/admin/v1/wsfed/relying-parties")
@post
op RegisterWsFedRelyingParty(
  @body body: IdMagic.Contract.WsFedRelyingPartyRequest
): RegisterWsFedRelyingPartySuccess_201 | RegisterWsFedRelyingPartySuccess_200 | RegisterWsFedRelyingPartyError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterWsFedRelyingPartyError403;

model ListWsFedRelyingPartiesHttpResponse {
  relying_parties: IdMagic.Contract.WsFedRelyingParty[];
}

model ListWsFedRelyingPartiesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListWsFedRelyingPartiesHttpResponse;
}

union ListWsFedRelyingPartiesError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListWsFedRelyingPartiesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListWsFedRelyingPartiesError403Body;
}

@doc("管理者が、自身のテナントの WS-Federation の Relying Party の信頼を一覧する。この操作は AdminFederationTrustsManage が守る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:read"])
@TypeSpec.OpenAPI.operationId("ListWsFedRelyingParties")
@route("/api/admin/v1/wsfed/relying-parties")
@get
op ListWsFedRelyingParties(

): ListWsFedRelyingPartiesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListWsFedRelyingPartiesError403;

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
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、自身のテナントの、カタログの外にある WS-Federation の Relying Party の信頼を `wtrealm` で削除する。`application_id` が null でない RP は ApplicationOwnedProtocolError（HTTP 409）で拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("DeleteWsFedRelyingParty")
@route("/api/admin/v1/wsfed/relying-parties")
@delete
op DeleteWsFedRelyingParty(
  @query wtrealm: string
): DeleteWsFedRelyingPartySuccess_204 | DeleteWsFedRelyingPartyError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteWsFedRelyingPartyError403 | DeleteWsFedRelyingPartyError409;
}

}
