---
status: completed
---

# MFA を失った制御面の運用者が復旧できることを確かめる

規範は変えていない。MFA を失った制御面の運用者の復旧の経路を、HTTP の受け入れテストと runbook で確かめた。

`system_admin` だけを持つ運用者は、デフォルトのテナントの別の管理者が認証器を全リセットすると、次のログインで再登録し、`system_admin` のまま `otp` で MFA を満たすセッションを得る。
`admin` も持つ運用者は、デフォルトのポリシーが MFA を求めない間、パスワードで管理画面へ入って自分をリセットし、アカウントの画面で新しい TOTP を登録し、MFA の水準を求める再認可でその TOTP を使って MFA のセッションを得る。
後者では、REQ-AUTHENTICATION-022 が求めるリセット後の再登録の強制が起きなかった。実装はポリシーが MFA を求めるときだけ強制する。この食い違いは [[wi-41281-align-reenrollment-with-reset-spec]] で扱う。

リセットできる管理者がいない構成（MFA を失った本人だけが `admin` で、デフォルトのポリシーが MFA を求める場合と、`admin` がいない場合）では復旧できない。
利用者と決めて、[[wi-468-system-console-privileged-session-assurance]] の前提にはせず、runbook の予防（`admin` を二人以上、復旧コードの保管）で避け、手段は [[wi-31080-recover-control-plane-mfa-from-the-runtime]] で扱う。
受け入れテストは既存の振る舞いを確かめるので RED にならない。最初の想定（自分でリセットした後の再登録の強制）が実際に失敗したことが、テストが経路の違いを検出する証拠になった。
