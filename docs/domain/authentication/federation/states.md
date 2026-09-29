# 外部 IdP との連携の状態遷移

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
