# DataKeys Internals

## BootstrapTenantDataKey
テナントの最初の `DataEncryptionKey` (バージョン 1) を生成し、`MasterKey` プロバイダーでラップして `active` にする内部インターフェース。プロバイダーへ到達できない場合はフェイルクローズで失敗する。
- Result invariant: active_key_count(input.tenant_id) <= 1

## RotateTenantDataKey
テナントに新しい `DataEncryptionKey` (バージョン + 1) を生成して `active` に切り替え、以前の `active` を `retiring` に遷移させる。旧バージョンは即座に `destroy` せず、引き続き復号できる。
- Input invariant: tenant_has_active_data_key(input.tenant_id)
- Result invariant: active_key_count(input.tenant_id) <= 1

## DisableTenantDataKey
ローテーション済み (`retiring`) の `DataEncryptionKey` 1 本を即時に無効化する。危殆化への対応など、`destroy` による暗号学的消去の前に復号を止める場合に使う。`active` なバージョンは対象にできず、先に `RotateTenantDataKey` でローテーションする必要がある。
- Input invariant: data_key_is_not_active(input.tenant_id, input.version)

## DestroyTenantDataKey
`retiring` または `disabled` の `DataEncryptionKey` 1 本について、`wrapped_dek` を破棄して `destroyed` とし、暗号学的に消去する。所有する Context ごとに登録された再暗号化ジョブ (`Jobs` 経由の `data_key_reencryption`) が、すべての参照を `active` バージョンへ移行済みであることを呼び出し前に検証する。未移行の参照が残っていれば `DataKeyStillReferencedError` でフェイルクローズに拒否する。この操作は不可逆である。
- Input invariant: data_key_is_not_active(input.tenant_id, input.version)
- Input invariant: no_pending_reencryption_references(input.tenant_id)

