# Feature: アプリケーションのカタログの例

## Rule: REQ-APPLICATION-001 管理者はアプリケーション詳細で IdMagic 側の連携設定を確認できる

### Example: EX-APPLICATION-001-01 通常経路

- Given OIDC または SAML プロトコルを持つ Application が存在する
- When 管理者が Application の詳細画面を開く
- Then 画面は IdMagic に登録済みの RP / SP 情報と、接続先に設定する IdMagic の Discovery またはメタデータを区別して表示する
- Then OIDC アプリケーションには Discovery URL と `client_id` を表示する
- Then SAML アプリケーションには IdP メタデータ URL、entityID、SSO URL、SLO URL、署名証明書を表示する
- Then クライアントシークレットは、作成、互換ローテーション、追加発行に成功したときのレスポンス以外には表示しない

## Rule: REQ-APPLICATION-002 管理者は通常設定とは独立したセクションでクライアントシークレットを管理できる

### Background:

- Given シークレットを使う OIDC プロトコルを持つ Application が存在する
- And 有効期限のない従来の資格情報が 1 件 `Active` である

### Example: EX-APPLICATION-002-01 通常経路

- When 管理者が Application の編集画面を開く
- Then `client_id` は通常の OIDC 設定カード内に参照項目として表示される
- Then 資格情報の一覧、追加発行、個別失効の操作は、通常設定の保存フォーム外にある専用の最上位セクションに表示される
- When 管理者が 90 日の有効期限を選んで新しいシークレットを追加発行する
- Then 新しいシークレットは一度だけ表示され、一覧には作成日、有効期限、`Active` ステータスが表示される
- When 管理者が以前の資格情報を個別に失効する
- Then その資格情報だけが `Revoked` ステータスになる

### Example: EX-APPLICATION-002-02 `Active` の資格情報がすでに 2 件存在する

- When 管理者が Application の編集画面を開く
- Then `client_id` は通常の OIDC 設定カード内に参照項目として表示される
- Then 資格情報の一覧、追加発行、個別失効の操作は、通常設定の保存フォーム外にある専用の最上位セクションに表示される
- When 管理者が 90 日の有効期限を選んで新しいシークレットを追加発行する
- But `Active` の資格情報がすでに 2 件存在する
- Then 追加発行操作を無効にし、先に既存の資格情報を失効するよう案内する

### Example: EX-APPLICATION-002-03 資格情報が `Expired` または `Revoked` である

- When 管理者が Application の編集画面を開く
- Then `client_id` は通常の OIDC 設定カード内に参照項目として表示される
- Then 資格情報の一覧、追加発行、個別失効の操作は、通常設定の保存フォーム外にある専用の最上位セクションに表示される
- When 管理者が 90 日の有効期限を選んで新しいシークレットを追加発行する
- Then 新しいシークレットは一度だけ表示され、一覧には作成日、有効期限、`Active` ステータスが表示される
- When 管理者が以前の資格情報を個別に失効する
- But 資格情報が `Expired` または `Revoked` である
- Then 個別失効操作を表示しない

## Rule: REQ-APPLICATION-004 管理 API クライアントは Application スコープで許可された操作だけを実行できる

### Example: EX-APPLICATION-004-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが Application、カテゴリ、割り当て、またはテナントのデフォルトサインインポリシーに対する操作をリクエストする
- Then `applications:read` スコープでは Application、カテゴリ、割り当ての参照だけを許可する
- Then `applications:write` スコープでは Application の作成、プロトコル設定の更新、削除を許可する
- Then `settings:read` または `settings:write` スコープでは、テナントのデフォルトサインインポリシーに対する対応種別の操作だけを許可する

### Example: EX-APPLICATION-004-02 `applications:read` だけで Application の変更をリクエストする

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが Application、カテゴリ、割り当て、またはテナントのデフォルトサインインポリシーに対する操作をリクエストする
- But `applications:read` だけで Application の変更をリクエストする
- Then 操作を 403 と `insufficient_scope` で拒否する

### Example: EX-APPLICATION-004-03 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは発行元テナントでは有効な Application スコープの API アクセストークンを持つ
- When クライアントがそのトークンを別テナントの Application admin API へ提示する
- Then 操作を 401 の InvalidAccessTokenError で拒否する

## Rule: REQ-APPLICATION-005 管理者は Application の SAML プロトコル設定を更新できる

### Example: EX-APPLICATION-005-01 通常経路

- Given 管理者が SAML プロトコルを持つ Application の編集画面を開いている
- When 管理者が ACS URL、署名方針、クレーム規則、IdP プロファイルの割り当てを更新する
- Then SAML サービスプロバイダー設定だけが同じテナント内で更新される

### Example: EX-APPLICATION-005-02 AuthnRequest 署名必須だが検証可能な証明書がない

- Given 管理者が SAML プロトコルを持つ Application の編集画面を開いている
- When 管理者が ACS URL、署名方針、クレーム規則、IdP プロファイルの割り当てを更新する
- But AuthnRequest 署名必須だが検証可能な証明書がない
- Then 更新は InvalidRequestError で拒否される

## Rule: REQ-APPLICATION-006 管理者は Application ごとに公開するクレームを制限できる

### Example: EX-APPLICATION-006-01 通常経路

- Given 同じテナントに OIDC Application "payroll" と "directory" が存在し、どちらも `employee_number`（`visibility=SelfReadable`）を含む同じ User 属性を参照できる
- When 管理者が "payroll" のクレーム公開規則に `claim_type="employee_number"`、`source=user_attribute`、`source_key="employee_number"` を追加して保存する
- Then "ApplicationClaimMappingUpdated" が発行される
- Then "payroll" 向けに発行される ID Token / Assertion には `employee_number` クレームが含まれる
- Then "directory" は自身の規則を更新していないため、`employee_number` クレームを含まない

### Scenario Outline: 条件ごとの結果

- Given 同じテナントに OIDC Application "payroll" と "directory" が存在し、どちらも `employee_number`（`visibility=SelfReadable`）を含む同じ User 属性を参照できる
- When 管理者が "payroll" のクレーム公開規則に `claim_type="employee_number"`、`source=user_attribute`、`source_key="employee_number"` を追加して保存する
- But <condition>
- Then 更新を InvalidRequestError で拒否する（`claim_release_rules_within_floor`）

#### Examples:

  | example_id | condition |
  | --- | --- |
  | EX-APPLICATION-006-02 | 管理者が `visibility=Private` の属性（パスワード関連の内部属性など）を `source_key` に指定する |
  | EX-APPLICATION-006-03 | 管理者が予約済みのクレーム型（`sub`、`iss` など）を `claim_type` に指定する |

## Rule: REQ-APPLICATION-007 管理者は管理画面でアプリケーションと 1 つのプロトコルを構成できる

### Example: EX-APPLICATION-007-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面のアプリケーション一覧を開いている
- When 管理者 "operator" が confidential アプリケーション "portal"（`type=oidc`）を作成する
- Then 作成レスポンスだけが、生成された `client_secret` を一度だけ含む
- When 管理者 "operator" がアプリケーション "portal" の OIDC 設定（`redirect_uris` / `scope`）を編集する
- Then OIDC 設定が保存される
- When 管理者 "operator" がアプリケーション "portal" をユーザー "alice" に割り当てる
- Then "alice" への割当が保存される
- When 管理者 "operator" がアプリケーション "portal" を取得する
- Then 同一テナントのアプリケーションだけが返る
- When 管理者 "operator" がアプリケーション "portal" を削除する
- Then "ApplicationCreated"、"ApplicationAssigned"、"ApplicationDeleted" が発行される

### Example: EX-APPLICATION-007-02 別テナントの主体または存在しない主体を指定する

- Given ロール=["admin"] のユーザー "operator" が管理画面のアプリケーション一覧を開いている
- When 管理者 "operator" が confidential アプリケーション "portal"（`type=oidc`）を作成する
- Then 作成レスポンスだけが、生成された `client_secret` を一度だけ含む
- When 管理者 "operator" がアプリケーション "portal" の OIDC 設定（`redirect_uris` / `scope`）を編集する
- Then OIDC 設定が保存される
- When 管理者 "operator" がアプリケーション "portal" をユーザー "alice" に割り当てる
- But 別テナントの主体または存在しない主体を指定する
- Then InvalidRequestError で拒否される

### Example: EX-APPLICATION-007-03 別テナントの管理者が同じ ID を指定する

- Given ロール=["admin"] のユーザー "operator" が管理画面のアプリケーション一覧を開いている
- When 管理者 "operator" が confidential アプリケーション "portal"（`type=oidc`）を作成する
- Then 作成レスポンスだけが、生成された `client_secret` を一度だけ含む
- When 管理者 "operator" がアプリケーション "portal" の OIDC 設定（`redirect_uris` / `scope`）を編集する
- Then OIDC 設定が保存される
- When 管理者 "operator" がアプリケーション "portal" をユーザー "alice" に割り当てる
- Then "alice" への割当が保存される
- When 管理者 "operator" がアプリケーション "portal" を取得する
- But 別テナントの管理者が同じ ID を指定する
- Then アプリケーションは存在しないものとして扱われ、応答は存在しない ID を指定したときと同じ 404 application_not_found である
- Then 応答にアプリケーション "portal" の名称と OIDC 設定は含まれない

## Rule: REQ-APPLICATION-008 管理者は Application のアイコンをアップロード・削除できる

### Example: EX-APPLICATION-008-01 通常経路

- Given 管理者が Application 編集画面を開いている
- When 管理者が PNG / JPEG / WebP / GIF の 256KiB 以下の画像をアップロードする
- Then Application は `icon_object_key` と内部の `icon_url` を持つ
- When 管理一覧、詳細、利用者ポータルが `icon_url` を取得する
- Then `icon_url` は IdP の配信 URL を指す
- When 管理者がアイコンを削除する
- Then `icon_object_key` と `icon_url` は空になる

### Example: EX-APPLICATION-008-02 非画像または上限超過ファイルをアップロードする

- Given 管理者が Application 編集画面を開いている
- When 管理者が PNG / JPEG / WebP / GIF の 256KiB 以下の画像をアップロードする
- But 非画像または上限超過ファイルをアップロードする
- Then 400 と `invalid_icon` で拒否され、既存アイコンは置き換わらない

### Example: EX-APPLICATION-008-03 別テナントの `application_id` と ID で同じアイコンを取得する

- Given 管理者が Application 編集画面を開いている
- When 管理者が PNG / JPEG / WebP / GIF の 256KiB 以下の画像をアップロードする
- Then Application は `icon_object_key` と内部の `icon_url` を持つ
- When 管理一覧、詳細、利用者ポータルが `icon_url` を取得する
- But 別テナントの `application_id` と ID で同じアイコンを取得する
- Then アイコンは存在しないものとして扱われ、応答は存在しない ID を指定したときと同じ 404 not_found である
- Then 応答にアップロードした画像の内容は含まれない

## Rule: REQ-APPLICATION-013 admin ロールを持たない利用者は Application を操作できない

### Example: EX-APPLICATION-013-01 通常経路

- Given "alice" は `admin` ロールを持たない認証済みユーザーである
- When "alice" が ListAdminApplications を呼び出す
- Then AccessDeniedError で拒否される
