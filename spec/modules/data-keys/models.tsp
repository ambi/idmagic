namespace IdMagic.Contract {

@doc("MasterKey のプロバイダーに到達できない、ラップまたはアンラップに失敗した、AAD が一致しない（改ざん）のいずれかの理由で、鍵の操作を完了できない。呼び出し元はフェイルクローズでアクセスを拒否しなければならない。これはドメインの失敗条件であり、HTTP のレスポンスボディではない。どの操作もこれを直接返さず、リクエストを中断した場合のレスポンスは汎用の 500 の Problem Details である。")
model DataKeyUnavailableError {

}


@doc("DestroyTenantDataKey が対象とするバージョンについて、所有するモジュールの再暗号化ジョブ（Jobs による `data_key_reencryption`）が、まだすべての参照を有効なバージョンへ移し終えていない。破棄する前に再暗号化を完了させる。これはドメインの失敗条件であり、HTTP のレスポンスボディではない。")
model DataKeyStillReferencedError {

}


@doc("DataEncryptionKeyLifecycle の状態。")
enum DataKeyStatus {
  active: "active",
  retiring: "retiring",
  disabled: "disabled",
  destroyed: "destroyed",
}


@doc("テナントの DEK の一つのバージョン。保存するのは MasterKey のプロバイダーがラップした鍵（`wrapped_dek`）だけで、平文の DEK は保存しない。テナントの Active なバージョンは一つまでである。")
model TenantDataEncryptionKey {
  @doc("内部の識別子。`EncryptedSecret.key_id` が参照する値である。")
  @minLength(1)
  id: string;
  @doc("鍵が属するテナント。既存の Tenant を指す。")
  tenant_id: string;
  @doc("テナントの中で単調に増えるバージョン番号。ローテーションのたびに 1 ずつ増える。")
  version: int64;
  status: DataKeyStatus;
  @doc("MasterKey のプロバイダーがラップした DEK。平文の DEK は、ストレージ、ログ、イベントのどこにも現れない。")
  wrapped_dek: bytes;
  @doc("この DEK をラップした MasterKey（プロバイダーの鍵とそのバージョン）を識別する。MasterKey 自体のローテーションを追跡できるようにするためである。")
  master_key_id: string;
  created_at: utcDateTime;
  activated_at?: utcDateTime;
  disabled_at?: utcDateTime;
  destroyed_at?: utcDateTime;
}


@doc("Active または Retiring の TenantDataEncryptionKey で暗号化した、可逆なシークレットを収めるエンベロープ。所有するモジュール（たとえば Authentication の MFA のシード）はこれを永続化し、平文は永続化しない。暗号文は Tink の AEAD プリミティブが出力した不透明なバイト列そのままである。ノンスとタグは Tink が扱うもので、ここに個別のフィールドはない。ノンスを自前で組み立てることがないためである。")
model EncryptedSecret {
  @doc("シークレットを暗号化したときの `TenantDataEncryptionKey.version`。所有する記録のテナント境界の中で一意なので、復号では呼び出し元の tenant_id とこのバージョンから TenantDataEncryptionKey を選ぶ。全体で一意な鍵 ID による検索はない。")
  key_version: int64;
  @doc("Tink の AEAD プリミティブが出力する不透明な暗号文。ノンスとタグを含む。呼び出し元は `(tenant, context, table, record id, field)` から AAD を組み立て直して渡す。")
  ciphertext: bytes;
}


@doc("system_admin 向けの鍵の健全性のスナップショット。秘密鍵は含まない。")
model TenantDataKeyHealth {
  tenant_id: string;
  active_version: int64;
  status: DataKeyStatus;
  @doc("MasterKey のプロバイダー名。`openbao` や `tink_cleartext` など。")
  provider: string;
  provider_reachable: boolean;
  rotated_at?: utcDateTime;
}


@doc("テナントの最初の DataEncryptionKey（バージョン 1、active）を生成した。")
model DataEncryptionKeyBootstrapped {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
}


@doc("新しい DataEncryptionKey が有効になり、それまでのバージョンが retiring へ移った。")
model DataEncryptionKeyRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  previousVersion: int64;
  newVersion: int64;
}


@doc("retiring の DataEncryptionKey を手動で、即座に締め出した。")
model DataEncryptionKeyDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
  reason?: string;
}


@doc("`wrapped_dek` を破棄したので、そのバージョンで暗号化したものは二度と復号できない。")
model DataEncryptionKeyDestroyed {
  occurredAt: utcDateTime;
  tenantId: string;
  version: int64;
}

}
