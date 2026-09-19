# Feature: Saml のシナリオ

## Rule: REQ-SAML-001 SP は署名証明書を取得できる

Primary actor: `EndUser`

### Example: EX-SAML-001-01 通常経路

- Given SP が IdMagic テナントの SAML メタデータまたは証明書ダウンロード URL を参照できる
- When SP が証明書ダウンロード URL にリクエストを送る
- Then SP は現在有効な `XmlFederationSigning` 証明書を PEM 形式で取得する
- Then 取得した証明書は同じ時点の SAML メタデータで公開される証明書と一致する
- Then ローテーションの移行期間中に信頼するすべての証明書は SAML メタデータから取得する

### Example: EX-SAML-001-02 フェデレーション署名資格情報を利用できない

- Given SP が IdMagic テナントの SAML メタデータまたは証明書ダウンロード URL を参照できる
- When SP が証明書ダウンロード URL にリクエストを送る
- Then フェデレーション署名資格情報を利用できない
- Then 証明書を返さずにエラーを返す

## Rule: REQ-SAML-002 SP は割り当てられた IdP プロファイルだけを利用できる

Primary actor: `EndUser`

### Example: EX-SAML-002-01 通常経路

- Given SP は `profile-a` に割り当てられている
- And `profile-a` と `profile-b` は同じテナント内に存在する
- When SP が `profile-a` の SSO エンドポイントに AuthnRequest を送る
- Then Destination、SP の Issuer、プロファイルとの関連付けを一体として検証する
- Then `profile-a` の entityID と署名資格情報を使用して SAMLResponse を発行する

### Example: EX-SAML-002-02 同じリクエストを `profile-b` の SSO エンドポイントに送る

- Given SP は `profile-a` に割り当てられている
- And `profile-a` と `profile-b` は同じテナント内に存在する
- When SP が `profile-a` の SSO エンドポイントに AuthnRequest を送る
- But 同じリクエストを `profile-b` の SSO エンドポイントに送る
- Then SAMLResponse を発行せず、SamlSignInRejected を発行する

### Example: EX-SAML-002-03 `profile-a` の SSO URL と異なる Destination を指定する

- Given SP は `profile-a` に割り当てられている
- And `profile-a` と `profile-b` は同じテナント内に存在する
- When SP が `profile-a` の SSO エンドポイントに AuthnRequest を送る
- But `profile-a` の SSO URL と異なる Destination を指定する
- Then フェイルクローズで拒否する

## Rule: REQ-SAML-003 専用プロファイルは固有のメタデータを公開する

Primary actor: `EndUser`

### Example: EX-SAML-003-01 通常経路

- Given テナントに `default` プロファイルと `dedicated` プロファイルが存在する
- When `dedicated` プロファイルのメタデータ URL を取得する
- Then メタデータでプロファイル固有の entityID、SSO / SLO URL、署名証明書を公開する
- Then `default` プロファイルのメタデータでは異なる署名資格情報を公開する

### Example: EX-SAML-003-02 存在しないプロファイルまたは別テナントのプロファイル ID を指定する

- Given テナントに `default` プロファイルと `dedicated` プロファイルが存在する
- When `dedicated` プロファイルのメタデータ URL を取得する
- But 存在しないプロファイルまたは別テナントのプロファイル ID を指定する
- Then メタデータや証明書を公開せず、not found を返す

## Rule: REQ-SAML-004 管理者は SAML IdP プロファイルを共有用または専用として管理できる

Primary actor: `TenantAdministrator`

### Example: EX-SAML-004-01 通常経路

- Given テナントには変更できない `default` の `shared` プロファイルが存在する
- When 管理者が読み取り専用の連携エンドポイント画面から、プロファイルの管理一覧と詳細画面へ移動する
- Then 専用プロファイルの一覧と詳細が表示される
- When 管理者がプロファイル作成画面で `shared` プロファイルを作成する
- Then 複数の SP からそのプロファイルを選択できる
- When 管理者がプロファイル詳細から編集画面へ移り、追加プロファイルの名前またはモードを変更する
- Then 変更が保存される
- When 管理者が `dedicated` プロファイルを作成して 1 つの SP に割り当てる
- Then `dedicated` プロファイルと SP の関連付けが保存される
- When 管理者が未使用の追加プロファイルを削除する
- Then プロファイルが削除される

### Example: EX-SAML-004-02 `dedicated` プロファイルを別の SP にも割り当てる

- Given テナントには変更できない `default` の `shared` プロファイルが存在する
- When 管理者が読み取り専用の連携エンドポイント画面から、プロファイルの管理一覧と詳細画面へ移動する
- Then 専用プロファイルの一覧と詳細が表示される
- When 管理者がプロファイル作成画面で `shared` プロファイルを作成する
- Then 複数の SP からそのプロファイルを選択できる
- When 管理者がプロファイル詳細から編集画面へ移り、追加プロファイルの名前またはモードを変更する
- Then 変更が保存される
- When 管理者が `dedicated` プロファイルを作成して 1 つの SP に割り当てる
- But `dedicated` プロファイルを別の SP にも割り当てる
- Then 関連付けを InvalidRequestError で拒否する

### Example: EX-SAML-004-03 プロファイルが SP から参照されている、またはデフォルトプロファイルである

- Given テナントには変更できない `default` の `shared` プロファイルが存在する
- When 管理者が読み取り専用の連携エンドポイント画面から、プロファイルの管理一覧と詳細画面へ移動する
- Then 専用プロファイルの一覧と詳細が表示される
- When 管理者がプロファイル作成画面で `shared` プロファイルを作成する
- Then 複数の SP からそのプロファイルを選択できる
- When 管理者がプロファイル詳細から編集画面へ移り、追加プロファイルの名前またはモードを変更する
- Then 変更が保存される
- When 管理者が `dedicated` プロファイルを作成して 1 つの SP に割り当てる
- Then `dedicated` プロファイルと SP の関連付けが保存される
- When 管理者が未使用の追加プロファイルを削除する
- But プロファイルが SP から参照されている、またはデフォルトプロファイルである
- Then 削除を conflict で拒否する

## Rule: REQ-SAML-005 管理 API クライアントは SAML スコープに従ってサービスプロバイダーを操作できる

Primary actor: `ManagementApiClient`

### Example: EX-SAML-005-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがサービスプロバイダーの参照、登録、または削除をリクエストする
- Then `saml:read` スコープではサービスプロバイダーの参照だけを許可する
- Then `saml:write` スコープではサービスプロバイダーの登録または削除だけを許可する

### Example: EX-SAML-005-02 `saml:read` だけで変更操作をリクエストする

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがサービスプロバイダーの参照、登録、または削除をリクエストする
- But `saml:read` だけで変更操作をリクエストする
- Then 操作を AccessDeniedError で拒否する

### Example: EX-SAML-005-03 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがサービスプロバイダーの参照、登録、または削除をリクエストする
- But トークンのテナントとリクエスト先のテナントが一致しない
- Then 操作を AccessDeniedError で拒否する

## Rule: REQ-SAML-006 SAML の SP 起点 SSO に成功する

Primary actor: `EndUser`

### Example: EX-SAML-006-01 通常経路

- Given 対象者は認証済みで、対象の Application に割り当てられている
- And SP の entityID、ACS URL、Destination は登録済みである
- When 登録済み SP の AuthnRequest を受信する
- Then Version / IssueInstant / Issuer / ACS / Destination / バインディング / NameIDPolicy / 対象者の割り当てを検証する
- Then 署名済み SAMLResponse を ACS へ POST し RelayState を同値で返す
- Then Assertion と Response の署名には、リクエスト先テナントで現在有効な `XmlFederationSigning` 鍵を使用する

### Example: EX-SAML-006-02 entityID、ACS、Destination、対象者の割り当てのいずれかが不正である

- Given 対象者は認証済みで、対象の Application に割り当てられている
- And SP の entityID、ACS URL、Destination は登録済みである
- When 登録済み SP の AuthnRequest を受信する
- Then entityID、ACS、Destination、対象者の割り当てのいずれかが不正である
- Then SAMLResponse を発行しない
- And SamlSignInRejected を発行してフェイルクローズで拒否する

### Example: EX-SAML-006-03 AuthnRequest の解析または署名検証に失敗する

- Given 対象者は認証済みで、対象の Application に割り当てられている
- And SP の entityID、ACS URL、Destination は登録済みである
- When 登録済み SP の AuthnRequest を受信する
- Then AuthnRequest の解析または署名検証に失敗する
- Then SamlSignInRejected を発行してプロトコルエラーを返す

### Example: EX-SAML-006-04 AuthnRequest の Version、IssueInstant、ProtocolBinding、ACS インデックス、NameIDPolicy の形式が未対応または矛盾する

- Given 対象者は認証済みで、対象の Application に割り当てられている
- And SP の entityID、ACS URL、Destination は登録済みである
- When 登録済み SP の AuthnRequest を受信する
- Then AuthnRequest の Version、IssueInstant、ProtocolBinding、ACS インデックス、NameIDPolicy の形式が未対応または矛盾する
- Then Assertion を発行しない
- And 検証済みの ACS が確定している場合だけ HTTP-POST の SAML プロトコルエラーを返す
- And それ以外は SamlSignInRejected を発行してフェイルクローズで拒否する

### Example: EX-SAML-006-05 `IsPassive=true` かつ利用可能な既存セッションがない

- Given 対象者は認証済みで、対象の Application に割り当てられている
- And SP の entityID、ACS URL、Destination は登録済みである
- When 登録済み SP の AuthnRequest を受信する
- Then `IsPassive=true` かつ利用可能な既存セッションがない
- Then ログイン画面へ遷移しない
- And 検証済みの ACS へ HTTP-POST の NoPassive SAML プロトコルレスポンスを返す

### Example: EX-SAML-006-06 同じテナント、SP、AuthnRequest ID の組み合わせに対する Assertion が発行済みである

- Given 対象者は認証済みで、対象の Application に割り当てられている
- And SP の entityID、ACS URL、Destination は登録済みである
- When 登録済み SP の AuthnRequest を受信する
- Then Version / IssueInstant / Issuer / ACS / Destination / バインディング / NameIDPolicy / 対象者の割り当てを検証する
- Then 同じテナント、SP、AuthnRequest ID の組み合わせに対する Assertion が発行済みである
- Then Assertion を発行しない
- And SamlSignInRejected を発行してフェイルクローズで拒否する

## Rule: REQ-SAML-007 未登録または不一致の SAML リクエストを拒否する

Primary actor: `EndUser`

### Example: EX-SAML-007-01 通常経路

- Given AuthnRequest の entityID、ACS URL、Destination、対象ユーザーの割り当てのいずれかが不正である
- When 不正な AuthnRequest を受信する
- Then SAMLResponse を発行せず SamlSignInRejected を発行する

## Rule: REQ-SAML-008 SAML ForceAuthn は古いセッションをログインへ戻す

Primary actor: `EndUser`

### Example: EX-SAML-008-01 通常経路

- Given ForceAuthn=true かつ認証時刻が再認証猶予より古い
- When ForceAuthn=true の AuthnRequest を受信する
- Then 古い認証コンテキストを検出する
- Then ログインへリダイレクトする
