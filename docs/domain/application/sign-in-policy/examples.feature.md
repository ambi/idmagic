# Feature: サインインポリシーの例

## Rule: REQ-APPLICATION-009 管理者はアプリケーション別サインインポリシーを設定できる

### Example: EX-APPLICATION-009-01 通常経路

- Given 管理者が Application 編集画面を開いている
- And Application は OIDC / SAML / WS-Fed のいずれか 1 つのプロトコルを持つ
- When 管理者が MFA 必須と再認証を求めるまでの時間（秒）を指定したサインインポリシーを保存する
- Then AppSignInPolicyUpdated が発行される
- When 単要素セッションの利用者が対象 Application にアクセスする
- Then システムはトークン / Assertion の発行前にポリシーを評価する（強制点は OAuth2.Authorize）
- Then ステップアップ認証が可能な経路ではステップアップ認証を要求し、認証強度を上げた後にフェデレーションを完了する

### Example: EX-APPLICATION-009-02 管理者以外がポリシーを更新する

- Given 管理者が Application 編集画面を開いている
- And Application は OIDC / SAML / WS-Fed のいずれか 1 つのプロトコルを持つ
- When 管理者が MFA 必須と再認証を求めるまでの時間（秒）を指定したサインインポリシーを保存する
- But 管理者以外がポリシーを更新する
- Then AccessDeniedError で拒否される

### Example: EX-APPLICATION-009-03 クライアント IP が許可 CIDR に含まれない、またはクライアント IP を取得できない

- Given 管理者が Application 編集画面を開いている
- And Application は OIDC / SAML / WS-Fed のいずれか 1 つのプロトコルを持つ
- When 管理者が MFA 必須と再認証を求めるまでの時間（秒）を指定したサインインポリシーを保存する
- Then AppSignInPolicyUpdated が発行される
- When 単要素セッションの利用者が対象 Application にアクセスする
- Then クライアント IP が許可 CIDR に含まれない、またはクライアント IP を取得できない
- Then フェデレーションを拒否し、AppAccessDeniedByPolicy を発行する

## Rule: REQ-APPLICATION-010 管理者はテナントデフォルトサインインポリシーを設定し全アプリに適用できる

### Example: EX-APPLICATION-010-01 通常経路

- Given ロール=["admin"] のユーザー "operator" がサインインポリシー画面を開いている
- And テナントに OIDC プロトコルを持つ複数の Application が存在し、いずれも個別のサインインポリシーを持たない
- When 管理者が MFA 必須、将来の適用開始日時、登録を一時的に迂回できる猶予期間、管理者承認を指定したテナントのデフォルトサインインポリシーを保存する
- Then TenantDefaultSignInPolicyUpdated が発行される
- Then 画面は有効なユーザーの MFA 未登録人数と、適用時に利用できなくなるユーザーへの影響を表示する
- When 単要素セッションの利用者が個別ポリシーを持たない Application にアクセスする
- Then システムはデフォルトポリシーを適用しステップアップ認証を要求する
- When 管理者が対象 Application にデフォルトより弱いサインインポリシーを保存する
- Then システムはデフォルトより弱い旨の警告を表示するが保存を許可する
- Then 当該 Application では弱いポリシーを適用し、他の Application ではデフォルトの MFA 必須を引き続き適用する
- When 管理者が Application の編集画面を開く
- Then 画面はテナントデフォルト・この Application の上書き・最終的に適用されるポリシーを区別して表示する

### Example: EX-APPLICATION-010-02 管理者が規則を空にして保存する

- Given ロール=["admin"] のユーザー "operator" がサインインポリシー画面を開いている
- And テナントに OIDC プロトコルを持つ複数の Application が存在し、いずれも個別のサインインポリシーを持たない
- When 管理者が MFA 必須、将来の適用開始日時、登録を一時的に迂回できる猶予期間、管理者承認を指定したテナントのデフォルトサインインポリシーを保存する
- But 管理者が規則を空にして保存する
- Then TenantDefaultSignInPolicyUpdated を発行し、独自ポリシーを持たない Application のフェデレーションに追加要件を課さない
