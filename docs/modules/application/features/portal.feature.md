# Feature: ポータルのアプリケーションの例

## Rule: REQ-APPLICATION-003 API トークン発行者は account スコープで自分のポータルアプリケーションだけを操作できる

### Example: EX-APPLICATION-003-01 通常経路

- Given クライアントは対象テナントの `active` User に固定された有効な API アクセストークンを提示している
- When クライアントが `account:read` スコープで、自分に割り当てられたアプリケーションと保存済みの順序をリクエストする
- Then クライアント自身のアプリケーションと保存済みの順序だけが返る
- When クライアントが `account:write` スコープで、自分のアプリケーション順序の保存をリクエストする
- Then クライアント自身のアプリケーション順序が保存される

### Example: EX-APPLICATION-003-02 別テナント向けのトークンを account API へ提示する

- Given クライアントは発行元テナントでは有効な `account:read` スコープの API アクセストークンを持つ
- When クライアントがそのトークンを別テナントの Application account API へ提示する
- Then 操作を 401 の InvalidAccessTokenError で拒否する

### Example: EX-APPLICATION-003-03 クライアントが `account:read` スコープだけを持つ

- Given クライアントは対象テナントの `active` User に固定された有効な API アクセストークンを提示している
- When クライアントが `account:read` スコープで、自分に割り当てられたアプリケーションと保存済みの順序をリクエストする
- Then クライアント自身のアプリケーションと保存済みの順序だけが返る
- When クライアントが `account:write` スコープで、自分のアプリケーション順序の保存をリクエストする
- But クライアントが `account:read` スコープだけを持つ
- Then 操作を 403 と `insufficient_scope` で拒否する
