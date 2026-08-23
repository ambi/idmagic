# Saml Scenarios

### REQ-SAML-001: SP は署名証明書を取得できる
- ACTOR EndUser
- GIVEN SP が IdMagic テナントの SAML メタデータまたは証明書ダウンロード URL を参照できる
- WHEN SP が証明書ダウンロード URL にリクエストを送る
- THEN SP は現在有効な `XmlFederationSigning` 証明書を PEM 形式で取得する
  - ALT フェデレーション署名資格情報を利用できない → 証明書を返さずにエラーを返す
- THEN 取得した証明書は同じ時点の SAML メタデータで公開される証明書と一致する
- THEN ローテーションの移行期間中に信頼するすべての証明書は SAML メタデータから取得する

### REQ-SAML-002: SP は割り当てられた IdP プロファイルだけを利用できる
- ACTOR EndUser
- GIVEN SP は `profile-a` に割り当てられている
- GIVEN `profile-a` と `profile-b` は同じテナント内に存在する
- WHEN SP が `profile-a` の SSO エンドポイントに AuthnRequest を送る
  - ALT 同じリクエストを `profile-b` の SSO エンドポイントに送る → SAMLResponse を発行せず、SamlSignInRejected を発行する
  - ALT `profile-a` の SSO URL と異なる Destination を指定する → フェイルクローズで拒否する
- THEN Destination、SP の Issuer、プロファイルとの関連付けを一体として検証する
- THEN `profile-a` の entityID と署名資格情報を使用して SAMLResponse を発行する

### REQ-SAML-003: 専用プロファイルは固有のメタデータを公開する
- ACTOR EndUser
- GIVEN テナントに `default` プロファイルと `dedicated` プロファイルが存在する
- WHEN `dedicated` プロファイルのメタデータ URL を取得する
  - ALT 存在しないプロファイルまたは別テナントのプロファイル ID を指定する → メタデータや証明書を公開せず、not found を返す
- THEN メタデータでプロファイル固有の entityID、SSO / SLO URL、署名証明書を公開する
- THEN `default` プロファイルのメタデータでは異なる署名資格情報を公開する

### REQ-SAML-004: 管理者は SAML IdP プロファイルを共有用または専用として管理できる
- ACTOR TenantAdministrator
- GIVEN テナントには変更できない `default` の `shared` プロファイルが存在する
- WHEN 管理者が読み取り専用の連携エンドポイント画面から、プロファイルの管理一覧と詳細画面へ移動する
- THEN 専用プロファイルの一覧と詳細が表示される
- WHEN 管理者がプロファイル作成画面で `shared` プロファイルを作成する
- THEN 複数の SP からそのプロファイルを選択できる
- WHEN 管理者がプロファイル詳細から編集画面へ移り、追加プロファイルの名前またはモードを変更する
- THEN 変更が保存される
- WHEN 管理者が `dedicated` プロファイルを作成して 1 つの SP に割り当てる
  - ALT `dedicated` プロファイルを別の SP にも割り当てる → 関連付けを InvalidRequestError で拒否する
- THEN `dedicated` プロファイルと SP の関連付けが保存される
- WHEN 管理者が未使用の追加プロファイルを削除する
  - ALT プロファイルが SP から参照されている、またはデフォルトプロファイルである → 削除を conflict で拒否する
- THEN プロファイルが削除される

### REQ-SAML-005: 管理 API クライアントは SAML スコープに従ってサービスプロバイダーを操作できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- WHEN クライアントがサービスプロバイダーの参照、登録、または削除をリクエストする
  - ALT `saml:read` だけで変更操作をリクエストする → 操作を AccessDeniedError で拒否する
  - ALT トークンのテナントとリクエスト先のテナントが一致しない → 操作を AccessDeniedError で拒否する
- THEN `saml:read` スコープではサービスプロバイダーの参照だけを許可する
- THEN `saml:write` スコープではサービスプロバイダーの登録または削除だけを許可する

### REQ-SAML-006: SAML の SP 起点 SSO に成功する
- ACTOR EndUser
- GIVEN 対象者は認証済みで、対象の Application に割り当てられている
- GIVEN SP の entityID、ACS URL、Destination は登録済みである
- WHEN 登録済み SP の AuthnRequest を受信する
- THEN Version / IssueInstant / Issuer / ACS / Destination / バインディング / NameIDPolicy / 対象者の割り当てを検証する
  - ALT entityID、ACS、Destination、対象者の割り当てのいずれかが不正である → SAMLResponse を発行しない → SamlSignInRejected を発行してフェイルクローズで拒否する
  - ALT AuthnRequest の解析または署名検証に失敗する → SamlSignInRejected を発行してプロトコルエラーを返す
  - ALT AuthnRequest の Version、IssueInstant、ProtocolBinding、ACS インデックス、NameIDPolicy の形式が未対応または矛盾する → Assertion を発行しない → 検証済みの ACS が確定している場合だけ HTTP-POST の SAML プロトコルエラーを返す → それ以外は SamlSignInRejected を発行してフェイルクローズで拒否する
  - ALT `IsPassive=true` かつ利用可能な既存セッションがない → ログイン画面へ遷移しない → 検証済みの ACS へ HTTP-POST の NoPassive SAML プロトコルレスポンスを返す
- THEN 署名済み SAMLResponse を ACS へ POST し RelayState を同値で返す
  - ALT 同じテナント、SP、AuthnRequest ID の組み合わせに対する Assertion が発行済みである → Assertion を発行しない → SamlSignInRejected を発行してフェイルクローズで拒否する
- THEN Assertion と Response の署名には、リクエスト先テナントで現在有効な `XmlFederationSigning` 鍵を使用する

### REQ-SAML-007: 未登録または不一致の SAML リクエストを拒否する
- ACTOR EndUser
- GIVEN AuthnRequest の entityID、ACS URL、Destination、対象ユーザーの割り当てのいずれかが不正である
- WHEN 不正な AuthnRequest を受信する
- THEN SAMLResponse を発行せず SamlSignInRejected を発行する

### REQ-SAML-008: SAML ForceAuthn は古いセッションをログインへ戻す
- ACTOR EndUser
- GIVEN ForceAuthn=true かつ認証時刻が再認証猶予より古い
- WHEN ForceAuthn=true の AuthnRequest を受信する
- THEN 古い認証コンテキストを検出する
- THEN ログインへリダイレクトする
