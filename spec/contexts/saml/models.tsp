namespace IdMagic.Contract {

@doc("How a protocol's assertion names its subject and which claims it carries. The SAML and WS-Federation relying-party contracts decode the same policy, so both point here. `NameIdConfiguration` and `ClaimMappingRule` are the ClaimMapping context's; this model only names the pair the wire carries.")
model ClaimMappingPolicy {
  name_id: NameIdConfiguration;
  rules?: ClaimMappingRule[];
}


@doc("The SAML service provider is owned by an Application and cannot be deleted on its own; delete the Application instead. Returned with HTTP 409. RFC 9457 Problem Details with `type` = `urn:idmagic:error:application_owned_protocol`.")
model ApplicationOwnedProtocolError is ProblemDetails;


@doc("The SAML IdP profile is referenced by a service provider and cannot be deleted. Returned with HTTP 409. RFC 9457 Problem Details with `type` = `urn:idmagic:error:profile_in_use`.")
model IdPProfileInUseError is ProblemDetails;


@doc("The tenant's default SAML IdP profile cannot be changed or deleted. Returned with HTTP 409 and `type` = `urn:idmagic:error:profile_in_use`, the same code the in-use rejection uses.")
model DefaultIdPProfileError is ProblemDetails;


@doc("The SAML 2.0 message bindings, used to receive an SP-initiated AuthnRequest and to return a SAMLResponse. HTTP-Redirect puts deflate+base64 in the query of a GET; HTTP-POST carries base64 in an auto-submitting form.")
enum SamlBinding {
  urn_oasis_names_tc_SAML_2_0_bindings_HTTP_Redirect: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
  urn_oasis_names_tc_SAML_2_0_bindings_HTTP_POST: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
}


@doc("The NameID format of an issued assertion. Both the SP metadata and the NameIDPolicy of an AuthnRequest may ask for one. When the AuthnRequest names anything other than unspecified it is honoured; when it names nothing, the format set in the SP's claim policy is used.")
enum SamlNameIDFormat {
  urn_oasis_names_tc_SAML_2_0_nameid_format_unspecified: "urn:oasis:names:tc:SAML:2.0:nameid-format:unspecified",
  urn_oasis_names_tc_SAML_1_1_nameid_format_emailAddress: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
  urn_oasis_names_tc_SAML_2_0_nameid_format_persistent: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
  urn_oasis_names_tc_SAML_2_0_nameid_format_transient: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
}


@doc("Whether an IdP profile may be shared by several SPs.")
enum SamlIdentityProviderProfileMode {
  shared: "shared",
  dedicated: "dedicated",
}


@doc("A SAML IdP trust boundary inside a tenant. The default profile uses the tenant-wide entityID and the existing SAML endpoints. An additional profile gets its own entityID, endpoints, and XmlFederationSigning key scope, each carrying the profile id.")
model SamlIdentityProviderProfile {
  tenant_id: string = "default";
  @minLength(1)
  profile_id: string;
  @minLength(1)
  name: string;
  mode: SamlIdentityProviderProfileMode;
  is_default: boolean = false;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("How an IdP profile is represented for the administration console. Besides the persisted profile it returns the settings the server derived from the request tenant's canonical issuer, and how many SPs are currently assigned to it.")
model SamlIdentityProviderProfileAdminResponse {
  profile: SamlIdentityProviderProfile;
  entity_id: string;
  metadata_url: string;
  sso_url: string;
  slo_url: string;
  signing_certificate_url: string;
  signing_certificate_fingerprint_sha256: string;
  service_provider_count: int64;
}


@doc("The registration of a service provider (SP) for SAML 2.0 Web Browser SSO. It is identified by its entityID and bundles the closed set of permitted AssertionConsumerService URLs, the audience, the NameID format, the signature policy, and the claim issuance policy. An SP outside the catalogue, whose `application_id` is null, is allowed; when it is not null it names a `saml` Application of the same tenant, and at most one SP is bound to one Application.")
model SamlServiceProvider {
  @doc("The tenant it belongs to. SP registrations and the SAML endpoints are separated per realm.")
  tenant_id: string = "default";
  @doc("The SAML IdP profile assigned to this SP inside the tenant. An SSO or SLO request is accepted only when it matches the profile of the route.")
  @minLength(1)
  idp_profile_id: string = "default";
  @doc("The identifier of the SP (its SAML entityID). It has to match the Issuer of the AuthnRequest exactly. The entityIDType of saml-schema-metadata-2.0.xsd fixes 1024 characters, but the ceiling here sits outside that so a non-conforming SP is not refused. A separate ceiling of 2048 bytes applies as the resource limit that keeps the value inside a btree index row.")
  @minLength(1)
  @maxLength(2048)
  entity_id: string;
  @doc("The Application it corresponds to. Ordinary CRUD leaves it alone; only the bulk creation and deletion of an Application manage the relationship.")
  application_id?: string;
  @doc("The display name in the administration UI.")
  display_name?: string;
  @doc("The closed set of permitted AssertionConsumerService URLs. When an AuthnRequest names an ACS URL, only an exact match against this set is accepted, and only when both the ACS URL and the index are absent does the first entry become the default reply target. Resolving an index is not offered in the initial implementation.")
  acs_urls: string[];
  @doc("The SingleLogoutService URL of the SP, where a Single Logout reply is sent.")
  slo_url?: string;
  @doc("The AudienceRestriction of an issued assertion. Empty means the entity_id is used.")
  audience?: string;
  @doc("The NameID and attribute (claim) issuance policy for this SP.")
  claim_policy: ClaimMappingPolicy;
  @doc("Whether the issued Assertion carries an enveloped signature. It does by default.")
  sign_assertion: boolean = true;
  @doc("Whether the whole Response carries an enveloped signature. Turn it on for an SP that asks to \"sign response\".")
  sign_response: boolean = false;
  @doc("The per-SP policy for verifying signatures on an AuthnRequest or a LogoutRequest. When true, either the Redirect binding signature or the XML signature is verified against `authn_request_signing_certificate_pem`. It is not reflected in WantAuthnRequestsSigned in the IdP metadata.")
  want_authn_requests_signed: boolean = false;
  @doc("The X.509 certificate of the SP, as PEM, used to verify signatures on an AuthnRequest or a LogoutRequest. Required when `want_authn_requests_signed` is true.")
  authn_request_signing_certificate_pem?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The intermediate form holding what has to be checked, extracted from a `<samlp:AuthnRequest>` received during SP-initiated SSO. HTTP-Redirect is deflate+base64 and HTTP-POST is base64; both are decoded before parsing. The decoded XML has a size ceiling, so a deflate bomb cannot get through. Version must be 2.0, and IssueInstant is required and has to sit inside both the permitted clock skew and the maximum age.")
model SamlAuthnRequest {
  @doc("The request id. It round-trips as the InResponseTo of the SAMLResponse and in the SubjectConfirmationData of the assertion.")
  @minLength(1)
  id: string;
  @doc("The entityID of the SP. It resolves to a registered SamlServiceProvider.")
  @minLength(1)
  issuer: string;
  @doc("`AuthnRequest/@Version`. Only `2.0` is accepted.")
  @minLength(1)
  version: string;
  @doc("`AuthnRequest/@IssueInstant`. A request beyond the future skew or the maximum age is refused.")
  issue_instant: utcDateTime;
  @doc("The AssertionConsumerService URL the request names. Only an exact match against the permitted set is accepted.")
  acs_url?: string;
  @doc("AssertionConsumerServiceIndex. Using it together with an ACS URL is refused, and so is naming it at all, because the initial implementation does not resolve an explicit index.")
  acs_index?: int64;
  @doc("`AuthnRequest/@ProtocolBinding`. Only absent or HTTP-POST is accepted.")
  protocol_binding?: SamlBinding;
  @doc("The IdP SSO endpoint URL the request was addressed to.")
  destination?: string;
  @doc("`NameIDPolicy/@Format`. Anything other than unspecified becomes the NameID format of the issued assertion.")
  name_id_format?: SamlNameIDFormat;
  @doc("A demand to reauthenticate. When true, a caller who did not reauthenticate recently is sent back to sign in.")
  force_authn: boolean = false;
  @doc("A demand that there be no interaction. When true and the existing session cannot be used, a NoPassive response is returned rather than a move to the sign-in page.")
  is_passive: boolean = false;
}


@doc("The `<samlp:Response>` returned to the SP. It carries Status Success, an Issuer holding the IdP entityID, and a signed Assertion, and the whole Response is signed too when the SP is configured for it. Destination is the settled ACS URL, and InResponseTo is the AuthnRequest id for an SP-initiated flow. It is base64-encoded under the HTTP-POST binding and auto-posted to the ACS.")
model SamlResponse {
  @doc("The Response id.")
  @minLength(1)
  id: string;
  @doc("The settled ACS URL, drawn from the permitted set.")
  @minLength(1)
  destination: string;
  @doc("The AuthnRequest id for an SP-initiated flow. Empty for an IdP-initiated one.")
  in_response_to?: string;
  issue_instant: utcDateTime;
  @doc("The signed assertion to embed.")
  assertion: SamlAssertion;
}


@doc("The `<saml:Assertion>` embedded in the SAMLResponse. It carries a Subject (the NameID and a bearer SubjectConfirmation), an AudienceRestriction, an AuthnStatement, and an AttributeStatement. The bearer SubjectConfirmationData carries Recipient (the ACS), NotOnOrAfter, and, for an SP-initiated flow, InResponseTo.")
model SamlAssertion {
  @minLength(1)
  id: string;
  @doc("The entityID of the IdP.")
  @minLength(1)
  issuer: string;
  @doc("The audience of the SP; the entity_id by default.")
  @minLength(1)
  audience: string;
  @doc("The Recipient of the bearer SubjectConfirmationData, that is, the ACS URL.")
  recipient: string;
  not_before: utcDateTime;
  not_on_or_after: utcDateTime;
  authn_instant: utcDateTime;
  name_id_format: SamlNameIDFormat;
  @minLength(1)
  name_id_value: string;
}


@doc("A signed SAMLResponse was issued to an SP through SAML SSO.")
model SamlSignInIssued {
  occurredAt: utcDateTime;
  @doc("The tenant that issued it.")
  tenantId: string;
  @doc("The entityID of the SP it was issued to.")
  entityId: string;
  @doc("The subject, the user the NameID resolved to.")
  userId: string;
}


@doc("A SAML SSO request was refused: an unregistered entityID, an ACS URL outside the permitted set, an unassigned subject, a parse failure, and the like.")
model SamlSignInRejected {
  occurredAt: utcDateTime;
  @doc("The tenant that refused it.")
  tenantId: string;
  @doc("The SP entityID that was asked for, when it could be determined.")
  entityId?: string;
  @doc("A summary of why it was refused.")
  reason: string;
}


@doc("A SAML Single Logout destroyed the local session. When there was a LogoutRequest, a LogoutResponse is returned to the registered SLO URL.")
model SamlLogout {
  occurredAt: utcDateTime;
  @doc("The tenant that logged out.")
  tenantId: string;
  @doc("The entityID of the SP that asked for the logout, when it could be determined.")
  entityId?: string;
}


@doc("The JSON request registering or updating a SAML service provider (an upsert). It carries only what the caller may set; `application_id`, the tenant, and the timestamps stay with the server.")
model SamlServiceProviderRequest {
  @doc("The identifier of the SP (its SAML entityID). It has to match the Issuer of the AuthnRequest exactly.")
  entity_id: string;
  @doc("The display name in the administration UI.")
  display_name: string;
  @doc("The SAML IdP profile in the same tenant to assign to this SP.")
  idp_profile_id?: string;
  @doc("The closed set of permitted AssertionConsumerService URLs.")
  acs_urls: url[];
  @doc("The SingleLogoutService URL of the SP.")
  slo_url?: url;
  @doc("The AudienceRestriction of an issued assertion. Empty means the entity_id is used.")
  audience?: string;
  @doc("The NameID and attribute (claim) issuance policy for this SP.")
  claim_policy: ClaimMappingPolicy;
  @doc("Whether the issued Assertion carries an enveloped signature. Left out, it does.")
  sign_assertion?: boolean;
  @doc("Whether the whole Response carries an enveloped signature.")
  sign_response?: boolean;
  @doc("Whether an AuthnRequest and a LogoutRequest have their signatures verified.")
  want_authn_requests_signed?: boolean;
  @doc("The X.509 certificate of the SP, as PEM, used to verify those signatures. Required when `want_authn_requests_signed` is true.")
  authn_request_signing_certificate_pem?: string;
}


@doc("The JSON request creating or updating an additional SAML IdP profile. The default profile can be neither created nor changed this way.")
model SamlIdentityProviderProfileRequest {
  @doc("The display name of the profile.")
  name: string;
  @doc("Whether the profile may be shared by several SPs.")
  mode: SamlIdentityProviderProfileMode;
}


@doc("The SAML identity provider profile named by the request does not exist in this tenant. RFC 9457 Problem Details with `type` = `urn:idmagic:error:not_found`.")
model SamlIdentityProviderProfileNotFoundError is ProblemDetails;

}
