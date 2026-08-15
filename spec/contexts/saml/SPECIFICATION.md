---
context: saml
updated_at: 2026-08-15
---

# Saml Specification

## Overview

SAML 2.0 IdP として、SP の信頼、IdP プロファイル、IdP メタデータ、AuthnRequest / Response、AssertionConsumerService、Single Logout を所有するプロトコルの Bounded Context である。Web Browser SSO Profile に基づき、SP 起点と IdP 起点の SSO を提供する。

WS-Fed / WS-Trust とは、クレームの発行処理と XML 署名だけを共有する。プロトコルに依存しないクレームの対応付けは `ClaimMapping`、署名鍵のライフサイクルは `SigningKeys` が所有する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Saml | SAML 2.0 IdP のプロトコル群。Web Browser SSO Profile に基づき、SP 起点と IdP 起点の SSO、IdP メタデータの公開、署名済み SAMLResponse の発行、Single Logout、SP ごとの AuthnRequest / LogoutRequest の署名検証を扱う。暗号化された Assertion、ECP、SAML SP、外部 IdP からのフェデレーションは初期範囲外とする。XML 署名と正規化は実績のあるライブラリに委ね、自前では実装しない。 | SAML, SAML2, SAML 2.0 |
| EndUser | SAML Web Browser SSO または Single Logout をブラウザで開始する利用者。 |  |
| SamlIdentityProviderProfile | テナント内の SAML IdP entityID、エンドポイント、XML 署名資格情報をまとめた信頼境界。`shared` プロファイルは複数の SP で共有でき、`dedicated` プロファイルは最大 1 つの SP にだけ割り当てられる。 | SAML IdP プロファイル, IdP プロファイル |

## Standards

### Assertions and Protocols for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Core-BearerAssertion | required | MUST | AuthnRequest は Version 2.0 と許容範囲内の IssueInstant を持たなければならない。発行する Assertion の Audience / Recipient / InResponseTo は、検証済みのリクエストと一致させる。同じテナント、SP、リクエスト ID の組み合わせに対して Assertion を発行できるのは一度だけとする |
| SAML2Core-EncryptedAssertion | excluded | MAY | 暗号化された Assertion |

### Bindings for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Bindings-RedirectPost | required | MUST | AuthnRequest は HTTP-Redirect または HTTP-POST バインディングで受理する。リクエストで指定された ProtocolBinding が HTTP-POST 以外なら拒否する。SAMLResponse と返信可能なプロトコルエラーは HTTP-POST で返す |

### Profiles for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Profile-WebBrowserSSO | required | MUST | SP 起点と IdP 起点の Web Browser SSO を提供する。未対応の ACS インデックス、NameID 形式、レスポンスバインディングはフェイルクローズで拒否する。`IsPassive=true` でログインが必要な場合は、NoPassive プロトコルレスポンスを返す |
| SAML2Profile-ECP | excluded | MAY | Enhanced Client or Proxy Profile |

### Metadata for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Metadata-IDPSSODescriptor | required | MUST | IdP メタデータでは、SSO エンドポイント、SLO エンドポイント、署名証明書、NameID 形式を公開する |
| SAML2Metadata-WantAuthnRequestsSigned | optional | MAY | SP ごとの信頼ポリシーとして、AuthnRequest / LogoutRequest の署名検証を要求できる |

## Authorization Boundary

SP と IdP プロファイルの登録・参照・削除は `AdminFederationTrustsManage` 権限 (AuthZEN action `admin:federation_trusts_manage`) を要し、`admin` ロールを持つ、有効かつ認証済みのユーザーが所属テナントに対して行える。API アクセストークンでは、ロールに加えて次のスコープをそれぞれの操作に要求する。

| スコープ | 許可する操作 |
|---|---|
| `saml:read` | ListSamlServiceProviders、ListSamlIdentityProviderProfiles |
| `saml:write` | RegisterSamlServiceProvider、DeleteSamlServiceProvider、CreateSamlIdentityProviderProfile、UpdateSamlIdentityProviderProfile、DeleteSamlIdentityProviderProfile |

プロトコルのエンドポイントは管理者の認可を通らない。IdP メタデータと証明書の取得は認証を要さない公開 Discovery であり、公開するのは entityID、エンドポイント、署名証明書に限る。SSO と SLO はブラウザーのログインセッションで主体を決め、SP の entityID、`AssertionConsumerServiceURL`、Destination、対象ユーザーの Application 割り当てをすべて検証してから発行する。1 つでも一致しなければ SAMLResponse を発行しない。

テナントとプロファイルはどちらも信頼境界である。SSO と SLO では、リクエスト先のルートが指すプロファイルと、対象 SP に割り当てられたプロファイルが一致することを確認する。ある信頼境界に対する正当なリクエストを、同じテナントの別のプロファイルへ送り直しても通らない。

## Design

### SSO Profile scope

初期対応の範囲は SAML 2.0 の Web Browser SSO Profile に限る。HTTP-Redirect（deflate と Base64）および HTTP-POST（Base64）のバインディング、署名済みの Response と Assertion、メタデータの公開、SP 起点と IdP 起点の SSO、Single Logout を提供する。SAML ECP、暗号化された Assertion、idmagic が外部 IdP に対して SAML SP として動作する外部 IdP からのフェデレーションは対象外とし、必要になった時点で別の実装単位として扱う。対応範囲を狭めることで、SAML で知られている署名ラッピング攻撃への露出を抑える。

クレームの発行と Assertion の署名には、WS-Federation と WS-Trust で共有している構築器と署名器 (`backend/wsfederation/tokens_saml`) を再利用する。これらは SAML のバージョン、Bearer SubjectConfirmation、audience の制限をすでに扱っている。この Context では署名処理を作り直さず、`InResponseTo` の対応付けなど SP 起点のフローに固有の入力だけを追加する。

デフォルトでは Assertion に署名し、Response への署名は任意に有効化できる。これは Okta や Entra が提供する「Sign Response」に相当する。`goxmldsig` は、署名対象要素の末尾に Enveloped Signature を追加する。署名後に要素を移動すると名前空間が再構成されてダイジェスト値が変わり、検証できなくなるため、署名済み要素は移動しない。この制約は Assertion と Response のどちらに署名する場合にも適用する。

相互運用時の安全性を保つ検証はドメイン層に集約し、フェイルクローズで処理する。Issuer は登録済み SP の entityID と完全に一致しなければならない。`AssertionConsumerServiceURL` は SP の許可リストと照合して任意の宛先への転送を防ぎ、audience は SP の entityID に限定する。検証結果を確定できない場合や値が一致しない場合は、すべて拒否する。

### Identity provider profiles

各サービスプロバイダーは、テナント内の IdP プロファイルのうち 1 つだけに関連付ける。テナントに必ず存在し変更できない `default` プロファイルは複数のサービスプロバイダーで共有し、短い `/saml/*` ルートを使用する。追加のプロファイルは `/saml/idp/{profile_id}/*` ルートとプロファイル固有の entityID を使用する。`shared` プロファイルは複数のサービスプロバイダーで共有できるが、`dedicated` プロファイルを割り当てられるサービスプロバイダーは最大 1 つとする。どちらも同じモデルで表し、プロトコル、永続化、管理のすべての経路で同じ信頼境界の規則を適用する。

プロファイル管理 API は、サーバーが生成した正式な entityID、メタデータ、SSO、SLO、証明書取得用の各 URL と、証明書のフィンガープリントを返す。関連付けられているサービスプロバイダーの数も返し、UI で使用中のプロファイルを削除できないようにする。ただし、最終的な整合性は Repository で保証する。デフォルトプロファイルの変更、`dedicated` プロファイルへの複数サービスプロバイダーの割り当て、使用中のプロファイルの削除はいずれも拒否する。

SSO と SLO では、リクエスト先のルートからプロファイルを特定し、対象サービスプロバイダーに関連付けられたプロファイルと一致することを確認する。Destination の検証には、そのプロファイルの正式なエンドポイントを使用する。これらを組み合わせて検証することで、ある信頼境界に対する正当なリクエストが、同じテナントの別のプロファイルを介して再送されることを防ぐ。

### Tenant signing

すべてのリクエストで、発行の直前にテナントとプロファイルに対応する署名器を取得する。署名プロバイダーは、そのスコープで有効な `XmlFederationSigning` 資格情報を選ぶ。署名プロバイダー、秘密鍵を扱う署名器、X.509 証明書のいずれかを利用できない場合は、フェイルクローズで失敗させる。リクエストごとに署名器を取得することで、プロセス内で共有された証明書がテナントやプロファイルの境界を越えて使われることを防ぐ。

各プロファイルのメタデータでは、有効な証明書に加え、同じ鍵スコープに属する有効期限内の検証用証明書をすべて公開する。新しい Assertion と Response の署名には現在有効な資格情報だけを使用するが、検証用証明書を重複して公開することで、サービスプロバイダーはローテーション直前に発行されたメッセージも検証できる。XML の構文解析と正規化には、XML 署名用に選定した検証済みライブラリを使用する。

### Persistence

`saml_authnrequest_replays` は、AuthnRequest の ID を初めて受信したときだけ記録する。`RecordIfNew` は `INSERT ... ON CONFLICT DO NOTHING` を実行し、挿入された行数によって初回のリクエストか再送かを判定する。

### Design Decisions

- 対応範囲を Web Browser SSO Profile に限り、ECP、暗号化 Assertion、SAML SP としての動作は含めない。対応するバインディングと形式を減らすほど、SAML で知られた署名ラッピング攻撃への露出が小さくなるからである。
- SAML IdP プロファイルは共有可能な 1 つのモデルとし、専用プロファイルを別の型にしない。`dedicated` は同じモデルに SP を 1 つだけ関連付けた状態として表す。信頼境界の規則をプロトコル、永続化、管理のすべての経路で 1 つに保つためである。
- XML の構文解析、正規化、署名は自作せず、検証済みの第三者製ライブラリに委ねる。

## Scenarios

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
