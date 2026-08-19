namespace IdMagic.Contract {

@doc("Published-language stub for the attribute visibility the Tenancy context owns. Only `private` is excluded unconditionally by the fail-closed floor of claim issuance.")
enum AttrVisibility {
  Private: "private",
  SelfReadable: "self_readable",
  AdminReadable: "admin_readable",
  ClaimExposed: "claim_exposed",
}


@doc("Published-language stub for the attribute definition the Tenancy context owns. Besides deciding attribute visibility during claim issuance, it appears verbatim in both the request and the response body of the tenant's user_attribute_schema.")
model UserAttributeDef {
  @doc("The machine name of the attribute: snake_case starting with a letter, unique across the builtin catalogue and the tenant's custom attributes.")
  @minLength(1)
  key: string;
  @doc("The label shown in the administration console and in self-service forms. Left out, the key itself is displayed.")
  @maxLength(100)
  label?: string;
  @doc("The type of the attribute value. Validating a value against a definition whose type contradicts `multi_valued` fails.")
  type: AttributeType;
  @doc("Whether the attribute holds several values. It can be true only when `type` is `string_array`.")
  multi_valued: boolean = false;
  @doc("Whether a value is required when a User is created or updated.")
  required: boolean = false;
  @doc("Whether the user may edit this attribute from their own self-service profile.")
  editable_by_user: boolean = false;
  @doc("The claim name to use when the attribute is issued as a claim. Left out, the key itself becomes the claim name.")
  @minLength(1)
  @maxLength(100)
  claim_name?: string;
  @doc("The OIDC scope that unlocks exposure as a claim. Left out, the `custom_attribute` scope is what unlocks it.")
  @minLength(1)
  @maxLength(64)
  oidc_scope?: string;
  @doc("How far the attribute may be disclosed. There is no default: a definition without a value is refused, and so is a value outside the enum. Only `claim_exposed` may reach a claim.")
  visibility: AttrVisibility;
  @doc("Whether the value holds personal information. Left out it is false, so an attribute that should be treated as PII has to say true explicitly.")
  pii: boolean = false;
}


@doc("Claim resolution was rejected fail-closed: a required source could not be resolved, an attribute whose visibility is `private` was named as the source of a rule or of the NameID, or a reserved claim type (iss, sub, aud, exp, iat, nbf, jti, azp, nonce, at_hash, c_hash, acr, amr, sid) was named as a `claim_type`. Reaches clients only through the OAuth 2.0 token and UserInfo endpoints, so it carries the OAuthError body rather than Problem Details.")
model ClaimReleaseDeniedError is OAuthError;


@doc("The declarative mapping of one output claim: it builds the output claim type (a URI) from a source.")
model ClaimMappingRule {
  @doc("The claim type URI to emit, such as `http://schemas.xmlsoap.org/claims/UPN`.")
  @minLength(1)
  claim_type: string;
  @doc("Where the claim value comes from.")
  source: ClaimMappingSource;
  @doc("The attribute name to read when `source` is `user_attribute`.")
  source_key?: string;
  @doc("The fixed value to emit when `source` is `fixed`.")
  fixed_value?: string;
  @doc("When true, issuance is refused if the source cannot be resolved (fail-closed).")
  required: boolean = false;
}


@doc("Where a claim value comes from. `user_attribute` reads an attribute of the identity principal, `fixed` emits a static value, and `nameid` carries the NameID that NameIdConfiguration resolved into a claim as well.")
enum ClaimMappingSource {
  user_attribute: "user_attribute",
  fixed: "fixed",
  nameid: "nameid",
}


@doc("The format of the NameID (the subject identifier) of an issued assertion, and where its value comes from.")
model NameIdConfiguration {
  @doc("The NameID format URI, such as `urn:oasis:names:tc:SAML:2.0:nameid-format:persistent`.")
  @minLength(1)
  format: string;
  @doc("The identity attribute that supplies the NameID value.")
  @minLength(1)
  source_attribute: string;
}


@doc("The output of the claim issuance engine: one claim type and its values.")
model IssuedClaim {
  @doc("The claim type URI.")
  @minLength(1)
  claim_type: string;
  @doc("The claim values; a multi-valued attribute yields several.")
  values: string[];
}

}
