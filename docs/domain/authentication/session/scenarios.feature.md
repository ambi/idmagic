# Feature: ログインセッションのシナリオ

## 失効と変更

### Rule: REQ-AUTHENTICATION-013 ユーザーは自分の有効なセッションを一覧して失効できる

Primary actor: `AuthenticatedSelf`

#### Example: EX-AUTHENTICATION-013-01 通常経路

- Given ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- When ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
- Then 自分の有効なセッションが返る
- When ユーザー "alice" が現在以外のセッションを 1 件失効させる
- Then 失効したセッションは一覧から消える
- When ユーザー "alice" が現在以外のすべてのセッションを一括失効させる
- Then 現在のセッションだけが残る

#### Example: EX-AUTHENTICATION-013-02 プロセスの再起動を挟んでセッション一覧を取得する

- Given ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- When ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
- But プロセスの再起動を挟んでセッション一覧を取得する
- Then サーバープロセスを再起動する
- And ユーザー "alice" が同じセッション Cookie でアクティビティ画面を開く
- And セッションは再起動前と同じ内容で解決できる

#### Example: EX-AUTHENTICATION-013-03 既に失効済みのセッションへ同じ失効操作を再送する

- Given ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- When ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
- Then 自分の有効なセッションが返る
- When ユーザー "alice" が現在以外のセッションを 1 件失効させる
- But 既に失効済みのセッションへ同じ失効操作を再送する
- Then ユーザー "alice" が直前に失効させた同じセッション ID へ再度失効を要求する
- And 要求は成功として扱われ、最初の失効時刻を保持する

### Rule: REQ-AUTHENTICATION-021 管理者は対象ユーザーのセッションを一覧・個別失効・全失効できる

Primary actor: `TenantAdministrator`

#### Example: EX-AUTHENTICATION-021-01 通常経路

- Given ユーザー "alice" が複数の有効な LoginSession を持つ
- When 管理者がユーザー "alice" の ListSessions を呼ぶ
- Then 開始時刻の降順で有効なセッション一覧が返る
- When 管理者がそのうち 1 件の `RevokeSession` を呼ぶ
- Then 対象セッションは `revoke_reason=admin_revoke` で失効し、"SessionEnded" が発行される
- When 管理者がユーザー "alice" の RevokeUserSessions を呼ぶ
- Then 残り全セッションが失効する

#### Example: EX-AUTHENTICATION-021-02 他テナントの管理者が呼び出す

- Given ユーザー "alice" が複数の有効な LoginSession を持つ
- When 管理者がユーザー "alice" の ListSessions を呼ぶ
- But 他テナントの管理者が呼び出す
- Then エラー "AccessDeniedError"

#### Example: EX-AUTHENTICATION-021-03 既に失効済みのセッションへ再度 `RevokeSession` を呼ぶ

- Given ユーザー "alice" が複数の有効な LoginSession を持つ
- When 管理者がユーザー "alice" の ListSessions を呼ぶ
- Then 開始時刻の降順で有効なセッション一覧が返る
- When 管理者がそのうち 1 件の `RevokeSession` を呼ぶ
- But 既に失効済みのセッションへ再度 `RevokeSession` を呼ぶ
- Then 204 が返り、`revoked_at` は初回の値を保持する

### Rule: REQ-AUTHENTICATION-035 サインアウトはテナントのエンドポイント形式によらずサーバー側のセッションを失効させる

Primary actor: `EndUser`

#### Example: EX-AUTHENTICATION-035-01 通常経路

- Given ユーザー "alice" がサブドメイン形式のテナントでログインし、ブラウザは `__Host-` 接頭辞つきのセッション Cookie を保持している
- When ユーザー "alice" が SAML シングルログアウトでサインアウトする
- Then そのセッションはサーバー側で失効し、以後の認証解決は未認証として扱われる
- Then ブラウザの Cookie を復元して同じセッション ID を再提示しても認証されない

#### Example: EX-AUTHENTICATION-035-02 WS-Federation のサインアウトを使う

- Given ユーザー "alice" がサブドメイン形式のテナントでログインし、ブラウザは `__Host-` 接頭辞つきのセッション Cookie を保持している
- When ユーザー "alice" が SAML シングルログアウトでサインアウトする
- But WS-Federation のサインアウトを使う
- Then 同じくサーバー側のセッションが失効する

#### Example: EX-AUTHENTICATION-035-03 パス形式のテナントで接頭辞のない Cookie を送る

- Given ユーザー "alice" がサブドメイン形式のテナントでログインし、ブラウザは `__Host-` 接頭辞つきのセッション Cookie を保持している
- When ユーザー "alice" が SAML シングルログアウトでサインアウトする
- But パス形式のテナントで接頭辞のない Cookie を送る
- Then 同じくサーバー側のセッションが失効する
