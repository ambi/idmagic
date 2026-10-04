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

#### REQ-SAML-008 SAML ForceAuthn は古いセッションをログインへ戻す

#### REQ-SAML-002 SP は割り当てられた IdP プロファイルだけを利用できる

#### REQ-SAML-007 未登録または不一致の SAML リクエストを拒否する

## セキュリティ上の考慮

SSO と SLO は管理者の認可を通らず、ブラウザーのログインセッションで主体を決める。
SP の entityID、`AssertionConsumerServiceURL`、Destination、対象のユーザーの Application への割り当てをすべて検証してから発行し、一つでも一致しなければ SAMLResponse を発行しない。

テナントとプロファイルは、どちらも信頼の境界である。
ある信頼の境界に対する正当な要求を、同じテナントの別のプロファイルへ送り直しても通らない。
