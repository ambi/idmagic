---
context: application
updated_at: 2026-08-11
---

# Application Specification

## Overview

運用者が「接続する業務アプリケーション」として扱う上位概念を所有する。OIDC クライアント、SAML SP、WS-Fed RP は Application に関連付けるプロトコル設定であり、表示名、アイコン、ライフサイクル、割り当てはここに集約する。割り当てによって、ポータルでの表示とフェデレーションの利用可否をフェイルクローズで制御する。通信時の動作は各プロトコルの Context が所有し、Application はプロトコル設定を中身に依存しないキーで参照する。

`domain` が Aggregate とポリシー評価の規則を、`ports` と `usecase` がカタログとポリシーの操作を、`handlers_http` が HTTP アダプターを持つ。`module.go` は、それらをルーティングに組み込む。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Application | 運用者が接続、割り当て、監査する業務アプリケーション。`federated` または `service` の Application は、プロトコル設定を最大 1 つ持つ。 | アプリケーション, Application |
| ApplicationProtocol | Application が利用する 1 つのプロトコル設定への型付き参照。OAuth2Client、SamlServiceProvider、WsFedRelyingParty のいずれか 1 つを指す。 | application_protocol |

## Authorization Boundary

認可の意味はアプリケーションとそのテストで保証する。本仕様では API の認証方式を定めるが、ポリシー用の DSL は定義しない。

## Design

### Internal Interfaces

#### AssignApplicationDesiredState
呼び出し元の Bounded Context（IdManagement の LifecycleWorkflow など）が、あるべき状態を指定して Application へのユーザー割り当てを作成する内部インターフェースである。HTTP には公開せず、同じプロセス内の Go 呼び出しとして各 Context のユースケースから利用する。同じ `id` と `user_id` の割り当てが指定した `visibility` ですでに存在する場合は、変更せずに `changed=false` を返す。呼び出し元が渡せるのは、同じテナントの `id` と `user_id` だけである。

#### UnassignApplicationDesiredState
呼び出し元の Bounded Context が、あるべき状態を指定して Application へのユーザー割り当てを解除する内部インターフェースである。割り当てが存在しない場合も何も変更せず、`changed=false` を返して正常終了する。

### Sign-in policy evaluation

`AppSignInPolicy` は、テナントとアプリケーションごとに順序付けた `SignInRule` の集合であり、ApplicationCatalog が所有する。OIDC の認可、SAML の SSO、WS-Fed のサインインなど、フェデレーションを開始するたびにトークンや Assertion の発行前に評価する。アプリケーションとプロトコル設定の関連付けを確認するのと同じ関門で評価するため、別のプロトコルを入口に選んでポリシー評価を迂回することはできない。設定できるのは評価器が実際に確認できる値だけである。

必須の認証強度は自由入力ではなく、`Password` または `Mfa` に制約した `RequiredAuthnStrength` 列挙であり、内部の ACR URN と AMR 値へ 1 対 1 で対応付ける。実際に存在する ACR 値は 2 種類だけで、制約のない文字列は設定ミスを招くためである。`reauth_max_age_seconds` は Authentication のステップアップ認証の直近性に対して評価し、`network_allow_cidrs` は管理者が指定して保存時に検証した CIDR に対してリクエスト元のクライアント IP を検査する。評価器が確認できない端末条件は受け付けない。

評価はすべてフェイルクローズで行う。OIDC は認証強度不足の結果を既存のステップアップ認証フローへ送れる。一方、SAML と WS-Fed には遷移先となるステップアップ認証機構がまだないため、明示的な拒否理由を付けてプロトコルトランザクションを直ちに停止する。空でない CIDR 許可リストにクライアント IP が一致しない場合や、リクエスト元のクライアント IP を特定できない場合は、ステップアップ認証の機会とはせず、無条件に拒否する。

### Tenant default policy composition

`TenantDefaultSignInPolicy` により、テナントは独自のポリシーを定義していないすべてのアプリケーションに対して、基準となるサインインポリシーを 1 つ設定できる。アプリケーションごとのポリシーと同じ `SignInRule` の語彙と評価器を使用し、別のポリシー言語は設けない。これを `Tenancy` ではなく ApplicationCatalog が所有するのは、テナント Aggregate そのものではなく、アプリケーションへのサインイン方法に関する概念だからである。これにより、サインインポリシーの所有者を 1 か所に保つ。

デフォルトポリシーとアプリケーションごとのポリシーは、合成せずに上書きする。アプリケーションが有効な規則を 1 つでも定義していれば、その規則がテナントのデフォルトを完全に置き換える。定義がなければデフォルトをそのまま適用する。`EffectiveSignInRules(default, app)` がどちらか一方を選び、アプリケーションごとのポリシーと同じフェイルクローズの評価器に渡す。これにより、各アプリケーションに実際に適用されるポリシーを 1 つだけ直接確認できる。

上書きによってアプリケーションはテナントのデフォルトより弱いポリシーを設定できるため、`AppSignInPolicyResponse` は `weaker_than_default` フラグを持つ。要求する認証強度を下げる、再認証の時間制限を緩めるか外す、許可ネットワークを広げる場合に、`AppPolicyWeakerThanDefault(default, app)` がこの値を算出する。保存を禁止するのではなく、UI に警告を表示する。新しいテナントは規則が空の、すべてを許可するデフォルトから始める。デフォルトは通常のテーブル行として保存するので、規則を空にするか行を削除すれば、スキーマを変更せずにすべてを許可する状態へ戻せる。

### Application/protocol relation

Application が持つプロトコル設定は最大 1 つとし、作成時に固定する。`weblink` アプリケーションはプロトコル設定を持たず、`federated` と `service` のアプリケーションは、OAuth2 クライアント、SAML サービスプロバイダー、WS-Federation のリライングパーティーのいずれか 1 つだけを持つ。作成後の再接続、切り離し、プロトコル種別の変更には対応しない。

各プロトコルのテーブル（`oauth2_clients`、`saml_service_providers`、`wsfed_relying_parties`）は、`NULL` を許容する一意な `application_id` を持つ。`application_id` が `NULL` でない場合は、テナントと固定のプロトコル判別子も含む複合外部キーで参照する。これにより、2 つのプロトコル行が同じ Application を参照すること、テーブルをまたいで重複して参照すること、テナントや種別が食い違うことをデータベース自身が拒否する。`NULL` は、Dynamic Client Registration や信頼管理 API で作成され、Application カタログには表示しない正当なレコードを表す。そのため、すべてのプロトコル設定に Application を必須とはしない。

カタログへの作成では、Application 行の作成とプロトコル行への `application_id` の設定を 1 つのトランザクションで確定する。後半が失敗しても、カタログにだけ表示される孤立した Application は残らない。Application を削除すると、それが所有するプロトコル設定も連鎖して削除する。一方、Application が所有するプロトコル設定を各プロトコルの管理 API から直接削除しようとした場合は、競合として拒否する。削除は必ず所有元の Application を経由する。

OAuth2 のプロトコルテーブルは `oauth2_clients` とする。SAML と WS-Fed のテーブル（`saml_service_providers`、`wsfed_relying_parties`）と同様に、プロトコル固有の標準用語を使う。

### Portal application ordering and category

ApplicationCatalog は、エンドユーザーポータルでの手動の並び順と、管理者が定義するカテゴリの両方を所有する。どちらも IdentityManagement の User Aggregate ではなく、`Application` の表示に関わる事柄だからである。手動の並び順は `ApplicationOrdering` であり、`(tenant_id, user_sub)` ごとの `application_id` の一覧で表す。保存された並び順を持たないアプリケーションは名前の昇順に並べる。`ListMyApplications` は、まず割り当て済みで可視かつ有効なアプリケーションを解決し、その後に保存された並び順を適用する。割り当てが外れた項目は除外し、保存された並び順にない割り当て済みアプリケーションは名前順で末尾に加えるため、割り当てが並行して変わっても一覧は壊れない。並び替え (`ReorderMyApplications`) は並び順の一覧を作成または更新するだけである。これは個人の表示設定であり、認可や監査対象の状態遷移ではないため、ドメインイベントを発行しない。カテゴリはテナントごとに管理者が定義し、Application ごとに 0 個以上を割り当てる。

### Design Decisions

- アプリケーションのサインインポリシー（`AppSignInPolicy`）は、評価器が確認できる構造化された規則だけを持つ。制限された `RequiredAuthnStrength` 列挙、`reauth_max_age_seconds`、`network_allow_cidrs` を、すべてのプロトコルに共通するフェデレーションの関門でフェイルクローズに評価する。
- テナント全体のデフォルトのサインインのポリシーは、アプリケーション自身のポリシーと合成せず上書きされる。アプリケーションごとに直接確認できる実効的なポリシーを 1 つに保つためである。
- Application が持つプロトコル設定は最大 1 つとし、作成時に固定する。JSON 配列による関連付けモデルではなく、所有するプロトコルテーブルからの複合外部キーで保証する。

## Scenarios

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
