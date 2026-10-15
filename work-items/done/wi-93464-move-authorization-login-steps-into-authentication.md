---
status: completed
---

# 認可エンドポイントの対話の手順を Authentication へ移す

`mise run spec-diff` は規範の差分なしを報告する。
`/api/auth/login`、`/api/auth/totp`、`/api/auth/webauthn/challenge`、`/api/auth/webauthn`、`/api/auth/recovery-code`、`/api/auth/mfa/enrollment/totp/{start,confirm}` の実装を、TypeSpec の担当どおり Authentication の `signin/handlers_http` へ移した。パスワードの照合、ログインの抑制、第二要素、信頼済みデバイス、MFA の登録、テナントのデフォルトのポリシーの適用、`return_to` の戻りが Authentication に閉じる。
OAuth2 との継ぎ目は三つで、どれも要求する側が定めて組み立て地点で結ぶ。Authentication が定める `signin/ports.PendingAuthorizations` を OAuth2 が実装し、認可要求が求めるサインインの条件を答えて認可を続ける。OAuth2 が定める `SignInSteps` と `LoginSessions` を Authentication が実装し、既存のセッションへの追加の認証と `/end_session` のセッションの終了を受け持つ。
境界の負債の台帳から `private-import` の 8 件と `module-cycle:OAuth2->Audit` を消した。
`/end_session` の主体の照合を HTTP の境界で確かめるテストを足した。
手順の順序、各判定、HTTP の応答、セッションの状態、ドメインイベントは変えていない。

全文は `git log -1 -- work-items/active/wi-93464-move-authorization-login-steps-into-authentication.md` が示すコミットの親、またはそのコミットで読む。
