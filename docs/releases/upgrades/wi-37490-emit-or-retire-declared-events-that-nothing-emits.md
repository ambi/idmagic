# WI-37490: 第二要素の失敗を MfaChallengeFailed で検索する

作業項目は `wi-37490-emit-or-retire-declared-events-that-nothing-emits` である。

監査を種類で検索している運用者は、次のとおり検索の条件を変える。

| これまでの種類 | 今後の種類 |
| --- | --- |
| `AuthenticationFailed`（理由 `invalid_code`、`no_factor`、`webauthn_invalid`） | `MfaChallengeFailed`（`factorType` が `totp` か `webauthn`） |
| `AuthenticationStepCompleted`、`AuthenticationStepFailed` | `MfaChallengeIssued`、`UserAuthenticated`、`AuthenticationFailed` |
| `SessionStarted` | `UserAuthenticated` |
| `SessionRefreshed` | なし |

外した 4 種類は一度も発行されていなかったので、保存済みの監査の記録に残っているものはない。
データ移行や設定変更は不要である。

互換性境界は [REQ-AUTHENTICATION-017](../../modules/authentication/totp/README.md) と [REQ-AUTHENTICATION-040](../../modules/authentication/webauthn/README.md) が定める。
