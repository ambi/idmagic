# 信頼済みデバイスの状態遷移

## TrustedDeviceLifecycle

信頼済みデバイスは、発行された `Active` と、二度と第二要素を肩代わりしない `Revoked` の 2 状態しか持たない。期限切れは状態ではなく `Active` のレコードに対する時刻の判定であり、絶対期限か idle 期限のどちらかを過ぎたレコードは評価の時点で失効として扱う。`Revoked` は終端で、失効はレコードを削除せず `revoked_at` と `revoke_reason` を設定する tombstone なので、同じ理由での再失効は安全な no-op になる。

利用そのものは状態を変えない。期限内の照合が成功するたびに verifier を回転させて `last_used_at` を進め、cookie を再発行するが、レコードは `Active` のままである。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 期限内であれば第二要素の提示を肩代わりする。期限切れは状態ではなく時刻の判定である |
| Revoked | terminal | 二度と第二要素を肩代わりしない。レコードは削除せず `revoked_at` と `revoke_reason` を持つ tombstone として残る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TrustedDeviceRevoked | — | Revoked | revoked_at と revoke_reason を設定する |
| Revoked | TrustedDeviceRevoked | — | Revoked |  |
