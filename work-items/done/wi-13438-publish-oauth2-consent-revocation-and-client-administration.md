---
status: completed
---

# OAuth2 の同意、トークン失効、クライアント管理を公開契約にする

`mise run spec-diff` は規範の差分なしを報告する。
OAuth2 は、本人の同意の一覧と撤回を `consent/ports.AccountConsents`、クライアントの作成、更新、シークレットの管理を `client/ports.ClientAdministration` として公開し、実装をルートパッケージの `Module` から組み立てる。
ログアウトに伴うトークンの失効は、Authentication が定める `deps_http.SessionTokenRevoker` を、OAuth2 のルートが返す値で組み立て地点が満たす。`authhttp.RegisterRoutes` はこれを必須の引数で受け取る。
本人の同意の撤回はクォータの枠を空けない現状のままにし、管理者の撤回との差を wi-23517 に起票した。
境界の負債の台帳から OAuth2 への `private-import` の 6 件を消し、台帳は 78 件から 75 件になった。
HTTP の応答、同意とクライアントの状態、発行するイベントは変えていない。

全文は `git log -1 -- work-items/active/wi-13438-publish-oauth2-consent-revocation-and-client-administration.md` が示すコミットの親、またはそのコミットで読む。
