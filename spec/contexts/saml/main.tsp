import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("SAML")
namespace IdMagic.Saml {

namespace Operations {
model SamlSingleSignOn1Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn1Success_303 {
  @statusCode statusCode: 303;
}

@doc("Sign-in for SAML 2.0 Web Browser SSO. An SP-initiated request arrives as an AuthnRequest over the HTTP-Redirect or HTTP-POST binding, and the service provider is resolved only after Version=2.0, the acceptance window around IssueInstant, Issuer, Destination, the HTTP-POST response ProtocolBinding, the mutual exclusion and the supportability of the ACS URL and index, and the NameIDPolicy format have all been checked. When the IdP profile of the route, the profile the SP is bound to, and Destination do not agree, the request is refused fail-closed. For an SP that requires signatures, either the Redirect binding signature or the XML signature is verified. A request that is unsupported or self-contradictory is refused fail-closed, and a SAML protocol error is HTTP-POSTed back only when there is a verified ACS to reply to. ForceAuthn=true sends the caller to sign in unless they reauthenticated recently. With IsPassive=true, an unusable existing session yields a NoPassive response rather than a move to the sign-in page. An IdP-initiated request names the target SP through the entityID query. An unauthenticated caller is sent to sign in, and the request survives the round trip. Once authenticated, the application assignment gate is applied fail-closed and nothing is issued to an unassigned subject. The NameID and attributes the SP's claim policy produces are carried in a bearer SAML assertion; the assertion and the response are signed with the XML signing credentials of the assigned profile and according to the SP's settings; and the SAMLResponse is auto-posted to the ACS over HTTP-POST. The tenant, the SP, and the AuthnRequest ID are recorded in the TTL replay store atomically and exactly once, immediately before the assertion is issued. InResponseTo, AudienceRestriction, Recipient, and Destination are kept consistent with one another.")
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

@doc("Sign-in for SAML 2.0 Web Browser SSO. An SP-initiated request arrives as an AuthnRequest over the HTTP-Redirect or HTTP-POST binding, and the service provider is resolved only after Version=2.0, the acceptance window around IssueInstant, Issuer, Destination, the HTTP-POST response ProtocolBinding, the mutual exclusion and the supportability of the ACS URL and index, and the NameIDPolicy format have all been checked. When the IdP profile of the route, the profile the SP is bound to, and Destination do not agree, the request is refused fail-closed. For an SP that requires signatures, either the Redirect binding signature or the XML signature is verified. A request that is unsupported or self-contradictory is refused fail-closed, and a SAML protocol error is HTTP-POSTed back only when there is a verified ACS to reply to. ForceAuthn=true sends the caller to sign in unless they reauthenticated recently. With IsPassive=true, an unusable existing session yields a NoPassive response rather than a move to the sign-in page. An IdP-initiated request names the target SP through the entityID query. An unauthenticated caller is sent to sign in, and the request survives the round trip. Once authenticated, the application assignment gate is applied fail-closed and nothing is issued to an unassigned subject. The NameID and attributes the SP's claim policy produces are carried in a bearer SAML assertion; the assertion and the response are signed with the XML signing credentials of the assigned profile and according to the SP's settings; and the SAMLResponse is auto-posted to the ACS over HTTP-POST. The tenant, the SP, and the AuthnRequest ID are recorded in the TTL replay store atomically and exactly once, immediately before the assertion is issued. InResponseTo, AudienceRestriction, Recipient, and Destination are kept consistent with one another.")
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
  @doc("The SP-initiated AuthnRequest, deflate+base64 or base64 depending on the binding. Empty for an IdP-initiated request.")
  saml_request?: string;
  @doc("The entityID naming the target SP for an IdP-initiated request.")
  entity_id?: string;
  @doc("The opaque context the SP round-trips. It comes back unchanged in the SAMLResponse.")
  relay_state?: string;
}

model SamlSingleSignOn3Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn3Success_303 {
  @statusCode statusCode: 303;
}

@doc("Sign-in for SAML 2.0 Web Browser SSO. An SP-initiated request arrives as an AuthnRequest over the HTTP-Redirect or HTTP-POST binding, and the service provider is resolved only after Version=2.0, the acceptance window around IssueInstant, Issuer, Destination, the HTTP-POST response ProtocolBinding, the mutual exclusion and the supportability of the ACS URL and index, and the NameIDPolicy format have all been checked. When the IdP profile of the route, the profile the SP is bound to, and Destination do not agree, the request is refused fail-closed. For an SP that requires signatures, either the Redirect binding signature or the XML signature is verified. A request that is unsupported or self-contradictory is refused fail-closed, and a SAML protocol error is HTTP-POSTed back only when there is a verified ACS to reply to. ForceAuthn=true sends the caller to sign in unless they reauthenticated recently. With IsPassive=true, an unusable existing session yields a NoPassive response rather than a move to the sign-in page. An IdP-initiated request names the target SP through the entityID query. An unauthenticated caller is sent to sign in, and the request survives the round trip. Once authenticated, the application assignment gate is applied fail-closed and nothing is issued to an unassigned subject. The NameID and attributes the SP's claim policy produces are carried in a bearer SAML assertion; the assertion and the response are signed with the XML signing credentials of the assigned profile and according to the SP's settings; and the SAMLResponse is auto-posted to the ACS over HTTP-POST. The tenant, the SP, and the AuthnRequest ID are recorded in the TTL replay store atomically and exactly once, immediately before the assertion is issued. InResponseTo, AudienceRestriction, Recipient, and Destination are kept consistent with one another.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleSignOn")
@route("/saml/idp/{profile_id}/sso")
@post
op SamlSingleSignOn3(
  @path profile_id: string,
  @body body: SamlSingleSignOn3HttpRequest
): SamlSingleSignOn3Success_200 | SamlSingleSignOn3Success_303;

model SamlSingleSignOn4HttpRequest {
  @doc("The SP-initiated AuthnRequest, deflate+base64 or base64 depending on the binding. Empty for an IdP-initiated request.")
  saml_request?: string;
  @doc("The entityID naming the target SP for an IdP-initiated request.")
  entity_id?: string;
  @doc("The opaque context the SP round-trips. It comes back unchanged in the SAMLResponse.")
  relay_state?: string;
  @doc("The IdP profile id the request route selected.")
  profile_id: string = "default";
}

model SamlSingleSignOn4Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn4Success_303 {
  @statusCode statusCode: 303;
}

@doc("Sign-in for SAML 2.0 Web Browser SSO. An SP-initiated request arrives as an AuthnRequest over the HTTP-Redirect or HTTP-POST binding, and the service provider is resolved only after Version=2.0, the acceptance window around IssueInstant, Issuer, Destination, the HTTP-POST response ProtocolBinding, the mutual exclusion and the supportability of the ACS URL and index, and the NameIDPolicy format have all been checked. When the IdP profile of the route, the profile the SP is bound to, and Destination do not agree, the request is refused fail-closed. For an SP that requires signatures, either the Redirect binding signature or the XML signature is verified. A request that is unsupported or self-contradictory is refused fail-closed, and a SAML protocol error is HTTP-POSTed back only when there is a verified ACS to reply to. ForceAuthn=true sends the caller to sign in unless they reauthenticated recently. With IsPassive=true, an unusable existing session yields a NoPassive response rather than a move to the sign-in page. An IdP-initiated request names the target SP through the entityID query. An unauthenticated caller is sent to sign in, and the request survives the round trip. Once authenticated, the application assignment gate is applied fail-closed and nothing is issued to an unassigned subject. The NameID and attributes the SP's claim policy produces are carried in a bearer SAML assertion; the assertion and the response are signed with the XML signing credentials of the assigned profile and according to the SP's settings; and the SAMLResponse is auto-posted to the ACS over HTTP-POST. The tenant, the SP, and the AuthnRequest ID are recorded in the TTL replay store atomically and exactly once, immediately before the assertion is issued. InResponseTo, AudienceRestriction, Recipient, and Destination are kept consistent with one another.")
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

@doc("SAML Single Logout. A LogoutRequest arrives over the HTTP-Redirect or HTTP-POST binding, the SP is resolved from Issuer, the IdP profile of the route is matched against the profile the SP is bound to, and for an SP that requires signatures the signature is verified. Destination must be the SLO endpoint of the selected profile, and the reply goes only to a registered SingleLogoutService. The local session is destroyed and a LogoutResponse, signed with the key of the assigned profile, is returned to the registered SLO URL. A local logout naming entityID or sp is accepted too, but nothing is ever redirected to a destination that cannot be resolved or is not registered, so this cannot become an open redirect.")
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
  @doc("The entityID of the SP that asked for the logout.")
  entity_id?: string;
  relay_state?: string;
  @doc("The IdP profile id the request route selected.")
  profile_id: string = "default";
}

model SamlSingleLogout2Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout2Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML Single Logout. A LogoutRequest arrives over the HTTP-Redirect or HTTP-POST binding, the SP is resolved from Issuer, the IdP profile of the route is matched against the profile the SP is bound to, and for an SP that requires signatures the signature is verified. Destination must be the SLO endpoint of the selected profile, and the reply goes only to a registered SingleLogoutService. The local session is destroyed and a LogoutResponse, signed with the key of the assigned profile, is returned to the registered SLO URL. A local logout naming entityID or sp is accepted too, but nothing is ever redirected to a destination that cannot be resolved or is not registered, so this cannot become an open redirect.")
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

@doc("SAML Single Logout. A LogoutRequest arrives over the HTTP-Redirect or HTTP-POST binding, the SP is resolved from Issuer, the IdP profile of the route is matched against the profile the SP is bound to, and for an SP that requires signatures the signature is verified. Destination must be the SLO endpoint of the selected profile, and the reply goes only to a registered SingleLogoutService. The local session is destroyed and a LogoutResponse, signed with the key of the assigned profile, is returned to the registered SLO URL. A local logout naming entityID or sp is accepted too, but nothing is ever redirected to a destination that cannot be resolved or is not registered, so this cannot become an open redirect.")
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
  @doc("The entityID of the SP that asked for the logout.")
  entity_id?: string;
  relay_state?: string;
}

model SamlSingleLogout4Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout4Success_303 {
  @statusCode statusCode: 303;
}

@doc("SAML Single Logout. A LogoutRequest arrives over the HTTP-Redirect or HTTP-POST binding, the SP is resolved from Issuer, the IdP profile of the route is matched against the profile the SP is bound to, and for an SP that requires signatures the signature is verified. Destination must be the SLO endpoint of the selected profile, and the reply goes only to a registered SingleLogoutService. The local session is destroyed and a LogoutResponse, signed with the key of the assigned profile, is returned to the registered SLO URL. A local logout naming entityID or sp is accepted too, but nothing is ever redirected to a destination that cannot be resolved or is not registered, so this cannot become an open redirect.")
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

@doc("Publishes the SAML 2.0 IdP metadata. The default route publishes the realm's default profile and a profile route the named one: its entityID, its SSO and SLO endpoints, the NameID formats it supports, and the XmlFederationSigning certificate of the profile scope, as an <EntityDescriptor><IDPSSODescriptor>. Alongside the active certificate it lists any verifying certificate still inside its rotation grace.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishSamlMetadata")
@route("/saml/metadata")
@get
op PublishSamlMetadata1(

): PublishSamlMetadata1Success_200;

model PublishSamlMetadata2Success_200 {
  @statusCode statusCode: 200;
}

@doc("Publishes the SAML 2.0 IdP metadata. The default route publishes the realm's default profile and a profile route the named one: its entityID, its SSO and SLO endpoints, the NameID formats it supports, and the XmlFederationSigning certificate of the profile scope, as an <EntityDescriptor><IDPSSODescriptor>. Alongside the active certificate it lists any verifying certificate still inside its rotation grace.")
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

@doc("Publishes the request tenant's active XmlFederationSigning X.509 certificate as PEM, for configuring trust in a SAML SP by hand. PublishSamlMetadata remains the authoritative trust document, verifying certificates inside a rotation overlap included; this endpoint returns only the active certificate, for setting trust up in the first place. No private key is returned.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DownloadSamlSigningCertificate")
@route("/saml/signing-certificate.pem")
@get
op DownloadSamlSigningCertificate1(

): DownloadSamlSigningCertificate1Success_200;

model DownloadSamlSigningCertificate2Success_200 {
  @statusCode statusCode: 200;
}

@doc("Publishes the request tenant's active XmlFederationSigning X.509 certificate as PEM, for configuring trust in a SAML SP by hand. PublishSamlMetadata remains the authoritative trust document, verifying certificates inside a rotation overlap included; this endpoint returns only the active certificate, for setting trust up in the first place. No private key is returned.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DownloadSamlSigningCertificate")
@route("/saml/idp/{profile_id}/signing-certificate.pem")
@get
op DownloadSamlSigningCertificate2(
  @path profile_id: string
): DownloadSamlSigningCertificate2Success_200;

model CreateSamlIdentityProviderProfileSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse;
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

@doc("A tenant administrator creates an additional shared or dedicated SAML IdP profile. The default profile cannot be created this way.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("CreateSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles")
@post
op CreateSamlIdentityProviderProfile(
  @body body: IdMagic.Contract.SamlIdentityProviderProfileRequest
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

@doc("Lists the SAML IdP profiles of the tenant, the default one first and the rest by name ascending.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:read"])
@TypeSpec.OpenAPI.operationId("ListSamlIdentityProviderProfiles")
@route("/api/admin/v1/saml/idp-profiles")
@get
op ListSamlIdentityProviderProfiles(

): ListSamlIdentityProviderProfilesSuccess_200 | ListSamlIdentityProviderProfilesError403;

model UpdateSamlIdentityProviderProfileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse;
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

@doc("A tenant administrator changes the name or the mode of an additional IdP profile. The default profile cannot be changed, and a change to dedicated is refused while more than one SP refers to the profile.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("UpdateSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles/{profile_id}")
@put
op UpdateSamlIdentityProviderProfile(
  @path profile_id: string,
  @body body: IdMagic.Contract.SamlIdentityProviderProfileRequest
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

@doc("Deletes an additional IdP profile nothing refers to. The default profile, and a profile an SP still refers to, are both refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("DeleteSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles/{profile_id}")
@delete
op DeleteSamlIdentityProviderProfile(
  @path profile_id: string
): DeleteSamlIdentityProviderProfileSuccess_204 | DeleteSamlIdentityProviderProfileError400 | DeleteSamlIdentityProviderProfileError403 | DeleteSamlIdentityProviderProfileError409;

model RegisterSamlServiceProviderSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SamlServiceProvider;
}

model RegisterSamlServiceProviderSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.SamlServiceProvider;
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

@doc("Registers or updates (upserts) a SAML service provider. It takes the entityID, the permitted ACS URLs, the NameID format, the audience, the signature policy, and the claim policy, and stores them inside the tenant boundary. The entityID and a closed set of ACS URLs are both required.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("RegisterSamlServiceProvider")
@route("/api/admin/v1/saml/service-providers")
@post
op RegisterSamlServiceProvider(
  @body body: IdMagic.Contract.SamlServiceProviderRequest
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

@doc("Lists the SAML service providers of the tenant by entityID ascending.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:read"])
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

@doc("Deletes the SAML service provider outside the catalog whose entityID matches, idempotently. An SP whose `application_id` is not null is refused with ApplicationOwnedProtocolError (HTTP 409).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("DeleteSamlServiceProvider")
@route("/api/admin/v1/saml/service-providers")
@delete
op DeleteSamlServiceProvider(
  @query entity_id: string
): DeleteSamlServiceProviderSuccess_204 | DeleteSamlServiceProviderError400 | DeleteSamlServiceProviderError403 | DeleteSamlServiceProviderError409;
}

}
