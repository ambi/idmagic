# Application Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-APPLICATION-001: 管理者はアプリ詳細でIdMagic側の連携設定を確認できる
- Actor: TenantAdministrator
- Given: 管理者が OIDC または SAML protocol を持つ Application の詳細画面を開いている
- Then: 画面は IdMagic に登録済みの RP / SP 情報と、相手側へ投入する IdMagic の discovery または metadata を区別して表示する
- Then: OIDC application には OpenID Discovery URL と client_id を表示する
- Then: SAML application には IdP metadata URL、entityID、SSO URL、SLO URL、署名証明書を表示する
- Then: client secret は作成・互換ローテーション・追加発行の成功応答以外では表示しない

### REQ-APPLICATION-002: 管理者は通常設定と独立したセクションでclient secretを管理できる
- Actor: TenantAdministrator
- Given: 管理者が secret-based OIDC protocol を持つ Application の編集画面を開いている
- Given: 期限なし legacy credential が1件 Active である
- Then: client_id は通常の OIDC 設定カード内に参照項目として表示される
- Then: credential 一覧と追加発行・個別失効操作は通常設定の保存 form 外にある専用トップレベルセクションへ表示される
- Then: 管理者が90日の期限を選んで新 secret を追加発行する
- Then: 新 secret は一度だけ表示され、一覧には作成日・有効期限・Active 状態が表示される
- Then: 管理者が旧 credential を個別失効すると、その行だけが Revoked 状態になる
- Alternative (Active credential が既に2件存在する): 追加発行操作は利用不可となり、先に既存 credential を失効する案内を表示する
- Alternative (credential が Expired または Revoked である): 個別失効操作は表示しない

### REQ-APPLICATION-003: API token発行者はaccount scope内で自身のportal applicationだけを操作できる
- Actor: SelfApiClient
- Given: client は対象 tenant の active User に固定された有効な API access token を提示している
- Then: account:read scope で自身に割り当てられた application と保存済み順序を参照できる
- Then: account:write scope で自身の application 順序を保存できる
- Alternative (account:read だけで順序変更を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant または user_id が操作対象と一致しない): 操作は AccessDeniedError で拒否される

### REQ-APPLICATION-004: management API clientはApplication scope内の操作だけを実行できる
- Actor: ManagementApiClient
- Given: client は対象 tenant の有効な API access token を提示している
- Then: applications:read scope で Application と category と assignment を参照できる
- Then: applications:write scope で Application の作成・protocol 設定更新・削除ができる
- Then: settings:read または settings:write scope で tenant default sign-in policy を対応する操作種別に限って扱える
- Alternative (applications:read だけで変更操作を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant と request tenant が一致しない): 操作は AccessDeniedError で拒否される

### REQ-APPLICATION-005: 管理者はApplicationのSAML protocol設定を更新できる
- Actor: TenantAdministrator
- Given: 管理者が SAML protocol を持つ Application の編集画面を開いている
- Then: 管理者が ACS URL、署名方針、claim 規則、IdP profile割当を更新する
- Then: SAML service provider 設定だけが同一テナント内で更新される
- Alternative (AuthnRequest 署名必須だが検証可能な証明書がない): 更新は InvalidRequestError で拒否される

### REQ-APPLICATION-006: 管理者はApplication単位でclaim releaseを絞り込める
- Actor: TenantAdministrator
- Given: 同一テナントに OIDC Application "payroll" と "directory" が存在し、いずれも employee_number (visibility=SelfReadable) を含む同じ User 属性を参照できる
- Then: 管理者が "payroll" の claim release rule に claim_type="employee_number"、source=user_attribute、source_key="employee_number" を追加して保存する
- Then: "ApplicationClaimMappingUpdated" が発行される
- Then: "payroll" 向けに発行される ID Token / assertion には employee_number claim が含まれる
- Then: "directory" は自身の rule を更新していないため employee_number claim を含まない
- Alternative (管理者が visibility=Private の属性 (例 password 関連の内部属性) を source_key に指定する): 更新は InvalidRequestError で拒否される (claim_release_rules_within_floor)
- Alternative (管理者が reserved claim type (例 \"sub\"、\"iss\") を claim_type に指定する): 更新は InvalidRequestError で拒否される (claim_release_rules_within_floor)

### REQ-APPLICATION-007: 管理者は管理画面でアプリケーションと単一protocolを構成できる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が管理画面のアプリケーション一覧を開いている
- Then: 管理者 "operator" が confidential なアプリケーション "portal" (type=oidc) を作成する
- Then: 作成応答だけが生成された client_secret を一度だけ含む
- Then: 管理者 "operator" がアプリケーション "portal" の OIDC 設定 (redirect_uris / scope) を編集する
- Then: 管理者 "operator" がアプリケーション "portal" をユーザー "alice" に割り当てる
- Then: 管理者 "operator" がアプリケーション "portal" を削除する
- Then: "ApplicationCreated"、"ApplicationAssigned"、"ApplicationDeleted" が発行される
- Alternative (別テナントまたは存在しない subject へ割り当てる): tenant_id "acme" の Application に tenant_id "default" の User sub を指定する → エラー "InvalidRequestError"
- Alternative (別テナントの管理者がアプリケーションを取得しようとする): tenant_id "acme" に Application "portal" が存在する → tenant_id "default" の管理者が同じ id を取得する → エラー "InvalidRequestError"

### REQ-APPLICATION-008: 管理者はApplicationアイコンをアップロード削除できる
- Actor: TenantAdministrator
- Given: 管理者が Application 編集画面を開いている
- Then: PNG / JPEG / WebP / GIF の 256KiB 以下の画像をアップロードする
- Then: Application は icon_object_key と内部 icon_url を持つ
- Then: 管理一覧・詳細・利用者ポータルの icon_url は IdP の配信 URL を指す
- Then: 管理者がアイコンを削除すると icon_object_key と icon_url は空になる
- Alternative (非画像または上限超過ファイルをアップロードする): エラー "InvalidRequestError" → 既存アイコンは置き換わらない
- Alternative (別テナントの application_id + id で同じアイコンの取得を試みる): アセットは存在しないものとして扱われ InvalidRequestError で拒否される

### REQ-APPLICATION-009: 管理者はアプリケーション別サインインポリシーを設定できる
- Actor: TenantAdministrator
- Given: 管理者が Application 編集画面を開いている
- Given: Application は OIDC / SAML / WS-Fed のいずれか単一の protocol を持つ
- Then: 管理者が MFA 必須と再認証を求めるまでの時間 (秒) を要求する sign-in policy を保存する
- Then: AppSignInPolicyUpdated が発行される
- Then: 単要素セッションの利用者が対象 Application にアクセスする
- Then: システムは step-up が可能な経路では step-up を要求し、昇格後に federation を完了する
- Then: システムは token / assertion 発行前に policy を評価する (強制点は OAuth2.Authorize)
- Alternative (許可 CIDR に含まれないクライアント IP、またはクライアント IP を取得できない): 対象 Application への federation は拒否される → AppAccessDeniedByPolicy が発行される
- Alternative (管理者以外が policy を更新する): エラー "AccessDeniedError"

### REQ-APPLICATION-010: 管理者はテナントデフォルトサインインポリシーを設定し全アプリに適用できる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" がサインインポリシー画面を開いている
- Given: テナントに OIDC protocol を持つ複数の Application が存在し、いずれも個別 sign-in policy を持たない
- Then: 管理者が MFA 必須、将来の強制開始日時、enrollment bypass 猶予、管理者承認を要求するテナントデフォルトサインインポリシーを保存する
- Then: TenantDefaultSignInPolicyUpdated が発行される
- Then: 画面は active user の MFA 未登録人数と強制時のロックアウト影響を表示する
- Then: 単要素セッションの利用者が個別ポリシーを持たない Application にアクセスする
- Then: システムはデフォルトポリシーを適用し step-up を要求する
- Alternative (低リスクアプリだけデフォルトを上書きして緩和する): 管理者が対象 Application に弱いサインインポリシー (例: パスワードのみ) を設定してデフォルトを上書きする → システムはデフォルトより弱い旨の警告を表示するが保存は許可する → 単要素セッションの利用者が当該 Application に step-up なしでアクセスできる → 他の Application ではデフォルトの MFA 必須が引き続き適用される
- Alternative (アプリ詳細で最終適用ポリシーを確認する): アプリ編集画面はテナントデフォルト・このアプリの上書き・最終的に適用されるポリシーを区別して表示する
- Alternative (デフォルトポリシーを空にして解除する): 管理者がルールを空にして保存する → TenantDefaultSignInPolicyUpdated が発行される → 以降、独自ポリシーを持たない Application の federation 開始は追加要件を課さない

### REQ-APPLICATION-011: 未割当のsubjectはprotocol経由でアプリケーションへフェデレーションできない
- Actor: TenantAdministrator
- Given: アプリケーション "portal" にユーザー "alice" は割り当てられていない
- Then: "alice" が "portal" への federation を試みる (強制点は OAuth2.Authorize)
- Then: 未割当のため federation は拒否される
- Alternative (管理者が事後に visible で割り当てる): 管理者が "portal" に "alice" を visibility=visible で割り当てる → "alice" は "portal" への federation を完了できる

### REQ-APPLICATION-012: hidden割当はポータル一覧から除外されるがprotocol利用は許可される
- Actor: TenantAdministrator
- Given: 管理者が "portal" にユーザー "alice" を visibility=hidden で割り当てている
- Then: "alice" が自分のポータルアプリ一覧 (ListMyApplications) を取得する
- Then: 一覧に "portal" は含まれない
- Then: "alice" は "portal" への federation は引き続き完了できる (強制点は OAuth2.Authorize)

### REQ-APPLICATION-013: adminロールを持たない利用者はApplicationを操作できない
- Actor: AuthenticatedSelf
- Given: "alice" は admin ロールを持たない認証済みユーザーである
- Then: "alice" が ListAdminApplications を呼び出す
- Then: AccessDeniedError で拒否される

### REQ-APPLICATION-014: desired-state割当はgroup経由の割当を変更しない
- Actor: TenantAdministrator
- Given: IdManagement の LifecycleWorkflow が "alice" に対して AssignApplicationDesiredState を呼び出す
- Given: "alice" は dynamic group 経由で "portal" への group assignment (subject_type=group) を既に持つ
- Then: AssignApplicationDesiredState が "alice" 個人への direct user assignment (subject_type=user) を新規作成する
- Then: group assignment (subject_type=group) の行は変更されない
- Then: 後から UnassignApplicationDesiredState を呼んでも group assignment は残り、federation は引き続き許可される
- Alternative (個人の direct assignment が既に指定どおりの visibility で存在する): AssignApplicationDesiredState は変更を行わず changed=false を返す

### REQ-APPLICATION-015: AssignApplicationDesiredState
呼び出し元 bounded context (IdManagement の LifecycleWorkflow 等) が Application への
user 割当を desired-state で付与する内部インターフェース。HTTP には公開せず、同一プロセス内の
Go 呼び出しとして各 context の usecase から使う。既に同じ (id, user_id) の割当が
指定どおりの visibility で存在する場合は変更せず changed=false を返す (冪等)。呼び出し元は
同一テナントの id / user_id だけを渡す。

### REQ-APPLICATION-016: UnassignApplicationDesiredState
呼び出し元 bounded context が Application への user 割当を desired-state で解除する内部
インターフェース。割当が存在しない場合も no-op (changed=false) で正常終了する (冪等)。

### REQ-APPLICATION-017: ListAdminApplications
管理者が所属テナントの Application を name 昇順 (同値は id で tie-break) の双方向 keyset
pagination で一覧する。cursor は応答の Link response header (rel="prev" / rel="next") から取得する。

### REQ-APPLICATION-018: GetAdminApplication
管理者が Application を取得する。protocol relation から解決した実設定も含む。OIDC / SAML application の詳細画面は、この登録済み相手側設定と Tenancy.GetAdminIntegrationEndpoints が返す IdMagic 側の setup guidance を並べて表示し、client secret は再表示しない。別テナントは未存在として扱う。

### REQ-APPLICATION-019: CreateAdminApplication
管理者が Application を一括作成する。種別に応じた protocol row を準備し、Application と application_id relation を同一 transaction で catalog へ公開する。protocol 種別は作成後に変更できない。

### REQ-APPLICATION-020: UpdateAdminApplication
管理者が Application のメタデータを更新する。

### REQ-APPLICATION-021: UploadApplicationIcon
管理者が Application のアイコン画像を multipart でアップロードする。受理形式は PNG / JPEG / WebP / GIF、最大 256KiB とし、magic byte で検証する。
- Precondition: context.upload.content_type in ["image/png", "image/jpeg", "image/webp", "image/gif"]
- Precondition: context.upload.magic_byte_matches_content_type
- Precondition: size(input.file) <= 262144

### REQ-APPLICATION-022: DeleteApplicationIcon
管理者が Application の保存済みアイコンを削除する。削除後は icon_url を返さない。

### REQ-APPLICATION-023: GetApplicationIcon
保存済み Application アイコンを配信する公開 interface。Content-Type は検証済み形式に固定し、X-Content-Type-Options: nosniff を付ける。別テナントまたは削除済み object は未存在として扱う。
- Postcondition: response.headers["Content-Type"] in ["image/png", "image/jpeg", "image/webp", "image/gif"]
- Postcondition: response.headers["X-Content-Type-Options"] == "nosniff"

### REQ-APPLICATION-024: DeleteAdminApplication
管理者が Application を削除する (冪等)。関連する protocol row は同一 transaction で cascade delete される。

### REQ-APPLICATION-025: UpdateApplicationOidcConfig
管理者が Application の OIDC protocol が指す OAuth2 client の設定を更新する。rules /
sub_source_attribute は claim release 上書きであり、visibility=Private の
属性や reserved claim type を参照する rule は拒否する。
- Precondition: claim_release_rules_within_floor(context.tenant_id, input.request.rules)

### REQ-APPLICATION-026: RotateApplicationClientSecret
互換性のため、管理者が Application の OIDC protocol の client secret を従来方式で回す。旧 credential は指定した overlap 期間だけ有効にし、0日は即時 revoke とする。新しい管理 UI は IssueApplicationClientSecret と RevokeApplicationClientSecret を使用する。

### REQ-APPLICATION-027: IssueApplicationClientSecret
管理者が Application の secret-based OIDC client に期限付き credential を追加発行する。既存 credential は変更せず、Active credential が2件なら原子的に拒否する。secret-based でない client は拒否する。

### REQ-APPLICATION-028: RevokeApplicationClientSecret
管理者が Application の secret-based OIDC client に属する指定 credential だけを即時失効する。既に失効済みなら冪等に成功し、別 client または存在しない credential は拒否する。

### REQ-APPLICATION-029: UpdateApplicationWsFedConfig
管理者が Application の WS-Fed protocol が指す relying party の設定を更新する。rules /
name_id_source は claim release 上書きであり、visibility=Private の属性や
reserved claim type を参照する rule は拒否する。
- Precondition: claim_release_rules_within_floor(context.tenant_id, input.request.rules)

### REQ-APPLICATION-030: UpdateApplicationSamlConfig
管理者が Application の SAML protocol が指す service provider 設定とIdP profile bindingを
部分更新する。署名要求を有効にする場合は検証可能な証明書を必須とする。rules / name_id_source は
claim release 上書きであり、visibility=Private の属性や reserved claim type
を参照する rule は拒否する。
- Precondition: claim_release_rules_within_floor(context.tenant_id, input.request.rules)

### REQ-APPLICATION-031: ListApplicationAssignments
管理者が Application の割当を subject_type, subject_id 昇順の双方向 keyset pagination で
一覧する。cursor は応答の Link response header (rel="prev" / rel="next") から取得する。

### REQ-APPLICATION-032: AssignApplication
管理者が Application にユーザー / グループを割当てる。
- Precondition: subject_exists_in_tenant(input.request.subject_type, input.request.subject_id, context.tenant_id)

### REQ-APPLICATION-033: UnassignApplication
管理者が Application の割当を解除する (冪等)。

### REQ-APPLICATION-034: GetAppSignInPolicy
管理者が Application の sign-in policy を取得する。アプリ個別ポリシー・テナントデフォルト・上書き後に実際に適用される effective ルール列と、デフォルトより弱いかの警告フラグを返す。未設定なら空 rules を返す。

### REQ-APPLICATION-035: UpdateAppSignInPolicy
管理者が Application の sign-in policy を置き換える。無効な CIDR を含む条件は拒否する。

### REQ-APPLICATION-036: GetTenantDefaultSignInPolicy
管理者がテナントデフォルト sign-in policy と MFA 未登録 active user 数を取得する。未設定なら空 rules の policy を返す。

### REQ-APPLICATION-037: UpdateTenantDefaultSignInPolicy
管理者がテナントデフォルト sign-in policy を置き換える。MFA 必須化では強制開始日時、enrollment bypass を利用できる猶予、管理者承認の可否を明示する。無効な CIDR、過去の強制開始日時、非正値の猶予は拒否する。空 rules で保存すればデフォルトは allow-all に戻る。

### REQ-APPLICATION-038: ListApplicationCategories
管理者が所属テナントの ApplicationCategory を position 昇順で一覧する。

### REQ-APPLICATION-039: CreateApplicationCategory
管理者が ApplicationCategory を作成する。

### REQ-APPLICATION-040: UpdateApplicationCategory
管理者が ApplicationCategory の名前 / 並び順を更新する。

### REQ-APPLICATION-041: DeleteApplicationCategory
管理者が ApplicationCategory を削除する (冪等)。付与済みの Application からも除く。

### REQ-APPLICATION-042: SetApplicationCategories
管理者が Application に付与するカテゴリ集合を置き換える。所属テナントの既存カテゴリのみ受け付ける。

### REQ-APPLICATION-043: ListMyApplications
利用者が自分に割当済み (visible) の Application を一覧する。hidden 割当は除外する。
手動並び順 (ApplicationOrdering) があればそれを優先し、無ければ name 昇順。応答の categories は
tenant のカテゴリ定義を position 昇順で含み、各アプリの category_ids でセクションに振り分ける。
- Postcondition: output.response.applications.all(a, is_visible_assignment(subject.id, a))

### REQ-APPLICATION-044: GetMyApplicationOrder
利用者が保存済みのポータル手動並び順を取得する。未保存なら空配列を返す。

### REQ-APPLICATION-045: ReorderMyApplications
利用者が自分のポータル手動並び順を保存する。application_ids は自分に割当済み visible な
アプリのみを含み、それ以外を含むと拒否する。
- Precondition: input.request.application_ids.all(id, is_visible_assignment(subject.id, id))

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Application | 運用者が接続・割当・監査する業務アプリケーション。federated / service Application は最大1個の protocol 設定を持つ。 | アプリケーション, Application |
| ApplicationProtocol | Application が利用する単一の protocol 設定への型付き参照。OAuth2Client、SamlServiceProvider、WsFedRelyingParty のいずれか1個を指す。 | application_protocol |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
