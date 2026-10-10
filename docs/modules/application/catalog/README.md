# アプリケーションのカタログ

## 概要

この文書は、管理者が Application とそのプロトコル設定を作成、参照、更新する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | Application とプロトコル設定の作成、IdMagic の側の連携の設定の提示、クライアントシークレットの管理、SAML のプロトコル設定の更新、公開するクレームの制限、アイコンの管理 |
| 行為者 | テナント管理者、管理 API のクライアント |
| 扱わないもの | 割り当ては[割り当て](../assignment/README.md)が、サインインポリシーは[サインインポリシー](../sign-in-policy/README.md)が扱う |

## モデル

Application に関連付けるプロトコル設定は最大一つとし、作成時に固定する。
`weblink` のアプリケーションにはプロトコル設定を関連付けず、`federated` と `service` のアプリケーションには、OAuth2 のクライアント、SAML の SP、WS-Federation の RP のいずれか一つだけを関連付ける。
作成の後の再接続、切り離し、プロトコルの種別の変更には対応しない。

プロトコル設定のうち `application_id` を持たないものは、Dynamic Client Registration や信頼の管理 API で作成され、カタログには表示しない正当なレコードである。
すべてのプロトコル設定に Application を必須とはしない。

Application を削除すると、それに紐づくプロトコル設定も連鎖して削除する。
Application に属するプロトコル設定を、各プロトコルの管理 API から直接削除しようとした場合は、競合として拒否する。

- **判断**：一対一の関係をデータベースで保証する理由は、[Application とプロトコル設定の関係を複合外部キーで保証する](../design/decisions.md#application-とプロトコル設定の関係を複合外部キーで保証する)。

## 操作

### 管理者によるアプリケーションの作成

#### REQ-APPLICATION-007 管理者は管理画面でアプリケーションと 1 つのプロトコルを構成できる

- 管理者が `weblink` のアプリケーションを作成したとき、Application は、プロトコル設定を関連付けずに Application を作り、`ApplicationCreated` を発行する。
- 管理者が `federated` または `service` のアプリケーションを作成したとき、Application は、OAuth2 のクライアント、SAML の SP、WS-Federation の RP のどれか一つと関連付けて Application を作り、`ApplicationCreated` を発行する。
- 管理者が confidential な OIDC のアプリケーションを作成したとき、Application は、生成した `client_secret` を作成の応答でだけ一度返す。
- 管理者がアプリケーションの名前や表示の設定を更新したとき、Application は、`ApplicationUpdated` を発行する。
- 管理者がアプリケーションを削除したとき、Application は、関連付けたプロトコル設定と割り当てを連鎖して削除し、`ApplicationDeleted` を発行する。
- 管理者が User または Group をアプリケーションに割り当てたとき、Application は、`visibility`（`visible` または `hidden`）の割り当てを作り、`ApplicationAssigned` を発行する。
- 管理者が割り当てを解除したとき、Application は、割り当てを消し、`ApplicationUnassigned` を発行する。
- 管理者がアプリケーションを取得または一覧したとき、Application は、呼び出し元のテナントのアプリケーションだけを返す。
- Application は、作成の後にプロトコル設定の再接続、切り離し、種別の変更を受け付けない。
- テナントの `applications` または `oauth2_clients` の上限を超える作成を要求された場合、Application は、422 と `quota_exceeded` で拒否し、Application を作らない。
- 別のテナントの主体か存在しない主体の割り当てを要求された場合、Application は、400 と `invalid_request` で拒否し、割り当てを作らない。
- 存在しないか別のテナントのアプリケーションを指定された場合、Application は、存在しない ID と同じ 404 と `application_not_found` で拒否し、名前もプロトコル設定も返さない。
- 不正なアプリケーションの作成または更新を要求された場合、Application は、400 と `invalid_request` で拒否し、Application を変えない。
- **例**：EX-APPLICATION-007-01、EX-APPLICATION-007-02、EX-APPLICATION-007-03

#### REQ-APPLICATION-013 admin ロールを持たない利用者は Application を操作できない

- `admin` のロールを持たない利用者がアプリケーション、プロトコル設定、カテゴリ、割り当て、サインインポリシー、テナントのデフォルトのサインインポリシーの管理 API を要求した場合、Application は、403 と `access_denied` で拒否し、どれも変えない。
- **例**：EX-APPLICATION-013-01

### 管理者によるアプリケーションの参照

#### REQ-APPLICATION-001 管理者はアプリケーション詳細で IdMagic 側の連携設定を確認できる

- 管理者がアプリケーションの詳細を開いたとき、Application は、IdMagic に登録した RP または SP の情報と、接続先に設定する IdMagic の Discovery またはメタデータを分けて表示する。
- 管理者が OIDC のアプリケーションの詳細を開いたとき、Application は、Discovery の URL と `client_id` を表示する。
- 管理者が SAML のアプリケーションの詳細を開いたとき、Application は、IdP のメタデータの URL、entityID、SSO の URL、SLO の URL、署名証明書を表示する。
- Application は、クライアントシークレットを、作成、互換のローテーション、追加の発行に成功した応答の中だけで返す。
- **例**：EX-APPLICATION-001-01

### 管理者によるクライアントシークレットの管理

#### REQ-APPLICATION-002 管理者は通常設定とは独立したセクションでクライアントシークレットを管理できる

- 管理者が有効期限を選んでクライアントシークレットを追加で発行したとき、Application は、新しいシークレットを応答で一度だけ返し、資格情報を作成日、有効期限、`Active` の状態とともに一覧に加える。
- 管理者が互換のローテーションを要求したとき、Application は、新しいシークレットを応答で一度だけ返す。
- 管理者が資格情報を個別に失効させたとき、Application は、その資格情報だけを `Revoked` にする。
- `Active` の資格情報がすでに 2 件ある場合、Application は、追加の発行を 422 と `client_secret_limit_exceeded` で拒否し、資格情報を作らない。
- `Active` の資格情報が 2 件ある間、管理者が編集画面を開いたとき、Application は、追加の発行の操作を無効にし、先に既存の資格情報を失効させるよう案内する。
- 資格情報が `Expired` または `Revoked` の間、管理者が編集画面を開いたとき、Application は、その資格情報の失効の操作を表示しない。
- 管理者が編集画面を開いたとき、Application は、資格情報の一覧、追加の発行、個別の失効を、通常の設定の保存フォームの外の専用のセクションに表示する。
- **例**：EX-APPLICATION-002-01、EX-APPLICATION-002-02、EX-APPLICATION-002-03

### 管理者によるプロトコル設定の更新

#### REQ-APPLICATION-005 管理者は Application の SAML プロトコル設定を更新できる

- 管理者が SAML のアプリケーションの ACS の URL、署名の方針、クレームの規則、IdP プロファイルの割り当てを更新したとき、Application は、同じテナントの SAML の SP の設定だけを更新し、`ApplicationUpdated` を発行する。
- 管理者が OIDC または WS-Federation のアプリケーションのプロトコル設定を更新したとき、Application は、関連付けたクライアントまたは RP の設定だけを更新し、`ApplicationUpdated` を発行する。
- AuthnRequest の署名を必須にしたのに検証できる証明書がない SAML の設定を指定された場合、Application は、400 と `invalid_request` で拒否し、設定を変えない。
- **例**：EX-APPLICATION-005-01、EX-APPLICATION-005-02

#### REQ-APPLICATION-006 管理者は Application ごとに公開するクレームを制限できる

- 管理者がアプリケーションのクレームの公開の規則を保存したとき、Application は、そのアプリケーションの規則だけを置き換え、`ApplicationClaimMappingUpdated` を発行する。
- 管理者がクレームの公開の規則を保存したとき、Application は、同じテナントのほかのアプリケーションの規則を変えない。
- `Private` の属性かテナントの属性定義にないキーを `source_key` に指定するか、予約したクレーム型を `claim_type` に指定した規則を保存しようとした場合、Application は、400 と `invalid_request` で拒否し、規則を変えない。
- **例**：EX-APPLICATION-006-01、EX-APPLICATION-006-02、EX-APPLICATION-006-03

### 管理者によるアイコンの管理

#### REQ-APPLICATION-008 管理者は Application のアイコンをアップロード・削除できる

- 管理者が 262,144 バイト以下の PNG、JPEG、WebP、GIF の画像をアップロードしたとき、Application は、`icon_object_key` と IdP の配信 URL を指す `icon_url` を保存し、`ApplicationIconUpdated` を発行する。
- 管理者がアイコンを削除したとき、Application は、`icon_object_key` と `icon_url` を空にし、`ApplicationIconUpdated` を発行する。
- 管理一覧、詳細、利用者のポータルがアイコンを取得したとき、Application は、IdP の配信 URL から画像を返す。
- 画像でないか、262,144 バイトを超えるか、画像のないアップロードを受けた場合、Application は、400 と `invalid_icon` で拒否し、既存のアイコンを置き換えない。
- 別のテナントのアプリケーションの ID でアイコンを取得された場合、Application は、存在しない ID と同じ 404 と `not_found` で拒否し、画像を返さない。
- **例**：EX-APPLICATION-008-01、EX-APPLICATION-008-02、EX-APPLICATION-008-03

### 管理者によるカテゴリの管理

#### REQ-APPLICATION-015 管理者によるカテゴリの作成、更新、削除と割り当ては、名前のあるカテゴリだけを受け付ける

- 管理者がカテゴリを作成、更新、削除したとき、Application は、`ApplicationCategoryCreated`、`ApplicationCategoryUpdated`、`ApplicationCategoryDeleted` を発行する。
- 管理者がアプリケーションにカテゴリを設定したとき、Application は、アプリケーションのカテゴリを指定した集合で置き換え、`ApplicationUpdated` を発行する。
- 名前のないカテゴリの作成または更新を要求された場合、Application は、422 と `category_name_required` で拒否する。
- 存在しないカテゴリをアプリケーションに設定しようとした場合、Application は、422 と `unknown_category` で拒否し、アプリケーションのカテゴリを変えない。
- 存在しないカテゴリの更新または削除を要求された場合、Application は、404 と `category_not_found` で拒否する。

### 管理 API のクライアントによる操作

#### REQ-APPLICATION-004 管理 API クライアントは Application スコープで許可された操作だけを実行できる

- `applications:read` のスコープの API アクセストークンで呼び出されたとき、Application は、アプリケーション、カテゴリ、割り当ての参照だけを許可する。
- `applications:write` のスコープの API アクセストークンで呼び出されたとき、Application は、アプリケーションの作成、プロトコル設定の更新、削除を許可する。
- `settings:read` または `settings:write` のスコープの API アクセストークンで呼び出されたとき、Application は、テナントのデフォルトのサインインポリシーの参照または変更だけを許可する。
- `applications:read` だけのトークンでアプリケーションの変更を要求された場合、Application は、403 と `insufficient_scope` で拒否し、アプリケーションを変えない。
- 別のテナントで発行したトークンを提示された場合、Application は、401 と `invalid_token` で拒否する。
- **例**：EX-APPLICATION-004-01、EX-APPLICATION-004-02、EX-APPLICATION-004-03

## セキュリティ上の考慮

Application、プロトコル設定、カテゴリ、割り当て、サインインポリシーの管理は、いずれも `admin` ロールを持つ、有効かつ認証済みのユーザーに限る。
AuthZEN の action は対象ごとに分かれ、`admin:applications_manage`、`admin:application_assignments_manage`、`admin:application_policies_manage`、`admin:application_categories_manage`、`admin:tenant_default_sign_in_policy_manage` を要求する。
いずれも、操作の対象が呼び出し元と同じテナントに属することを条件とする。
