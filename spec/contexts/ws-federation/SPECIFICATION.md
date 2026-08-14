---
context: ws-federation
updated_at: 2026-08-11
---

# WsFederation Specification

## Overview

受動的な WS-Federation と能動的な WS-Trust STS について、信頼関係、AD FS 互換の `federationmetadata.xml`、MEX、RST と RSTR、WS-Fed リライングパーティーとの関連付けを所有する。SAML Assertion、クレームの公開、署名鍵のライフサイクルには共有機能を利用する。プロトコルに依存しないクレーム発行は `ClaimMapping` と共有し、XML Assertion の署名は `tokens_saml` アダプターと共有する。SAML 2.0 SP との関連付けは Saml Context が所有する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| WsFederation | WS-Federation Passive Requestor Profile と WS-Trust の能動的 STS を組み合わせた WS-* プロトコル群。UsernameToken による Issue、WS-Addressing の MessageID / To / Action、Bearer SAML Assertion の発行を扱う。WindowsTransport / Kerberos とサイレントサインインは初期範囲外とする。 | WS-Federation, WS-Trust, WS-Fed |
| EndUser | WS-Federation passive sign-in / sign-out をブラウザで開始する利用者。 |  |
| SecurityTokenRequester | UsernameToken 資格情報を提示して WS-Trust Issue を呼び出す能動的クライアント。 |  |

## Standards

### Web Services Federation Language (WS-Federation) Version 1.2

1.2 — https://docs.oasis-open.org/wsfed/federation/v1.2/ws-federation.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSFed-PassiveSignIn | required | MUST | wsignin1.0 で登録済み wtrealm と許可済み wreply にだけトークンを返す |
| WSFed-SilentSignIn | excluded | MAY | silent sign-in / prompt=none 相当の無音認証 |

### WS-Trust 1.3

1.3 — https://docs.oasis-open.org/ws-sx/ws-trust/v1.3/ws-trust.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSTrust13-IssueBearer | required | MUST | Issue 要求に対して Bearer SAML assertion を RSTR で返す |
| WSTrust13-WindowsTransport | excluded | MAY | WindowsTransport / Kerberos based active profile |

### Web Services Security UsernameToken Profile 1.1.1

1.1.1 — https://docs.oasis-open.org/wss-m/wss/v1.1.1/os/wss-UsernameTokenProfile-v1.1.1-os.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSS-UsernameTokenPassword | required | MUST | WS-Trust active STS は UsernameToken username/password を認証する |

### Web Services Addressing 1.0 - Core

1.0 — https://www.w3.org/TR/ws-addr-core/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSAddressing-MessageIDToAction | required | MUST | MessageID はリプレイ防止のために検証し、To は能動的 STS のエンドポイント、Action は Issue として検証する |

## Authorization Boundary

認可の意味はアプリケーションとそのテストで保証する。本仕様では API の認証方式を定めるが、ポリシー用の DSL はあえて定義しない。ポリシー言語を採用する前に、別の作業項目で Cedar を評価する。

## Design

### Internal Interfaces

#### WsFederationSignOut
`WsFederationSignIn` が所有する単一のパッシブ HTTP エンドポイントにおけるサインアウトの意味を定める。ローカルセッションを破棄し、`wsignout1.0` では許可済みの `wreply` へのリダイレクトまで行い、`wsignoutcleanup1.0` では破棄だけを行って 200 を返す。
- Input invariant: input.wa == "wsignout1.0" || input.wa == "wsignoutcleanup1.0"
- Input invariant: input.wtrealm == null || wtrealm_registered(input.wtrealm, context.tenant_id)
- Input invariant: input.wreply == null || reply_url_allowed(input.wtrealm, input.wreply, context.tenant_id)
- Result invariant: local_session_cleared
- Result invariant: no_unregistered_redirect

### Tenant signing

受動的な発行でも能動的な発行でも、発行時にリクエスト元テナントの有効な `XmlFederationSigning` 資格情報を取得する。フェデレーションメタデータでは、広告する役割ごとに現在有効な証明書と有効期限内の検証用証明書を公開する。これにより、リライングパーティーは計画されたローテーションの前後でも検証を継続できる。

署名の提供元は、プロセスの起動時の状態ではなく `SigningKeys` を裏に持つ。これにより WS-Fed と SAML は 1 つの XML の資格情報のライフサイクルに乗りつつ、OAuth2 と JWT の鍵からの分離を保てる。

### Federation metadata

この Context は、各レルムの `/{realm}/federationmetadata/2007-06/federationmetadata.xml` で AD FS 互換の `federationmetadata.xml` を公開し、テナントの発行者（デフォルトテナントでは `/realms/default`）を entityID として広告する。これにより、WS-Fed リライングパーティーと Microsoft Entra のドメインフェデレーションは、別の導入手順を使わずに発行者、エンドポイント、署名証明書を検出できる。

`EntityDescriptor` は `SecurityTokenServiceType` と `ApplicationServiceType` の両方の `RoleDescriptor` を持ち、`PassiveRequestorEndpoint`、`SecurityTokenServiceEndpoint`、`MetadataEndpoint`、そして署名の `KeyDescriptor` を広告する。署名の証明書は OAuth と OIDC の JWK の形を再利用せず、WS-* が本来使う X.509 の形で公開する。鍵の用途、ローテーション、重なりは `SigningKeys` の責務のままであり、メタデータは WS-* の利用者が既に期待するものを広告すれば足りるからである。`/{realm}/trust/mex` は能動的な STS (下記) の探索として、`usernamemixed` のエンドポイントと UsernameToken を必須とするその方針を公開する。RST と RSTR のやり取り自体はメタデータの一部ではない。

クレームの公開は宣言的に定義する。AD FS のクレーム規則言語は採用せず、`ClaimMappingPolicy` を WS-Fed、WS-Trust、SAML で共有する。対応付けたクレームの集合には AD FS 規則言語ほどの表現力は不要であり、検証コストだけが増えるためである。対応付けのない属性は決して発行しない。

### WS-Trust active STS scope

能動的な WS-Trust への対応は、一般的な相互運用ではなく Microsoft 365 風のリッチクライアントのサインインを狙う。SOAP、WS-Security、WS-Addressing、SAML の署名は互いに十分重なっており、束縛を広く覆うと再送と XML の包み替えの危険が実質的に高まるので、最初の範囲は意図的に狭くする。

能動的な STS のエンドポイントは `/trust/usernamemixed` だけであり、受け付けるのは WS-Trust 1.3 の `Issue` のみである。`Validate`、`Renew`、`Cancel` は実装しない。認証は UsernameToken のみで、既存の `UserRepository`、`PasswordHasher`、`LoginAttemptThrottle` に対して検証する。Kerberos と IWA の `windowstransport` は範囲外であり、別のスライスへ残す。

WS-Addressing と WS-Security の必須要素（`MessageID`、`To`、`Action`、UsernameToken、Timestamp、`AppliesTo`）はフェイルクローズで検証する。Timestamp は期限切れの値と遠い未来の値を拒否し、`MessageID` は有効期間の短いリプレイ防止ストアに記録する。`AppliesTo` は登録済みの WS-Fed リライングパーティーに解決できなければならず、未登録の宛先は拒否する。発行する Assertion の audience と recipient はその RP に限定し、クレームは RP の `ClaimMappingPolicy` を通じて発行する。これにより、リプレイや audience の取り違えがリライングパーティーの境界を越えることを防ぐ。RSTR は SOAP 1.2 で署名済み SAML Assertion を返し、RST が SAML 1.1 または SAML 2.0 を明示的に要求しない場合は SAML 1.1 をデフォルトとする。

### Entra domain federation profile

`EntraFederationProfile` は、WS-Federation リライングパーティー用の定型設定である。ドメイン、IssuerUri、sourceAnchor 属性、受動・能動・MEX の各エンドポイントを受け取り、wtrealm と audience に同じ IssuerUri を持つ `WsFedRelyingParty` を作成または更新する。定型設定にすることで、クレーム設定の JSON を手書きする必要がなくなる。手書きの設定を誤ると Entra 側では原因を特定しにくく、設定時に sourceAnchor の安定性や一意性も保証できない。

必須クレームは定型設定で固定し、フェイルクローズで扱う。UPN は `preferred_username` から `http://schemas.xmlsoap.org/claims/UPN` として発行する。ImmutableID は正規化した sourceAnchor（`entra_immutable_id`）から導き、永続的な NameID と `http://schemas.xmlsoap.org/claims/nameidentifier` の両方に含める。sourceAnchor は、プロファイル設定時には既存ユーザーの欠落、重複、変換できない値を拒否し、発行時には対象ユーザーの ImmutableID を導けない場合にクレーム発行を拒否する。

GUID の形をした sourceAnchor の値は、ImmutableID として使う前に .NET の `Guid.ToByteArray()` のバイト順 — AD FS と Entra の慣行 — で base64 に符号化する。既に base64 の値はそのまま通す。このバイト順を誤ると、Entra は assertion を元の社内の同じユーザーへ関連付けられず、アカウントの重複やサインインの失敗を招く。プロファイルのデフォルトのトークンの型は SAML 1.1 であり、Entra と AD FS の WS-Fed のデフォルトに合わせている。Hybrid Azure AD Join の端末の登録 (`windowstransport` とコンピューターアカウントの Kerberos) は明確に範囲外であり、設定の案内では managed や PHS、あるいは AD FS の併存の配備へ誘導する。

### Design Decisions

- フェデレーションメタデータの公開とクレーム対応付けの所有を分ける。`WsFederation` が Discovery 情報（発行者、エンドポイント、署名証明書）を公開し、`ClaimMapping` が WS-Fed、WS-Trust、SAML に共通するクレーム公開ポリシーを所有する。
- 能動的な WS-Trust の STS への対応は `/trust/usernamemixed` の `Issue` のみに限る。一般的な WS-Trust の相互運用ではなく、Microsoft 365 風のリッチクライアントのサインインを狙う。
- Microsoft Entra のドメインフェデレーションプロファイルは、手書きのクレーム設定ではなく、UPN と ImmutableID のクレーム形式および sourceAnchor 検証を固定した定型のリライングパーティー設定とする。設定ミスが Entra 側で原因を特定しにくい障害として現れることを防ぐためである。

## Scenarios

### REQ-WSFEDERATION-001: management API クライアントはWS-Fed スコープのtrustだけを操作できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API access トークンを提示している
- WHEN クライアントがリライングパーティーまたは Entra フェデレーションの操作をリクエストする
  - ALT wsfed:read だけで変更操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントとリクエスト先のテナントが一致しない → 操作を AccessDeniedError で拒否する
- THEN `wsfed:read` スコープではリライングパーティーの参照だけを許可する
- THEN `wsfed:write` スコープではリライングパーティーと Entra フェデレーションの変更だけを許可する

### REQ-WSFEDERATION-002: WS-Federation passive sign-in succeeds
- ACTOR EndUser
- GIVEN wtrealm と wreply は登録済みで subject は対象 Application に割り当てられている
- WHEN 登録済み RP の wsignin1.0 を受信する
- THEN wtrealm / wreply / wfresh / subject assignment を検証する
  - ALT wfresh より認証が古い → トークンを発行せず再認証へ誘導する
  - ALT wtrealm、wreply、wauth、対象者の割り当てのいずれかが不正である → WsFedSignInRejected を発行してフェイルクローズで拒否する
- THEN 署名済み Assertion を RSTR フォームで返し、wctx を同じ値で返す

### REQ-WSFEDERATION-003: WS-Federation passive sign-in rejects untrusted target
- ACTOR EndUser
- GIVEN wtrealm が未登録、wreply が許可外、または subject が未割当である
- WHEN 不正な wsignin1.0 を受信する
- THEN トークンを発行せず WsFedSignInRejected を発行する

### REQ-WSFEDERATION-004: WS-Trust Issue succeeds
- ACTOR SecurityTokenRequester
- GIVEN UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- WHEN WS-Trust Issue RST を受信する
- THEN UsernameToken と RST の閉集合条件を検証する
  - ALT MessageID が Assertion の有効期間内に再利用されている → WsTrustTokenRejected を発行してプロトコルエラーを返す
  - ALT UsernameToken credential が不正である → AccessDeniedError を返しトークンを発行しない
- THEN RSTR を返す

### REQ-WSFEDERATION-005: WS-Trust Issue は不正なエンベロープを拒否する
- ACTOR SecurityTokenRequester
- GIVEN RST の To、MessageID、AppliesTo、Action、RequestType、KeyType のいずれかが不正である
- WHEN 不正な RST を受信する
- THEN WsTrustTokenRejected を発行し、400 または 401 を返す
