namespace IdMagic.Contract {

@doc("The JWT signing algorithms. Confined to the strong asymmetric ones FAPI 2.0 permits; symmetric algorithms such as HS256 are excluded.")
enum SignatureAlgorithm {
  PS256: "PS256",
  ES256: "ES256",
}


@doc("The internal states of SigningKeyLifecycle. Active is the current signing key, Verifying is a key that only verifies\nalready-issued tokens during the overlap after a rotation, Retired has left the JWKS, and Archived is kept for audit.")
enum SigningKeyState {
  Active: "Active",
  Verifying: "Verifying",
  Retired: "Retired",
  Archived: "Archived",
}


@doc("Where the key material is kept and who performs the signature. Local holds the private key in the process only; Database\nstores it as cleartext in the application database, so it is in that database's backups; VaultTransit never lets the private\nkey leave Vault. Selecting a provider that persists cleartext key material is an explicit choice, never a default. Database\nnames no particular product.")
enum KeyProvider {
  Local: "Local",
  Database: "Database",
  VaultTransit: "VaultTransit",
}


@doc("What the key is for. Signing covers JWT signatures for OAuth 2.0 and OpenID Connect; XmlFederationSigning covers XML\nsignatures for SAML 2.0, WS-Federation, and WS-Trust, and the X.509 metadata published with them. Even inside one tenant,\neach use has its own active key.")
enum KeyUsage {
  Signing: "Signing",
  XmlFederationSigning: "XmlFederationSigning",
}


@doc("An asymmetric key for signing tokens. Per tenant, use, and scope, one Active key and zero or more Verifying predecessors\nare published (SigningKeyMinJwksOverlap). Listing, searching, rotating, and choosing a signing key all narrow by\n`tenant_id`, `usage`, and `scope_id`. `tenant_id` names an existing Tenant. `scope_id` is `default` normally, and the\nprofile id for an additional SAML IdP profile, so that XML signing credentials stay separate.")
model SigningKey {
  @doc("The tenant the key belongs to. It names an existing Tenant, and the kid of tenant B never appears in the JWKS of tenant A.")
  tenant_id: string = "default";
  @minLength(1)
  kid: string;
  alg: SignatureAlgorithm;
  @doc("Where the key material is kept and who performs the signature.")
  provider: KeyProvider;
  @doc("What the key is for.")
  usage: KeyUsage;
  @doc("An independent key lifecycle inside one tenant and use. Normally `default`; the profile id for an additional SAML IdP profile.")
  @minLength(1)
  scope_id: string = "default";
  public_jwk: JWK;
  @doc("The DER of the self-signed X.509 certificate that goes with an XmlFederationSigning key. Empty for a Signing key, required for XML use. It is public information and holds no private key.")
  certificate_der?: bytes;
  state: SigningKeyState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  @doc("When the key first became Active.")
  activated_at?: utcDateTime;
  @doc("When the key moved to Retired.")
  retired_at?: utcDateTime;
  @doc("When the grace during which a Verifying key stays published on the JWKS ends. From then on it leaves the JWKS and becomes eligible for archiving.")
  expires_at?: utcDateTime;
  archived_at?: utcDateTime;
}


@doc("A JSON Web Key (RFC 7517): one public key, identified by its kid.")
model JWK {
  kty: string;
  use?: string;
  kid: string;
  alg?: string;
  extra?: Record<unknown>;
}


@doc("A JWK Set (RFC 7517 section 5), the public keys the /jwks endpoint returns. A client fetches it to verify token signatures.")
model JWKS {
  keys: JWK[];
}


@doc("The signing key metadata exposed to a tenant administrator. It returns neither the private key nor how the key is stored,\nonly the same public JWK the JWKS carries. It is detailed enough to tell the current signing key from a key that merely\nverifies already-issued tokens during the overlap after a rotation.")
model AdminKeyResponse {
  @doc("The JWK key id, the same one the JWKS publishes.")
  kid: string;
  @doc("The signing algorithm; today PS256 only.")
  alg: SignatureAlgorithm;
  @doc("True for the key that signs new ID tokens and access tokens. False for a key that stays on the JWKS only to verify already-issued tokens during the overlap after a rotation.")
  active: boolean;
  @doc("When the key was generated; SigningKeyMaxAge and SigningKeyMinJwksOverlap are measured from here.")
  created_at: utcDateTime;
  @doc("The public JWK, in the same representation the JWKS uses.")
  public_jwk: JWK;
}


@doc("Every signing key. Two keys are never active at once, because rotation deactivates the old one in the same step.")
model AdminKeyListResponse {
  keys: AdminKeyResponse[];
}


@doc("The result of a key rotation: the kid of the new active key and of the one that just stepped down to verifying.")
model AdminRotateKeyResponse {
  @doc("The new active signing key.")
  next: AdminKeyResponse;
  @doc("The key that was active until now. Absent on the first rotation.")
  previous?: AdminKeyResponse;
}


@doc("Signing key health for one tenant, returned so a system_admin can survey health across tenants:\nprovider, active kid, key count, and provider reachability.\nNo private key is included.")
model TenantSigningKey {
  @doc("The tenant this row is about.")
  tenant_id: string;
  @doc("The KeyProvider this tenant uses.")
  provider: KeyProvider;
  @doc("What the keys are for; today Signing.")
  usage: KeyUsage;
  @doc("The kid of the currently active signing key. Absent for a tenant that has no key yet.")
  active_kid?: string;
  @doc("The state of the active key.")
  active_state?: SigningKeyState;
  @doc("How many keys the JWKS carries (active plus verifying).")
  jwks_key_count: int64;
  @doc("Whether the KeyProvider is reachable. When false, new issuance stops fail-closed; the enforcement point is OAuth2.Token.")
  provider_healthy: boolean;
}


@doc("The signing key on the JWKS was rotated, every 90 days by default or as an emergency rotation.")
model SigningKeyRotated {
  occurredAt: utcDateTime;
  @doc("The tenant the rotated signing key belongs to.")
  tenantId: string;
  newKid: string;
  previousKid?: string;
}


@doc("A signing key moved from Verifying to Retired and left the JWKS, either because SigningKeyMinJwksOverlap had passed or\nbecause it was disabled as an emergency.")
model SigningKeyRetired {
  occurredAt: utcDateTime;
  tenantId: string;
  kid: string;
}


@doc("A signing key moved from Retired to Archived after its grace ended; Archived is terminal and kept for audit.")
model SigningKeyArchived {
  occurredAt: utcDateTime;
  tenantId: string;
  kid: string;
  retiredAt: utcDateTime;
  expiresAt: utcDateTime;
  disposedAt: utcDateTime;
}


@doc("The signing key named by `kid` does not exist in this tenant. RFC 9457 Problem Details with `type` = `urn:idmagic:error:key_not_found`.")
model SigningKeyNotFoundError is ProblemDetails;


@doc("The signing key store is not configured, so keys can neither be read nor rotated. RFC 9457 Problem Details with `type` = `urn:idmagic:error:key_store_unavailable`.")
model KeyStoreUnavailableError is ProblemDetails;

}
