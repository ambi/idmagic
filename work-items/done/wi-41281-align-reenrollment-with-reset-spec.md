---
status: completed
---

# 認証器の全リセット後の再登録の強制について規範と実装を揃える

全リセット後の再登録は、実効のサインインポリシーが MFA を求めるときだけ強制する方針に揃えた。
認証強度の選択を SignInPolicy に保つため、ポリシーによらない強制は採らなかった。
`spec-diff` が示す差分は REQ-AUTHENTICATION-022 と、TypeSpec の `ResetUserAuthenticators`、`AuthenticatorResetResponse` の説明である。
`reenrollment_required=true` は認証器がなくなり登録の許可を発行した状態を表し、MFA 不要なら許可を消費せずパスワードだけでログインすることを明記した。
MFA 必須時の再登録に加え、MFA 不要時の新しい直接ログイン、認可コードの発行、pwd の AMR と登録の許可の非消費を受け入れテストで固定した。
本番の振る舞いは維持し、復旧の手段と既存セッションの失効方式は対象外とした。
