# SAML の SSO

## 概要

この文書は、SAML 2.0 の Web Browser SSO Profile による SSO と Single Logout の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | AuthnRequest の復号、解析、署名の検証、SP とプロファイルと宛先の検証、署名した SAMLResponse の発行、Single Logout |
| 行為者 | EndUser（ブラウザーで SSO と Single Logout を始める利用者） |
| 扱わないもの | ログインセッションの確立は `Authentication` が、Application への割り当ては `Application` が扱う |

## モデル

Assertion にはデフォルトで署名し、Response への署名は SP ごとに任意で有効にできる。
これは Okta や Entra の「Sign Response」に当たる。

| 検証 | 内容 |
| --- | --- |
| Issuer | 登録した SP の entityID と完全に一致する |
| `AssertionConsumerServiceURL` | SP の許可リストと照合し、任意の宛先への転送を防ぐ |
| audience | SP の entityID に限る |
| Destination | 要求先の経路が指すプロファイルの正式なエンドポイントと一致する |
| プロファイル | 要求先の経路が指すプロファイルと、SP に割り当てたプロファイルが一致する |
| AuthnRequest の ID | 同じテナント、SP、要求の ID の組には、Assertion を一度だけ発行する |

検証の結果を確定できない場合や、値が一致しない場合は、すべて拒否する。

## 操作

### 利用者による SP 起点の SSO

#### REQ-SAML-006 SAML の SP 起点 SSO に成功する

- `Active` の User のログインセッションがある間、登録済みの SP の妥当な AuthnRequest を受けたとき、Saml は、署名した SAMLResponse を SP の ACS へ自動で POST するフォームで返し、`RelayState` を同じ値で返し、`SamlSignInIssued` を発行する。
- SAMLResponse を発行するとき、Saml は、要求先のプロファイルの entityID と、リクエスト先のテナントで現在有効な `XmlFederationSigning` の鍵を使い、Assertion にデフォルトで署名し、Response への署名を有効にした SP では Response にも署名する。
- SAMLResponse を発行するとき、Saml は、SP の `ClaimMappingPolicy` でクレームを発行する。
- AuthnRequest が NameIDPolicy の形式を指定したとき、Saml は、その形式の NameID を返す。
- ログインセッションがないか、認証が完了していないか、User が `Active` でない間、`IsPassive` でない AuthnRequest を受けたとき、Saml は、SAMLResponse を発行せず、元の要求に戻るログイン画面へ 303 でリダイレクトする。
- ログインセッションがない間、`IsPassive=true` の AuthnRequest を受けたとき、Saml は、ログイン画面へ遷移せず、`NoPassive` の状態の SAMLResponse を ACS へ返す。
- Version が `2.0` でないか、`IssueInstant` がないか、10 分より前か 30 秒より先の `IssueInstant` の AuthnRequest を受けた場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、Assertion を発行しない。
- ACS の URL と ACS のインデックスの両方、ACS のインデックス、未対応の `ProtocolBinding`、未対応の NameIDPolicy の形式を指定された AuthnRequest を受けた場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、Assertion を発行しない。
- AuthnRequest を復号または解析できないか、署名を求める SP の AuthnRequest の署名を検証できない場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、Assertion を発行しない。
- 同じテナント、SP、AuthnRequest の ID の組に Assertion を発行済み（10 分以内）の場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、Assertion を発行しない。
- SP が属する Application に利用者が割り当てられていないか、Application のサインインポリシーを満たさない場合、Saml は、403 で拒否し、`SamlSignInRejected` を発行し、Assertion を発行しない。
- クレームを発行できないか、再送の記録を確かめられない場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、Assertion を発行しない。
- **例**：EX-SAML-006-01、EX-SAML-006-02、EX-SAML-006-03、EX-SAML-006-04、EX-SAML-006-05、EX-SAML-006-06

#### REQ-SAML-008 SAML ForceAuthn は古いセッションをログインへ戻す

- 30 秒より前に認証したログインセッションの間、`ForceAuthn=true` の AuthnRequest を受けたとき、Saml は、SAMLResponse を発行せず、元の要求に戻るログイン画面へ 303 でリダイレクトする。
- 30 秒以内に認証したログインセッションの間、`ForceAuthn=true` の AuthnRequest を受けたとき、Saml は、ほかの検証を満たせば SAMLResponse を発行する。
- **例**：EX-SAML-008-01

#### REQ-SAML-002 SP は割り当てられた IdP プロファイルだけを利用できる

- SP が関連付けた IdP プロファイルの SSO の経路で AuthnRequest を受けたとき、Saml は、Destination、SP の Issuer、プロファイルとの関連付けをまとめて検証し、そのプロファイルの entityID と署名の資格情報で SAMLResponse を発行する。
- SP が関連付けていないプロファイルの経路で AuthnRequest を受けた場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、SAMLResponse を発行しない。
- Destination が要求先のプロファイルの SSO の URL と一致しない AuthnRequest を受けた場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、SAMLResponse を発行しない。
- 存在しないプロファイルの経路で要求された場合、Saml は、404 で拒否する。
- **例**：EX-SAML-002-01、EX-SAML-002-02、EX-SAML-002-03

#### REQ-SAML-007 未登録または不一致の SAML リクエストを拒否する

- Issuer がテナントに登録した SP の entityID でない AuthnRequest を受けた場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、SAMLResponse を発行しない。
- `AssertionConsumerServiceURL` が SP の許可した ACS の URL と完全には一致しないか、ACS の URL を登録していない SP の AuthnRequest を受けた場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行し、SAMLResponse を発行しない。
- `SAMLRequest` も SP の entityID もない要求を受けた場合、Saml は、400 で拒否し、`SamlSignInRejected` を発行する。
- **例**：EX-SAML-007-01

### 利用者による Single Logout

#### REQ-SAML-009 Single Logout はローカルセッションを破棄し、登録した SLO の URL にだけ応答を返す

- 登録済みの SP の妥当な LogoutRequest を受けたとき、Saml は、ローカルのログインセッションを失効させ、セッション Cookie を消し、`SamlLogout` を発行し、署名した LogoutResponse を SP の SLO の URL へ 303 でリダイレクトして返し、`RelayState` を同じ値で返す。
- LogoutRequest を伴わない Single Logout の要求を受けたとき、Saml は、ローカルのログインセッションを失効させ、セッション Cookie を消し、`SamlLogout` を発行する。
- LogoutRequest を伴わない要求の SP が SLO の URL を登録し、要求先のプロファイルに関連付けられている間、Single Logout の要求を受けたとき、Saml は、その SLO の URL へ 303 でリダイレクトする。
- LogoutRequest を伴わない要求の SP を解決できない場合、Saml は、リダイレクトせずに 200 を返す。
- LogoutRequest を復号または解析できないか、SP が要求先のプロファイルに関連付けられていないか、署名を検証できないか、Destination が SLO の URL と一致しない場合、Saml は、400 で拒否し、ローカルのログインセッションを変えない。
- LogoutRequest の Issuer が SLO の URL を登録した SP でない場合、Saml は、400 で拒否し、ローカルのログインセッションを変えずに `SamlLogout` を発行する。

## セキュリティ上の考慮

SSO と SLO は管理者の認可を通らず、ブラウザーのログインセッションで主体を決める。
SP の entityID、`AssertionConsumerServiceURL`、Destination、対象のユーザーの Application への割り当てをすべて検証してから発行し、一つでも一致しなければ SAMLResponse を発行しない。

テナントとプロファイルは、どちらも信頼の境界である。
ある信頼の境界に対する正当な要求を、同じテナントの別のプロファイルへ送り直しても通らない。
