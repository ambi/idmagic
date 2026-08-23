# DataKeys State Transitions

## DataEncryptionKeyLifecycle

`bootstrap` で最初の鍵を `active` として生成する。ローテーションで新しいバージョンが `active` になると、旧バージョンは `retiring` へ遷移する。`retiring` の鍵は `disable` で即時に無効化できる。`retiring` または `disabled` の鍵を `destroyed` へ遷移できるのは、すべての参照を再暗号化したことを `Jobs` 経由で確認した後だけである。`active` の鍵は直接 `disable` または `destroy` できず、先にローテーションする必要がある。

| State | Kind | Meaning |
|---|---|---|
| active | initial | 新規の暗号化操作に使う。テナントごとに高々 1 本だけ存在する |
| retiring | — | 新規の暗号化には使わないが、既存の暗号文の復号には引き続き使う |
| disabled | — | 鍵素材の危殆化などにより手動で即時に無効化した。以後この鍵による復号はフェイルクローズで拒否する |
| destroyed | terminal | `wrapped_dek` を破棄し、暗号学的消去が成立した。元に戻せない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| active | DataEncryptionKeyRotated | — | retiring |  |
| retiring | DataEncryptionKeyDisabled | — | disabled |  |
| retiring | DataEncryptionKeyDestroyed | — | destroyed |  |
| disabled | DataEncryptionKeyDestroyed | — | destroyed |  |
