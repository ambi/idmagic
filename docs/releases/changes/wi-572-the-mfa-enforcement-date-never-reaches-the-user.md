# WI-572: Announce the upcoming MFA enforcement date on the account security page

作業項目は `wi-572-the-mfa-enforcement-date-never-reaches-the-user` である。

WI-572 は、テナントデフォルトのサインインポリシーが将来時刻から MFA を必須にするとき、その強制開始日時を認証要素が未登録の利用者へ知らせる。

`GET /api/account/v1/security` の応答 [AccountSecurityResponse](../../../spec/contexts/authentication/models.tsp) に任意の `mfa_enforcement_start_at` が加わる。
この値は、強制開始がまだ来ておらず、利用者が認証アプリ (TOTP) などの MFA 要素もパスキーも登録していないときだけ返る。
ポリシーのルールそのものは返さない。

アカウントのセキュリティ画面は、この日時と、認証アプリまたはパスキーの事前登録を促す警告を表示する。
画面上で登録を終えると警告は消える。

規範上の条件は [REQ-AUTHENTICATION-019](../../domain/authentication/scenarios.feature.md#rule-req-authentication-019-mfa-の強制開始前は未登録のユーザーもログインできるが登録を促される) が定める。
