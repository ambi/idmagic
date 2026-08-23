# Authentication State Transitions

## IdentityProviderConnectionLifecycle

上流との接続は、利用できる `Active` と経路を止めた `Disabled` の 2 状態だけを行き来する。作成直後は `Disabled` である。メタデータの再取得の失敗や、信頼の根拠にあたらない項目の更新は状態を変えず、最後に成功した内容を保持する。

| State | Kind | Meaning |
|---|---|---|
| Disabled | initial | 経路を止めている。作成直後はこの状態である |
| Active | — | 上流との接続を利用できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | IdentityProviderConnectionDisabled | — | Disabled |  |
| Disabled | IdentityProviderConnectionActivated | — | Active |  |

## TrustedDeviceLifecycle

信頼済みデバイスは、発行された `Active` と、二度と第二要素を肩代わりしない `Revoked` の 2 状態しか持たない。期限切れは状態ではなく `Active` の行に対する時刻の判定であり、絶対期限か idle 期限のどちらかを過ぎた行は評価の時点で失効として扱う。`Revoked` は終端で、失効は行を削除せず `revoked_at` と `revoke_reason` を設定する tombstone なので、同じ理由での再失効は安全な no-op になる。

利用そのものは状態を変えない。期限内の照合が成功するたびに verifier を回転させて `last_used_at` を進め、cookie を再発行するが、行は `Active` のままである。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 期限内であれば第二要素の提示を肩代わりする。期限切れは状態ではなく時刻の判定である |
| Revoked | terminal | 二度と第二要素を肩代わりしない。行は削除せず `revoked_at` と `revoke_reason` を持つ tombstone として残る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TrustedDeviceRevoked | — | Revoked | revoked_at と revoke_reason を設定する |
| Revoked | TrustedDeviceRevoked | — | Revoked |  |
