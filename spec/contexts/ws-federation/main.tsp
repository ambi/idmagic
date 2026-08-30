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

@doc("The single HTTP endpoint of the WS-Federation passive requestor profile. `wa` dispatches between sign-in and sign-out. Sign-in resolves the relying party from `wtrealm` and confines `wreply` to the allowed set, so it cannot become an open redirect. An unauthenticated caller is sent to sign in; an authenticated one gets the claims built by the relying party's claim policy, carried in a signed SAML assertion (SAML 1.1 by default, which Entra expects; SAML 2.0 when the RP is configured for it), wrapped in an RSTR and auto-posted. When the authentication is older than the maximum age `wfresh` states, the caller is sent to reauthenticate, and the authentication method `wauth` asks for is honoured. The meaning of sign-out belongs to WsFederationSignOut.")
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

@doc("The WS-Trust 1.3 active STS Issue endpoint. It verifies the UsernameToken, confines `To` to the active STS endpoint of the current realm, `Action` and `RequestType` to Issue, and `KeyType` to Bearer, and resolves `AppliesTo` to a registered WsFedRelyingParty. The claims built by the RP's claim policy are carried in a signed SAML assertion and returned as an RSTR. An unregistered `AppliesTo`, an expired Timestamp, and a malformed `To`, `Action`, `RequestType`, `KeyType`, or `TokenType` are all rejected fail-closed.")
@TypeSpec.OpenAPI.operationId("WsTrustIssue")
@route("/trust/usernamemixed")
@post
op WsTrustIssue(
  @body body: IdMagic.Contract.WsTrustRequestSecurityToken
): WsTrustIssueSuccess_200 | WsTrustIssueError400 | IdMagic.Contract.AuthenticationRequiredResponse | WsTrustIssueError401 | WsTrustIssueError403;

model WsTrustMetadataExchangeSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/xml; charset=utf-8";
  @doc("The WS-MetadataExchange document, as `mex:Metadata` XML.")
  @body body: string;
}

@doc("The WS-Trust MEX endpoint. It advertises the usernamemixed endpoint of the active STS, the UsernameToken policy, and the WS-Addressing and SOAP bindings, all derived from the realm issuer.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("WsTrustMetadataExchange")
@route("/trust/mex")
@get
op WsTrustMetadataExchange(

): WsTrustMetadataExchangeSuccess_200;

model PublishWsFederationMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/xml; charset=utf-8";
  @doc("The AD FS compatible `federationmetadata.xml` document, as `EntityDescriptor` XML.")
  @body body: string;
}

@doc("WS-Federation and WS-Trust metadata publication. It publishes the realm entityID, the passive endpoint, the active STS endpoint, the MEX endpoint, and the tenant's XmlFederationSigning certificate as an AD FS compatible `federationmetadata.xml`. Alongside the active certificate it lists any verifying certificate still inside its rotation grace.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishWsFederationMetadata")
@route("/federationmetadata/2007-06/federationmetadata.xml")
@get
op PublishWsFederationMetadata(

): PublishWsFederationMetadataSuccess_200;

model ConfigureEntraFederationHttpRequest {
  @doc("A DNS domain already verified in Entra.")
  domain: string;
  @doc("The IssuerUri. Left empty, it is derived from the domain, stably.")
  issuer_uri?: string;
  @doc("The immutable user attribute the ImmutableID is built from.")
  source_anchor_attribute: string;
  @doc("The URL Entra uses as `wreply`. Left empty, the default Microsoft Online URL is used.")
  reply_url?: string;
}

model ConfigureEntraFederationHttpResponse {
  profile: IdMagic.Contract.EntraFederationProfile;
  relying_party: IdMagic.Contract.WsFedRelyingParty;
  @doc("The values to feed into `Set-MsolDomainAuthentication`, keyed by its parameter names: IssuerUri, PassiveLogOnUri, ActiveLogOnUri, MetadataExchangeUri, and SigningCertificate.")
  powershell: Record<string>;
  @doc("What this preset does not provide, stated so the operator finds out before configuring rather than after. Hybrid Azure AD Join device registration is listed here.")
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
  IdMagic.Contract.AccessDeniedError,
}

model ConfigureEntraFederationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfigureEntraFederationError403Body;
}

@doc("Configures the Microsoft Entra domain federation preset. It takes the verified domain, the IssuerUri, and the sourceAnchor attribute, and checks that sourceAnchor is neither missing nor duplicated across the existing users and that it normalizes to a GUID or a base64 ImmutableID. On success it upserts a WsFedRelyingParty carrying the claim preset for UPN, ImmutableID, and a persistent NameID, and returns the PassiveLogOnUri, ActiveLogOnUri, and MetadataExchangeUri to register in Entra. Hybrid Azure AD Join device registration needs windowstransport and computer-account Kerberos, so the diagnostics report it as not offered.")
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
  IdMagic.Contract.AccessDeniedError,
}

model RegisterWsFedRelyingPartyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterWsFedRelyingPartyError403Body;
}

@doc("An administrator registers or updates a WS-Federation relying party trust. `wtrealm`, `reply_urls`, `audience`, `token_type`, and `claim_policy` are stored as the contract, and `reply_urls` is a non-empty closed set. The operation is guarded by AdminFederationTrustsManage.")
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
  IdMagic.Contract.AccessDeniedError,
}

model ListWsFedRelyingPartiesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListWsFedRelyingPartiesError403Body;
}

@doc("An administrator lists the WS-Federation relying party trusts of their tenant. The operation is guarded by AdminFederationTrustsManage.")
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

@doc("An administrator deletes, by `wtrealm`, a WS-Federation relying party trust of their tenant that sits outside the catalog. An RP whose `application_id` is not null is refused with ApplicationOwnedProtocolError (HTTP 409).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("DeleteWsFedRelyingParty")
@route("/api/admin/v1/wsfed/relying-parties")
@delete
op DeleteWsFedRelyingParty(
  @query wtrealm: string
): DeleteWsFedRelyingPartySuccess_204 | DeleteWsFedRelyingPartyError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteWsFedRelyingPartyError403 | DeleteWsFedRelyingPartyError409;
}

}
