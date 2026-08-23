# Application Scenarios

### REQ-APPLICATION-001: 管理者はアプリケーション詳細で IdMagic 側の連携設定を確認できる
- ACTOR TenantAdministrator
- GIVEN OIDC または SAML プロトコルを持つ Application が存在する
- WHEN 管理者が Application の詳細画面を開く
- THEN 画面は IdMagic に登録済みの RP / SP 情報と、接続先に設定する IdMagic の Discovery またはメタデータを区別して表示する
- THEN OIDC アプリケーションには Discovery URL と `client_id` を表示する
- THEN SAML アプリケーションには IdP メタデータ URL、entityID、SSO URL、SLO URL、署名証明書を表示する
- THEN クライアントシークレットは、作成、互換ローテーション、追加発行に成功したときのレスポンス以外には表示しない

### REQ-APPLICATION-002: 管理者は通常設定とは独立したセクションでクライアントシークレットを管理できる
- ACTOR TenantAdministrator
- GIVEN シークレットを使う OIDC プロトコルを持つ Application が存在する
- GIVEN 有効期限のない従来の資格情報が 1 件 `Active` である
- WHEN 管理者が Application の編集画面を開く
- THEN `client_id` は通常の OIDC 設定カード内に参照項目として表示される
- THEN 資格情報の一覧、追加発行、個別失効の操作は、通常設定の保存フォーム外にある専用の最上位セクションに表示される
- WHEN 管理者が 90 日の有効期限を選んで新しいシークレットを追加発行する
  - ALT `Active` の資格情報がすでに 2 件存在する → 追加発行操作を無効にし、先に既存の資格情報を失効するよう案内する
- THEN 新しいシークレットは一度だけ表示され、一覧には作成日、有効期限、`Active` ステータスが表示される
- WHEN 管理者が以前の資格情報を個別に失効する
  - ALT 資格情報が `Expired` または `Revoked` である → 個別失効操作を表示しない
- THEN その資格情報だけが `Revoked` ステータスになる

### REQ-APPLICATION-003: API トークン発行者は account スコープで自分のポータルアプリケーションだけを操作できる
- ACTOR SelfApiClient
- GIVEN クライアントは対象テナントの `active` User に固定された有効な API アクセストークンを提示している
- WHEN クライアントが `account:read` スコープで、自分に割り当てられたアプリケーションと保存済みの順序をリクエストする
  - ALT トークンのテナントまたは `user_id` が操作対象と一致しない → 操作を AccessDeniedError で拒否する
- THEN クライアント自身のアプリケーションと保存済みの順序だけが返る
- WHEN クライアントが `account:write` スコープで、自分のアプリケーション順序の保存をリクエストする
  - ALT クライアントが `account:read` スコープだけを持つ → 操作を AccessDeniedError で拒否する
- THEN クライアント自身のアプリケーション順序が保存される

### REQ-APPLICATION-004: 管理 API クライアントは Application スコープで許可された操作だけを実行できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- WHEN クライアントが Application、カテゴリ、割り当て、またはテナントのデフォルトサインインポリシーに対する操作をリクエストする
  - ALT `applications:read` だけで Application の変更をリクエストする → 操作を AccessDeniedError で拒否する
  - ALT トークンのテナントとリクエスト先のテナントが一致しない → 操作を AccessDeniedError で拒否する
- THEN `applications:read` スコープでは Application、カテゴリ、割り当ての参照だけを許可する
- THEN `applications:write` スコープでは Application の作成、プロトコル設定の更新、削除を許可する
- THEN `settings:read` または `settings:write` スコープでは、テナントのデフォルトサインインポリシーに対する対応種別の操作だけを許可する

### REQ-APPLICATION-005: 管理者は Application の SAML プロトコル設定を更新できる
- ACTOR TenantAdministrator
- GIVEN 管理者が SAML プロトコルを持つ Application の編集画面を開いている
- WHEN 管理者が ACS URL、署名方針、クレーム規則、IdP プロファイルの割り当てを更新する
  - ALT AuthnRequest 署名必須だが検証可能な証明書がない → 更新は InvalidRequestError で拒否される
- THEN SAML サービスプロバイダー設定だけが同じテナント内で更新される

### REQ-APPLICATION-006: 管理者は Application ごとに公開するクレームを制限できる
- ACTOR TenantAdministrator
- GIVEN 同じテナントに OIDC Application "payroll" と "directory" が存在し、どちらも `employee_number`（`visibility=SelfReadable`）を含む同じ User 属性を参照できる
- WHEN 管理者が "payroll" のクレーム公開規則に `claim_type="employee_number"`、`source=user_attribute`、`source_key="employee_number"` を追加して保存する
  - ALT 管理者が `visibility=Private` の属性（パスワード関連の内部属性など）を `source_key` に指定する → 更新を InvalidRequestError で拒否する（`claim_release_rules_within_floor`）
  - ALT 管理者が予約済みのクレーム型（`sub`、`iss` など）を `claim_type` に指定する → 更新を InvalidRequestError で拒否する（`claim_release_rules_within_floor`）
- THEN "ApplicationClaimMappingUpdated" が発行される
- THEN "payroll" 向けに発行される ID Token / Assertion には `employee_number` クレームが含まれる
- THEN "directory" は自身の規則を更新していないため、`employee_number` クレームを含まない

### REQ-APPLICATION-007: 管理者は管理画面でアプリケーションと 1 つのプロトコルを構成できる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のアプリケーション一覧を開いている
- WHEN 管理者 "operator" が confidential アプリケーション "portal"（`type=oidc`）を作成する
- THEN 作成レスポンスだけが、生成された `client_secret` を一度だけ含む
- WHEN 管理者 "operator" がアプリケーション "portal" の OIDC 設定（`redirect_uris` / `scope`）を編集する
- THEN OIDC 設定が保存される
- WHEN 管理者 "operator" がアプリケーション "portal" をユーザー "alice" に割り当てる
  - ALT 別テナントの主体または存在しない主体を指定する → InvalidRequestError で拒否される
- THEN "alice" への割当が保存される
- WHEN 管理者 "operator" がアプリケーション "portal" を取得する
  - ALT 別テナントの管理者が同じ ID を指定する → InvalidRequestError で拒否される
- THEN 同一テナントのアプリケーションだけが返る
- WHEN 管理者 "operator" がアプリケーション "portal" を削除する
- THEN "ApplicationCreated"、"ApplicationAssigned"、"ApplicationDeleted" が発行される

### REQ-APPLICATION-008: 管理者は Application のアイコンをアップロード・削除できる
- ACTOR TenantAdministrator
- GIVEN 管理者が Application 編集画面を開いている
- WHEN 管理者が PNG / JPEG / WebP / GIF の 256KiB 以下の画像をアップロードする
  - ALT 非画像または上限超過ファイルをアップロードする → InvalidRequestError で拒否され、既存アイコンは置き換わらない
- THEN Application は `icon_object_key` と内部の `icon_url` を持つ
- WHEN 管理一覧、詳細、利用者ポータルが `icon_url` を取得する
  - ALT 別テナントの `application_id` と ID で同じアイコンを取得する → アセットは存在しないものとして扱い、InvalidRequestError で拒否する
- THEN `icon_url` は IdP の配信 URL を指す
- WHEN 管理者がアイコンを削除する
- THEN `icon_object_key` と `icon_url` は空になる

### REQ-APPLICATION-009: 管理者はアプリケーション別サインインポリシーを設定できる
- ACTOR TenantAdministrator
- GIVEN 管理者が Application 編集画面を開いている
- GIVEN Application は OIDC / SAML / WS-Fed のいずれか 1 つのプロトコルを持つ
- WHEN 管理者が MFA 必須と再認証を求めるまでの時間（秒）を指定したサインインポリシーを保存する
  - ALT 管理者以外がポリシーを更新する → AccessDeniedError で拒否される
- THEN AppSignInPolicyUpdated が発行される
- WHEN 単要素セッションの利用者が対象 Application にアクセスする
- THEN システムはトークン / Assertion の発行前にポリシーを評価する（強制点は OAuth2.Authorize）
  - ALT クライアント IP が許可 CIDR に含まれない、またはクライアント IP を取得できない → フェデレーションを拒否し、AppAccessDeniedByPolicy を発行する
- THEN ステップアップ認証が可能な経路ではステップアップ認証を要求し、認証強度を上げた後にフェデレーションを完了する

### REQ-APPLICATION-010: 管理者はテナントデフォルトサインインポリシーを設定し全アプリに適用できる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" がサインインポリシー画面を開いている
- GIVEN テナントに OIDC プロトコルを持つ複数の Application が存在し、いずれも個別のサインインポリシーを持たない
- WHEN 管理者が MFA 必須、将来の適用開始日時、登録を一時的に迂回できる猶予期間、管理者承認を指定したテナントのデフォルトサインインポリシーを保存する
  - ALT 管理者が規則を空にして保存する → TenantDefaultSignInPolicyUpdated を発行し、独自ポリシーを持たない Application のフェデレーションに追加要件を課さない
- THEN TenantDefaultSignInPolicyUpdated が発行される
- THEN 画面は有効なユーザーの MFA 未登録人数と、適用時に利用できなくなるユーザーへの影響を表示する
- WHEN 単要素セッションの利用者が個別ポリシーを持たない Application にアクセスする
- THEN システムはデフォルトポリシーを適用しステップアップ認証を要求する
- WHEN 管理者が対象 Application にデフォルトより弱いサインインポリシーを保存する
- THEN システムはデフォルトより弱い旨の警告を表示するが保存を許可する
- THEN 当該 Application では弱いポリシーを適用し、他の Application ではデフォルトの MFA 必須を引き続き適用する
- WHEN 管理者が Application の編集画面を開く
- THEN 画面はテナントデフォルト・この Application の上書き・最終的に適用されるポリシーを区別して表示する

### REQ-APPLICATION-011: 割り当てのない主体はプロトコル経由でアプリケーションへフェデレーションできない
- ACTOR TenantAdministrator
- GIVEN アプリケーション "portal" にユーザー "alice" は割り当てられていない
- WHEN "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
  - ALT 管理者が事前に "portal" へ "alice" を `visibility=visible` で割り当てる → "alice" はフェデレーションを完了できる
- THEN 割り当てがないため、フェデレーションを拒否する

### REQ-APPLICATION-012: hidden の割り当てはポータル一覧から除外するがプロトコルの利用は許可する
- ACTOR TenantAdministrator
- GIVEN 管理者が "portal" にユーザー "alice" を `visibility=hidden` で割り当てている
- WHEN "alice" が自分のポータルアプリケーション一覧（ListMyApplications）を取得する
- THEN 一覧に "portal" は含まれない
- WHEN "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
- THEN `hidden` の割り当てがあるため、フェデレーションを完了できる

### REQ-APPLICATION-013: admin ロールを持たない利用者は Application を操作できない
- ACTOR AuthenticatedSelf
- GIVEN "alice" は `admin` ロールを持たない認証済みユーザーである
- WHEN "alice" が ListAdminApplications を呼び出す
- THEN AccessDeniedError で拒否される

### REQ-APPLICATION-014: あるべき状態を指定した割り当てはグループ割り当てを変更しない
- ACTOR TenantAdministrator
- GIVEN "alice" は動的グループを介して "portal" へのグループ割り当て（`subject_type=group`）をすでに持つ
- WHEN IdManagement の LifecycleWorkflow が "alice" に対して AssignApplicationDesiredState を呼び出す
  - ALT 個人への直接割り当てが指定どおりの `visibility` ですでに存在する → 変更せずに `changed=false` を返す
- THEN "alice" 個人への直接ユーザー割り当て（`subject_type=user`）が作成される
- THEN グループ割り当て（`subject_type=group`）の行は変更されない
- WHEN LifecycleWorkflow が後から UnassignApplicationDesiredState を呼び出す
- THEN 直接ユーザー割り当てだけが削除され、グループ割り当ては残る
- THEN フェデレーションは引き続き許可される
