# wi-37490-emit-or-retire-declared-events-that-nothing-emits

宣言されていたのにどの経路も発行していなかったドメインイベントを、発行するか、契約から外した。

- パスワードが正しく第二要素を求めたログインは、提示した要素の種類（`factorTypes`）を載せた `MfaChallengeIssued` を記録する。第二要素の成功は `MfaChallengeSucceeded` を、失敗は `MfaChallengeFailed` を、要素の種類（`totp`、`webauthn`）とともに記録する（[`REQ-AUTHENTICATION-015`](../../domain/authentication/mfa/README.md)、[`REQ-AUTHENTICATION-017`](../../domain/authentication/totp/README.md)、[`REQ-AUTHENTICATION-040`](../../domain/authentication/webauthn/README.md)）。
- TOTP と WebAuthn の第二要素の失敗は、`AuthenticationFailed` ではなく `MfaChallengeFailed` として記録する。復旧コードの失敗は引き続き `AuthenticationFailed` である。
- 接続の設定の更新、無効化、削除は、それぞれ `ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` を記録する（[`REQ-PROVISIONING-002`](../../domain/provisioning/connection/README.md)）。
- 登録済みのクライアントの認可か PAR の要求の `authorization_details` を拒否すると、`AuthorizationDetailsRejected` を記録する。`/authorize` は `authorization_details` をクライアントの確認の後に検証するので、不正な `authorization_details` と未登録のクライアントが重なった要求は `invalid_client` で拒否する（[`REQ-OAUTH2-053`](../../domain/oauth2/authorization/README.md)）。
- 発行されていなかった `AuthenticationStepCompleted`、`AuthenticationStepFailed`、`SessionStarted`、`SessionRefreshed` を契約と監査の検索の絞り込みから外した。同じ事実は `UserAuthenticated`、`MfaChallengeIssued`、`AuthenticationFailed` が記録する。
