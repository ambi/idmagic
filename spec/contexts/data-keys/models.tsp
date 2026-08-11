namespace IdMagic.Contract {

@doc("MasterKey provider への到達不能、wrap/unwrap 失敗、または AAD/tamper 不一致により鍵操作が完了できない。呼び出し元は fail-closed としてアクセスを拒否しなければならない。")
model DataKeyUnavailableError {

}


@doc("DestroyTenantDataKey の対象バージョンについて、所有 context の再暗号化 job (Jobs 経由の data_key_reencryption) がまだ全参照を active バージョンへ移行し終えていない。先に再暗号化を完了させてから destroy する。")
model DataKeyStillReferencedError {

}


@doc("DataEncryptionKeyLifecycle の状態。")
enum DataKeyStatus {
  active: "active",
  retiring: "retiring",
  disabled: "disabled",
  destroyed: "destroyed",
}


@doc("テナントごとの DEK 1 バージョン。MasterKey provider が wrap した鍵材料 (wrapped_dek) だけを保持し、平文 DEK は保存しない。テナントにつき Active は高々 1 本。")
model TenantDataEncryptionKey {
  @doc("内部識別子。EncryptedSecret.key_id が参照する値。")
  @minLength(1)
  id: string;
  @doc("鍵が属するテナント ID。既存 Tenant を指す。")
  tenant_id: string;
  @doc("テナント内で単調増加するバージョン番号。rotation のたびに 1 増える。")
  version: int64;
  status: DataKeyStatus;
  @doc("MasterKey provider で wrap 済みの DEK。平文 DEK は保存・ログ・イベントに一切出さない。")
  wrapped_dek: bytes;
  @doc("この DEK を wrap した MasterKey (provider 側の鍵/バージョン) の識別子。MasterKey 自体の rotation を追跡する。")
  master_key_id: string;
  created_at: utcDateTime;
  activated_at?: utcDateTime;
  disabled_at?: utcDateTime;
  destroyed_at?: utcDateTime;
}


@doc("可逆秘密を Active または Retiring の TenantDataEncryptionKey で暗号化した封筒。所有 context (例: Authentication の MFA シード) はこれを永続化し、平文を保存しない。ciphertext は Tink の AEAD primitive がそのまま出力する不透明な bytes で、nonce/tag は Tink 内部管理のため別フィールドとして持たない (自前で nonce を組み立てない)。")
model EncryptedSecret {
  @doc("暗号化に使った TenantDataEncryptionKey.version。所有レコードのテナント境界内で一意なため、復号は呼び出し側が持つ tenant_id とこの version から対象 TenantDataEncryptionKey を選択する (グローバルな鍵 ID lookup を持たない)。")
  key_version: int64;
  @doc("Tink AEAD primitive が出力する不透明な ciphertext (nonce/tag を内包する)。AAD は呼び出し側が `(tenant, context, table, record id, field)` から再構成して渡す。")
  ciphertext: bytes;
}


@doc("system_admin 向けの鍵健全性スナップショット。秘密鍵材料は含まない。")
model TenantDataKeyHealth {
  tenant_id: string;
  active_version: int64;
  status: DataKeyStatus;
  @doc("MasterKey provider 名 (例: \"openbao\" / \"tink_cleartext\")。")
  provider: string;
  provider_reachable: boolean;
  rotated_at?: utcDateTime;
}


@doc("テナントの最初の DataEncryptionKey (version 1, active) が生成された。")
model DataEncryptionKeyBootstrapped {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
}


@doc("新しい DataEncryptionKey が active になり、旧バージョンが retiring に遷移した。")
model DataEncryptionKeyRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  previousVersion: int64;
  newVersion: int64;
}


@doc("retiring の DataEncryptionKey が手動で即時ロックアウトされた。")
model DataEncryptionKeyDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
  reason?: string;
}


@doc("wrapped_dek が破棄され、当該バージョンの復号が恒久的に不能になった。")
model DataEncryptionKeyDestroyed {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
}

}
