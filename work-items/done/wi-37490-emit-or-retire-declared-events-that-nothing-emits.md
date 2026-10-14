---
status: completed
archived_from: 790a71869085939e6c4b610e06504f41bd150d2e
---

# 宣言しているのに発行されないドメインイベントを、発行するか宣言から外す

`mise run spec-diff` は、main に対して REQ-AUTHENTICATION-040 の追加、REQ-AUTHENTICATION-015、REQ-AUTHENTICATION-017、REQ-OAUTH2-053、REQ-PROVISIONING-002 の変更、TypeSpec の `AuthenticationStepCompleted`、`AuthenticationStepFailed`、`SessionStarted`、`SessionRefreshed` の削除、`MfaChallengeIssued` の変更を報告した。
ログインセッションを第二要素の照合の待ちにする四つの経路は、提示できる要素の種類を `factorTypes` に載せた `MfaChallengeIssued` を発行する。TOTP と WebAuthn の第二要素の成功は `MfaChallengeSucceeded` を、失敗は `AuthenticationFailed` の代わりに `MfaChallengeFailed` を発行する。復旧コードは要素の種類ではないので、従来どおり `AuthenticationFailed` と `BackupCodeConsumed` が記録する。
WebAuthn のログインの第二要素の要件 REQ-AUTHENTICATION-040 を新設し、`webauthn_not_enrolled` も要件に書いた。
Provisioning の接続の設定の更新、無効化への遷移、削除は、保存の後に `ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` を発行する。`DeleteConnection` は時刻を引数に取る。
`/authorize` と PAR は、登録済みのクライアントの `authorization_details` を拒否したとき `AuthorizationDetailsRejected` を発行する。`/authorize` はクライアントを確かめた後に `authorization_details` を解析するようになり、`AuthorizeRequestInput` は生の文字列 `AuthorizationDetailsRaw` を受け取る。
同じ事実をほかのイベントが記録する 4 件を、Go の型、保持期間の分類、監査の検索の候補（バックエンドと画面）から外した。語彙の検査の許容リストには、なりすましの 2 件だけが残る。なりすまし機能の新設は wi-59470 として起票した。

全文は `git show 790a71869085939e6c4b610e06504f41bd150d2e:<このファイルのパス>` で読む。
