# Feature: SAML の SSO の例

## Rule: REQ-SAML-002 SP は割り当てられた IdP プロファイルだけを利用できる

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

## Rule: REQ-SAML-006 SAML の SP 起点 SSO に成功する

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

### Example: EX-SAML-007-01 通常経路

- Given AuthnRequest の entityID、ACS URL、Destination、対象ユーザーの割り当てのいずれかが不正である
- When 不正な AuthnRequest を受信する
- Then SAMLResponse を発行せず SamlSignInRejected を発行する

## Rule: REQ-SAML-008 SAML ForceAuthn は古いセッションをログインへ戻す

### Example: EX-SAML-008-01 通常経路

- Given ForceAuthn=true かつ認証時刻が再認証猶予より古い
- When ForceAuthn=true の AuthnRequest を受信する
- Then 古い認証コンテキストを検出する
- Then ログインへリダイレクトする
