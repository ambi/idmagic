namespace IdMagic.Contract {

@doc("The SAML version of the assertion WS-Federation issues, which also becomes the TokenType of the RSTR. SAML 1.1 (urn:oasis:names:tc:SAML:1.0:assertion) is what Entra and AD FS default to for WS-Fed.")
enum WsFedTokenType {
  urn_oasis_names_tc_SAML_1_0_assertion: "urn:oasis:names:tc:SAML:1.0:assertion",
  urn_oasis_names_tc_SAML_2_0_assertion: "urn:oasis:names:tc:SAML:2.0:assertion",
}


@doc("The registration of a relying party (RP) for WS-Federation passive. It is identified by its wtrealm and bundles the closed set of permitted `wreply` targets, the audience, and the claim issuance policy. An RP outside the catalogue, whose `application_id` is null, is allowed; when it is not null it names a `wsfed` Application of the same tenant, and at most one RP is bound to one Application.")
model WsFedRelyingParty {
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The identifier of the RP: the WS-Fed wtrealm, which is also the SAML audience. WS-Federation says only that it is a URI, so the ceiling follows the URI class. A separate ceiling of 2048 bytes applies as the resource limit that keeps the value inside a btree index row.")
  @minLength(1)
  @maxLength(2048)
  wtrealm: string;
  @doc("The Application it corresponds to. Ordinary CRUD leaves it alone; only the bulk creation and deletion of an Application manage the relationship.")
  application_id?: string;
  @doc("The display name in the administration UI.")
  display_name?: string;
  @doc("The closed set of permitted `wreply` targets. Only an exact match against this set is accepted.")
  reply_urls: string[];
  @doc("The audience of an issued assertion. Empty means the wtrealm is used.")
  audience?: string;
  @doc("The SAML version of an issued assertion. Empty means SAML 1.1, the default Entra's WS-Fed expects.")
  token_type?: WsFedTokenType;
  @doc("The claim issuance policy for this RP.")
  claim_policy: ClaimMappingPolicy;
  @doc("The Microsoft Entra domain federation preset. It bundles the required claims and the sourceAnchor transform used when the federation is configured.")
  entra_profile?: EntraFederationProfile;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The profile for Microsoft Entra domain federation. It bundles the IssuerUri, the sourceAnchor attribute, and the passive, active, and MEX endpoints, and applies the claim preset for UPN, ImmutableID, and a persistent NameID to the WS-Fed RP.")
model EntraFederationProfile {
  @doc("A DNS domain already verified in Entra.")
  @minLength(1)
  domain: string;
  @doc("The IssuerUri registered in Entra. It matches the issuer of the tokens issued and the wtrealm of the RP.")
  @minLength(1)
  issuer_uri: string;
  @doc("The immutable user attribute the ImmutableID is built from: either the GUID string of objectGUID or a value already base64-encoded.")
  @minLength(1)
  source_anchor_attribute: string;
  @doc("The synthetic attribute name the converted ImmutableID is stored under; `entra_immutable_id` by default.")
  @minLength(1)
  immutable_id_attribute: string;
  @doc("The Entra PassiveLogOnUri, normally `/wsfed`.")
  passive_logon_uri: string;
  @doc("The Entra ActiveLogOnUri, normally `/trust/usernamemixed`.")
  active_logon_uri: string;
  @doc("The Entra MetadataExchangeUri, normally `/trust/mex`.")
  metadata_exchange_uri: string;
}


@doc("An STS endpoint advertised in the WS-Trust and WS-Federation metadata, describing the authentication method and the binding it accepts.")
model SecurityTokenEndpoint {
  @doc("The published endpoint URL.")
  url: string;
  @doc("For instance `usernamemixed`, `mex`, or `passive`.")
  @minLength(1)
  binding: string;
  @doc("The authentication method accepted, `UsernameToken` for instance.")
  auth_methods: string[];
}


@doc("The intermediate form holding what has to be checked, extracted from the RST of the WS-Trust 1.3 Issue binding.")
model WsTrustRequestSecurityToken {
  @doc("The WS-Addressing MessageID, used to prevent replay.")
  @minLength(1)
  message_id: string;
  @doc("WS-Addressing `To`: the STS endpoint the request was addressed to.")
  to: string;
  @doc("WS-Addressing `Action`. Only Issue is accepted.")
  action: string;
  @doc("The username of the WS-Security UsernameToken.")
  @minLength(1)
  username: string;
  @doc("The password credential of the WS-Security UsernameToken.")
  @minLength(1)
  password: string;
  @doc("WS-Security Timestamp `Created`.")
  created_at: utcDateTime;
  @doc("WS-Security Timestamp `Expires`.")
  expires_at: utcDateTime;
  @doc("The RP, that is, the audience. It resolves to a registered WsFedRelyingParty.")
  @minLength(1)
  applies_to: string;
  @doc("The token type requested. Empty means the RP's default.")
  token_type?: WsFedTokenType;
  @doc("The WS-Trust RequestType. Only absent or Issue is accepted.")
  request_type?: string;
  @doc("The WS-Trust KeyType. Only absent or Bearer is accepted.")
  key_type?: string;
}


@doc("The RSTR for a WS-Trust Issue. It carries the signed SAML assertion together with the lifetime and appliesTo.")
model WsTrustResponse {
  @doc("The MessageID of the originating RST.")
  relates_to?: string;
  @minLength(1)
  applies_to: string;
  token_type: WsFedTokenType;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  assertion_id?: string;
}


@doc("A signed assertion was issued to an RP through WS-Federation passive sign-in.")
model WsFedSignInIssued {
  occurredAt: utcDateTime;
  @doc("The tenant that issued it.")
  tenantId: string;
  @doc("The wtrealm of the RP it was issued to.")
  wtrealm: string;
  @doc("The subject, the user the NameID resolved to.")
  userId: string;
}


@doc("A WS-Federation passive sign-in was refused: an unregistered wtrealm, a `wreply` outside the permitted set, a failed check, and the like.")
model WsFedSignInRejected {
  occurredAt: utcDateTime;
  @doc("The tenant that refused it.")
  tenantId: string;
  @doc("The wtrealm that was asked for, when it could be determined.")
  wtrealm?: string;
  @doc("A summary of why it was refused.")
  reason: string;
}


@doc("A WS-Federation sign-out destroyed the local session.")
model WsFedSignOut {
  occurredAt: utcDateTime;
  @doc("The tenant that signed out.")
  tenantId: string;
  @doc("The wtrealm of the RP that asked for the sign-out, when it could be determined.")
  wtrealm?: string;
}


@doc("A signed assertion was issued through the WS-Trust active STS.")
model WsTrustTokenIssued {
  occurredAt: utcDateTime;
  @doc("The tenant that issued it.")
  tenantId: string;
  @doc("The RP, that is, the audience, it was issued to.")
  appliesTo: string;
  @doc("The subject user.")
  userId: string;
}


@doc("An Issue request to the WS-Trust active STS was refused.")
model WsTrustTokenRejected {
  occurredAt: utcDateTime;
  @doc("The tenant that refused it.")
  tenantId: string;
  @doc("The AppliesTo that was asked for, when it could be determined.")
  appliesTo?: string;
  @doc("A summary of why it was refused.")
  reason: string;
}


@doc("A Microsoft Entra domain federation preset was configured and the matching WS-Fed RP trust was upserted.")
model EntraFederationConfigured {
  occurredAt: utcDateTime;
  @doc("The tenant that was configured.")
  tenantId: string;
  @doc("The domain verified in Entra.")
  domain: string;
  @doc("The IssuerUri registered in Entra.")
  issuerUri: string;
}


@doc("The JSON request registering or updating a WS-Federation relying party. It carries only what the caller may set; `application_id`, the tenant, and the timestamps stay with the server.")
model WsFedRelyingPartyRequest {
  @doc("The identifier of the RP: the WS-Fed wtrealm, which is also the SAML audience.")
  wtrealm: string;
  @doc("The display name in the administration UI.")
  display_name: string;
  @doc("The closed set of permitted `wreply` targets. Only an exact match against this set is accepted.")
  reply_urls: url[];
  @doc("The audience of an issued assertion. Empty means the wtrealm is used.")
  audience?: string;
  @doc("The claim issuance policy for this RP.")
  claim_policy: ClaimMappingPolicy;
}

}
