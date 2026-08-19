namespace IdMagic.Contract {

@doc("The key operation cannot complete because the MasterKey provider is unreachable, a wrap or unwrap failed, or the AAD did not match (tampering). Callers must deny access fail-closed. A domain failure condition, not an HTTP response body: no operation returns it directly, and where it aborts a request the response is the generic 500 Problem Details.")
model DataKeyUnavailableError {

}


@doc("For the version DestroyTenantDataKey targets, the re-encryption job of the owning context (`data_key_reencryption` through Jobs) has not yet moved every reference onto the active version. Complete the re-encryption before destroying. A domain failure condition, not an HTTP response body.")
model DataKeyStillReferencedError {

}


@doc("The states of DataEncryptionKeyLifecycle.")
enum DataKeyStatus {
  active: "active",
  retiring: "retiring",
  disabled: "disabled",
  destroyed: "destroyed",
}


@doc("One version of a tenant's DEK. It holds only the key material the MasterKey provider wrapped (`wrapped_dek`); the plaintext DEK is never stored. A tenant has at most one Active version.")
model TenantDataEncryptionKey {
  @doc("Internal identifier, the value `EncryptedSecret.key_id` refers to.")
  @minLength(1)
  id: string;
  @doc("The tenant the key belongs to. It names an existing Tenant.")
  tenant_id: string;
  @doc("A version number that rises monotonically within the tenant, by one on every rotation.")
  version: int64;
  status: DataKeyStatus;
  @doc("The DEK as the MasterKey provider wrapped it. The plaintext DEK never appears in storage, in a log, or in an event.")
  wrapped_dek: bytes;
  @doc("Identifies the MasterKey -- the provider's key and version -- that wrapped this DEK, so that rotation of the MasterKey itself can be traced.")
  master_key_id: string;
  created_at: utcDateTime;
  activated_at?: utcDateTime;
  disabled_at?: utcDateTime;
  destroyed_at?: utcDateTime;
}


@doc("An envelope holding a reversible secret encrypted under an Active or Retiring TenantDataEncryptionKey. The owning context (the MFA seed of Authentication, for one) persists this and never the plaintext. The ciphertext is the opaque bytes the Tink AEAD primitive emits as they are; nonce and tag are Tink's own business and get no separate field here, because we never assemble a nonce ourselves.")
model EncryptedSecret {
  @doc("The `TenantDataEncryptionKey.version` the secret was encrypted under. It is unique inside the tenant boundary of the owning record, so decryption selects the TenantDataEncryptionKey from the caller's tenant_id together with this version; there is no global key id lookup.")
  key_version: int64;
  @doc("The opaque ciphertext the Tink AEAD primitive emits, nonce and tag included. The caller rebuilds the AAD from `(tenant, context, table, record id, field)` and passes it in.")
  ciphertext: bytes;
}


@doc("A key health snapshot for a system_admin. It carries no private key material.")
model TenantDataKeyHealth {
  tenant_id: string;
  active_version: int64;
  status: DataKeyStatus;
  @doc("The MasterKey provider name, such as `openbao` or `tink_cleartext`.")
  provider: string;
  provider_reachable: boolean;
  rotated_at?: utcDateTime;
}


@doc("The tenant's first DataEncryptionKey (version 1, active) was generated.")
model DataEncryptionKeyBootstrapped {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
}


@doc("A new DataEncryptionKey became active and the previous version moved to retiring.")
model DataEncryptionKeyRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  previousVersion: int64;
  newVersion: int64;
}


@doc("A retiring DataEncryptionKey was locked out by hand, immediately.")
model DataEncryptionKeyDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
  reason?: string;
}


@doc("The `wrapped_dek` was destroyed, so nothing encrypted under that version can ever be decrypted again.")
model DataEncryptionKeyDestroyed {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
}

}
