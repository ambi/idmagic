# Feature: Application のシナリオ

## Rule: REQ-APPLICATION-001 管理者はアプリケーション詳細で IdMagic 側の連携設定を確認できる

Primary actor: `TenantAdministrator`

### Example: EX-APPLICATION-001-01 通常経路

- Given OIDC または SAML プロトコルを持つ Application が存在する
- When 管理者が Application の詳細画面を開く
- Then 画面は IdMagic に登録済みの RP / SP 情報と、接続先に設定する IdMagic の Discovery またはメタデータを区別して表示する
- Then OIDC アプリケーションには Discovery URL と `client_id` を表示する
- Then SAML アプリケーションには IdP メタデータ URL、entityID、SSO URL、SLO URL、署名証明書を表示する
- Then クライアントシークレットは、作成、互換ローテーション、追加発行に成功したときのレスポンス以外には表示しない

## Rule: REQ-APPLICATION-002 管理者は通常設定とは独立したセクションでクライアントシークレットを管理できる

Primary actor: `TenantAdministrator`

### Example: EX-APPLICATION-002-01 通常経路

- Given シークレットを使う OIDC プロトコルを持つ Application が存在する
- And 有効期限のない従来の資格情報が 1 件 `Active` である
- When 管理者が Application の編集画面を開く
- Then `client_id` は通常の OIDC 設定カード内に参照項目として表示される
- Then 資格情報の一覧、追加発行、個別失効の操作は、通常設定の保存フォーム外にある専用の最上位セクションに表示される
- When 管理者が 90 日の有効期限を選んで新しいシークレットを追加発行する
- Then 新しいシークレットは一度だけ表示され、一覧には作成日、有効期限、`Active` ステータスが表示される
- When 管理者が以前の資格情報を個別に失効する
- Then その資格情報だけが `Revoked` ステータスになる

### Example: EX-APPLICATION-002-02 `Active` の資格情報がすでに 2 件存在する

- Given シークレットを使う OIDC プロトコルを持つ Application が存在する
- And 有効期限のない従来の資格情報が 1 件 `Active` である
- When 管理者が Application の編集画面を開く
- Then `client_id` は通常の OIDC 設定カード内に参照項目として表示される
- Then 資格情報の一覧、追加発行、個別失効の操作は、通常設定の保存フォーム外にある専用の最上位セクションに表示される
- When 管理者が 90 日の有効期限を選んで新しいシークレットを追加発行する
- But `Active` の資格情報がすでに 2 件存在する
- Then 追加発行操作を無効にし、先に既存の資格情報を失効するよう案内する

### Example: EX-APPLICATION-002-03 資格情報が `Expired` または `Revoked` である

- Given シークレットを使う OIDC プロトコルを持つ Application が存在する
- And 有効期限のない従来の資格情報が 1 件 `Active` である
- When 管理者が Application の編集画面を開く
- Then `client_id` は通常の OIDC 設定カード内に参照項目として表示される
- Then 資格情報の一覧、追加発行、個別失効の操作は、通常設定の保存フォーム外にある専用の最上位セクションに表示される
- When 管理者が 90 日の有効期限を選んで新しいシークレットを追加発行する
- Then 新しいシークレットは一度だけ表示され、一覧には作成日、有効期限、`Active` ステータスが表示される
- When 管理者が以前の資格情報を個別に失効する
- But 資格情報が `Expired` または `Revoked` である
- Then 個別失効操作を表示しない

## Rule: REQ-APPLICATION-003 API トークン発行者は account スコープで自分のポータルアプリケーションだけを操作できる

Primary actor: `SelfApiClient`

### Example: EX-APPLICATION-003-01 通常経路

- Given クライアントは対象テナントの `active` User に固定された有効な API アクセストークンを提示している
- When クライアントが `account:read` スコープで、自分に割り当てられたアプリケーションと保存済みの順序をリクエストする
- Then クライアント自身のアプリケーションと保存済みの順序だけが返る
- When クライアントが `account:write` スコープで、自分のアプリケーション順序の保存をリクエストする
- Then クライアント自身のアプリケーション順序が保存される

### Example: EX-APPLICATION-003-02 トークンのテナントまたは `user_id` が操作対象と一致しない

- Given クライアントは対象テナントの `active` User に固定された有効な API アクセストークンを提示している
- When クライアントが `account:read` スコープで、自分に割り当てられたアプリケーションと保存済みの順序をリクエストする
- But トークンのテナントまたは `user_id` が操作対象と一致しない
- Then 操作を AccessDeniedError で拒否する

### Example: EX-APPLICATION-003-03 クライアントが `account:read` スコープだけを持つ

- Given クライアントは対象テナントの `active` User に固定された有効な API アクセストークンを提示している
- When クライアントが `account:read` スコープで、自分に割り当てられたアプリケーションと保存済みの順序をリクエストする
- Then クライアント自身のアプリケーションと保存済みの順序だけが返る
- When クライアントが `account:write` スコープで、自分のアプリケーション順序の保存をリクエストする
- But クライアントが `account:read` スコープだけを持つ
- Then 操作を AccessDeniedError で拒否する

## Rule: REQ-APPLICATION-004 管理 API クライアントは Application スコープで許可された操作だけを実行できる

Primary actor: `ManagementApiClient`

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
- Then 操作を AccessDeniedError で拒否する

### Example: EX-APPLICATION-004-03 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが Application、カテゴリ、割り当て、またはテナントのデフォルトサインインポリシーに対する操作をリクエストする
- But トークンのテナントとリクエスト先のテナントが一致しない
- Then 操作を AccessDeniedError で拒否する

## Rule: REQ-APPLICATION-005 管理者は Application の SAML プロトコル設定を更新できる

Primary actor: `TenantAdministrator`

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

Primary actor: `TenantAdministrator`

### Example: EX-APPLICATION-006-01 通常経路

- Given 同じテナントに OIDC Application "payroll" と "directory" が存在し、どちらも `employee_number`（`visibility=SelfReadable`）を含む同じ User 属性を参照できる
- When 管理者が "payroll" のクレーム公開規則に `claim_type="employee_number"`、`source=user_attribute`、`source_key="employee_number"` を追加して保存する
- Then "ApplicationClaimMappingUpdated" が発行される
- Then "payroll" 向けに発行される ID Token / Assertion には `employee_number` クレームが含まれる
- Then "directory" は自身の規則を更新していないため、`employee_number` クレームを含まない

### Example: EX-APPLICATION-006-02 管理者が `visibility=Private` の属性（パスワード関連の内部属性など）を `source_key` に指定する

- Given 同じテナントに OIDC Application "payroll" と "directory" が存在し、どちらも `employee_number`（`visibility=SelfReadable`）を含む同じ User 属性を参照できる
- When 管理者が "payroll" のクレーム公開規則に `claim_type="employee_number"`、`source=user_attribute`、`source_key="employee_number"` を追加して保存する
- But 管理者が `visibility=Private` の属性（パスワード関連の内部属性など）を `source_key` に指定する
- Then 更新を InvalidRequestError で拒否する（`claim_release_rules_within_floor`）

### Example: EX-APPLICATION-006-03 管理者が予約済みのクレーム型（`sub`、`iss` など）を `claim_type` に指定する

- Given 同じテナントに OIDC Application "payroll" と "directory" が存在し、どちらも `employee_number`（`visibility=SelfReadable`）を含む同じ User 属性を参照できる
- When 管理者が "payroll" のクレーム公開規則に `claim_type="employee_number"`、`source=user_attribute`、`source_key="employee_number"` を追加して保存する
- But 管理者が予約済みのクレーム型（`sub`、`iss` など）を `claim_type` に指定する
- Then 更新を InvalidRequestError で拒否する（`claim_release_rules_within_floor`）

## Rule: REQ-APPLICATION-007 管理者は管理画面でアプリケーションと 1 つのプロトコルを構成できる

Primary actor: `TenantAdministrator`

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
- Then InvalidRequestError で拒否される

## Rule: REQ-APPLICATION-008 管理者は Application のアイコンをアップロード・削除できる

Primary actor: `TenantAdministrator`

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
- Then InvalidRequestError で拒否され、既存アイコンは置き換わらない

### Example: EX-APPLICATION-008-03 別テナントの `application_id` と ID で同じアイコンを取得する

- Given 管理者が Application 編集画面を開いている
- When 管理者が PNG / JPEG / WebP / GIF の 256KiB 以下の画像をアップロードする
- Then Application は `icon_object_key` と内部の `icon_url` を持つ
- When 管理一覧、詳細、利用者ポータルが `icon_url` を取得する
- But 別テナントの `application_id` と ID で同じアイコンを取得する
- Then アセットは存在しないものとして扱い、InvalidRequestError で拒否する

## Rule: REQ-APPLICATION-009 管理者はアプリケーション別サインインポリシーを設定できる

Primary actor: `TenantAdministrator`

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

Primary actor: `TenantAdministrator`

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

## Rule: REQ-APPLICATION-011 割り当てのない主体はプロトコル経由でアプリケーションへフェデレーションできない

Primary actor: `TenantAdministrator`

### Example: EX-APPLICATION-011-01 通常経路

- Given アプリケーション "portal" にユーザー "alice" は割り当てられていない
- When "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
- Then 割り当てがないため、フェデレーションを拒否する

### Example: EX-APPLICATION-011-02 管理者が事前に "portal" へ "alice" を `visibility=visible` で割り当てる

- Given アプリケーション "portal" にユーザー "alice" は割り当てられていない
- When "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
- But 管理者が事前に "portal" へ "alice" を `visibility=visible` で割り当てる
- Then "alice" はフェデレーションを完了できる

## Rule: REQ-APPLICATION-012 hidden の割り当てはポータル一覧から除外するがプロトコルの利用は許可する

Primary actor: `TenantAdministrator`

### Example: EX-APPLICATION-012-01 通常経路

- Given 管理者が "portal" にユーザー "alice" を `visibility=hidden` で割り当てている
- When "alice" が自分のポータルアプリケーション一覧（ListMyApplications）を取得する
- Then 一覧に "portal" は含まれない
- When "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
- Then `hidden` の割り当てがあるため、フェデレーションを完了できる

## Rule: REQ-APPLICATION-013 admin ロールを持たない利用者は Application を操作できない

Primary actor: `AuthenticatedSelf`

### Example: EX-APPLICATION-013-01 通常経路

- Given "alice" は `admin` ロールを持たない認証済みユーザーである
- When "alice" が ListAdminApplications を呼び出す
- Then AccessDeniedError で拒否される

## Rule: REQ-APPLICATION-014 あるべき状態を指定した割り当てはグループ割り当てを変更しない

Primary actor: `TenantAdministrator`

### Example: EX-APPLICATION-014-01 通常経路

- Given "alice" は動的グループを介して "portal" へのグループ割り当て（`subject_type=group`）をすでに持つ
- When IdManagement の LifecycleWorkflow が "alice" に対して AssignApplicationDesiredState を呼び出す
- Then "alice" 個人への直接ユーザー割り当て（`subject_type=user`）が作成される
- Then グループ割り当て（`subject_type=group`）の行は変更されない
- When LifecycleWorkflow が後から UnassignApplicationDesiredState を呼び出す
- Then 直接ユーザー割り当てだけが削除され、グループ割り当ては残る
- Then フェデレーションは引き続き許可される

### Example: EX-APPLICATION-014-02 個人への直接割り当てが指定どおりの `visibility` ですでに存在する

- Given "alice" は動的グループを介して "portal" へのグループ割り当て（`subject_type=group`）をすでに持つ
- When IdManagement の LifecycleWorkflow が "alice" に対して AssignApplicationDesiredState を呼び出す
- But 個人への直接割り当てが指定どおりの `visibility` ですでに存在する
- Then 変更せずに `changed=false` を返す
