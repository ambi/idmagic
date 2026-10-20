---
status: completed
---

# 本人の同意管理の経路を OAuth2 へ移す

TypeSpec に `ConsentNotFoundError` を加え、`RevokeMyConsent`、`GetAdminConsent`、`RevokeAdminConsent` の 404 を宣言した。
要件文は変えていない。REQ-OAUTH2-032 と REQ-OAUTH2-038 がすでに定めていた応答を契約へ写した。

本人の同意の一覧と取り消しのハンドラーを Authentication から OAuth2 の HTTP アダプターへ移し、OAuth2 の同意の usecase を直接呼ぶようにした。
Authentication からは同意操作と表示名の解決の依存が消え、`Authentication → OAuth2` の循環が実測で消えた。台帳は 14 件から 13 件になった。
起票時の設計は要求側のポートと変換を置く案だったが、同じ操作の契約、値、エラーを複製するため採らなかった。
ルーターの group と本人認証の認証器は両モジュールで共有されており、移設で動かすものはなかった。

URL、応答、スコープの判定、クォータを空けない本人の取り消し、`ConsentRevoked` の発行は変えていない。
移設の前に、本人に同意のないクライアントの撤回が HTTP 越しに 404 と `consent_not_found` で拒否されるテストを足した。
`check-status-drift` は、ヘルパー関数が書く応答を `unverified` として扱うため、今回の 404 の欠落を検出していなかった。

全文は `git log -1 -- work-items/active/wi-75520-move-account-consent-routes-to-oauth2.md` が示すコミットの親、またはそのコミットで読む。
