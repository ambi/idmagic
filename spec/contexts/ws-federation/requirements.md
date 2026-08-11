# WsFederation Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-WSFEDERATION-001: management API clientはWS-Fed scope内のtrustだけを操作できる
- Actor: ManagementApiClient
- Given: client は対象 tenant の有効な API access token を提示している
- Then: wsfed:read scope で relying party を参照できる
- Then: wsfed:write scope で relying party と Entra federation を変更できる
- Alternative (wsfed:read だけで変更操作を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant と request tenant が一致しない): 操作は AccessDeniedError で拒否される

### REQ-WSFEDERATION-002: WS-Federation passive sign-in succeeds
- Actor: EndUser
- Given: wtrealm と wreply は登録済みで subject は対象 Application に割り当てられている
- Then: 登録済み RP の wsignin1.0 を受信する
- Then: wtrealm / wreply / wfresh / subject assignment を検証する
- Then: 署名済み assertion を RSTR form で返し wctx を同値で往復する
- Alternative (wfresh より認証が古い): token を発行せず再認証へ誘導する
- Alternative (wtrealm、wreply、wauth、subject assignment のいずれかが不正である): WsFedSignInRejected を発行して fail-closed で拒否する

### REQ-WSFEDERATION-003: WS-Federation passive sign-in rejects untrusted target
- Actor: EndUser
- Given: wtrealm が未登録、wreply が許可外、または subject が未割当である
- Then: 不正な wsignin1.0 を受信する
- Then: token を発行せず WsFedSignInRejected を発行する

### REQ-WSFEDERATION-004: WS-Trust Issue succeeds
- Actor: SecurityTokenRequester
- Given: UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- Then: WS-Trust Issue RST を受信する
- Then: UsernameToken と RST の閉集合条件を検証する
- Then: RSTR を返す
- Alternative (MessageID が assertion lifetime 内に再利用されている): WsTrustTokenRejected を発行して protocol error を返す
- Alternative (UsernameToken credential が不正である): AccessDeniedError を返し token を発行しない

### REQ-WSFEDERATION-005: WS-Trust Issue rejects invalid envelope
- Actor: SecurityTokenRequester
- Given: RST の To、MessageID、AppliesTo、Action、RequestType、KeyType のいずれかが不正である
- Then: 不正な RST を受信する
- Then: WsTrustTokenRejected を emit して 400 または 401 を返す

### REQ-WSFEDERATION-006: WsFederationSignIn
WS-Federation passive requestor profile の単一 HTTP endpoint。wa で sign-in と sign-out を dispatch する。sign-in は wtrealm で relying party を解決し、wreply を許可集合に限定する (open redirect 防止)。未認証はログインへ誘導し、認証済みなら relying party の claim policy で発行した claim を署名済み SAML assertion (既定 SAML 1.1 / Entra 互換、RP 設定で SAML 2.0) に載せ、RSTR に包んで自動 POST する。wfresh が示す最大経過時間より認証が古ければ再認証へ誘導し、wauth が要求する認証方式を尊重する。sign-out の意味契約は WsFederationSignOut が所有する。
- Precondition: input.wa == "wsignin1.0" || input.wa == "wsignout1.0" || input.wa == "wsignoutcleanup1.0"
- Precondition: input.wa != "wsignin1.0" || wtrealm_registered(input.wtrealm, context.tenant_id)
- Precondition: input.wa != "wsignin1.0" || reply_url_allowed(input.wtrealm, input.wreply, context.tenant_id)
- Postcondition: authentication_fresh_enough(input.wfresh)
- Postcondition: requested_authentication_method_satisfied(input.wauth)
- Postcondition: subject_assigned_to_application(input.wtrealm, context.tenant_id)
- Postcondition: wctx_round_tripped(input.wctx)

### REQ-WSFEDERATION-007: WsFederationSignOut
WsFederationSignIn が所有する単一 passive HTTP endpoint 内の sign-out 意味契約。ローカルセッションを破棄する。wsignout1.0 は許可済み wreply へのリダイレクトまで行い、wsignoutcleanup1.0 は破棄のみで 200 を返す。
- Precondition: input.wa == "wsignout1.0" || input.wa == "wsignoutcleanup1.0"
- Precondition: input.wtrealm == null || wtrealm_registered(input.wtrealm, context.tenant_id)
- Precondition: input.wreply == null || reply_url_allowed(input.wtrealm, input.wreply, context.tenant_id)
- Postcondition: local_session_cleared
- Postcondition: no_unregistered_redirect

### REQ-WSFEDERATION-008: WsTrustIssue
WS-Trust 1.3 active STS Issue endpoint。UsernameToken を検証し、 To を現在 realm の active STS endpoint、Action / RequestType を Issue、KeyType を Bearer に限定し、AppliesTo を登録済み WsFedRelyingParty に解決する。RP の claim policy で組み立てた claim を署名済み SAML assertion に載せ、RSTR として返す。未登録 AppliesTo、期限切れ Timestamp、不正な To / Action / RequestType / KeyType / TokenType は fail-closed で拒否する。
- Precondition: input.request.to == active_sts_endpoint(context.tenant_id)
- Precondition: input.request.action == "Issue"
- Precondition: input.request.request_type == null || input.request.request_type == "Issue"
- Precondition: input.request.key_type == null || input.request.key_type == "Bearer"
- Precondition: applies_to_registered(input.request.applies_to, context.tenant_id)
- Precondition: message_id_not_seen_within_replay_window(input.request.message_id)
- Precondition: timestamp_is_valid(input.request.created_at, input.request.expires_at)
- Postcondition: output.response.relates_to == input.request.message_id
- Postcondition: output.response.applies_to == input.request.applies_to

### REQ-WSFEDERATION-009: WsTrustMetadataExchange
WS-Trust MEX endpoint。active STS の usernamemixed endpoint、UsernameToken policy、WS-Addressing / SOAP binding を realm issuer から派生して広告する。

### REQ-WSFEDERATION-010: PublishWsFederationMetadata
WS-Federation / WS-Trust metadata publication。realm entityID、passive endpoint、 active STS endpoint、MEX endpoint、tenant の XmlFederationSigning 証明書を AD FS 互換 federationmetadata.xml として公開する。active 証明書に加え rotation grace 中の verifying 証明書も掲載する。

### REQ-WSFEDERATION-011: ConfigureEntraFederation
Microsoft Entra domain federation preset を設定する。検証済み domain、 IssuerUri、sourceAnchor 属性を受け取り、sourceAnchor が既存 user で欠落・重複しないこと、 GUID または base64 ImmutableID として正規化できることを検証する。成功時は UPN / ImmutableID / persistent NameID の claim preset を持つ WsFedRelyingParty を upsert し、Entra に登録する PassiveLogOnUri / ActiveLogOnUri / MetadataExchangeUri を返す。 Hybrid Azure AD Join device registration は windowstransport + コンピュータアカウント Kerberos を要するため未提供として診断に含める。
- Precondition: source_anchor_is_unique_and_normalizable(input.source_anchor_attribute, context.tenant_id)

### REQ-WSFEDERATION-012: RegisterWsFedRelyingParty
管理者が WS-Federation relying party trust を登録または更新する。wtrealm、reply_urls、audience、token_type、claim_policy を契約として保存し、reply_urls は空でない閉集合にする。管理操作は AdminFederationTrustsManage で保護する。
- Precondition: input.relying_party.tenant_id == context.tenant_id
- Precondition: input.relying_party.reply_urls.size > 0

### REQ-WSFEDERATION-013: ListWsFedRelyingParties
管理者が所属テナントの WS-Federation relying party trust を一覧する。管理操作は AdminFederationTrustsManage で保護する。
- Postcondition: relying_parties_belong_to_tenant(output.relying_parties, context.tenant_id)

### REQ-WSFEDERATION-014: DeleteWsFedRelyingParty
管理者が所属テナントの catalog 外 WS-Federation relying party trust を wtrealm で削除する。application_id が非 NULL の RP は ApplicationOwnedProtocolError (HTTP 409) で拒否する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| WsFederation | WS-Federation Passive Requestor Profile と WS-Trust active STS を組み合わせた WS-* protocol family。UsernameToken による Issue、WS-Addressing の MessageID / To / Action、Bearer SAML assertion 発行を扱う。WindowsTransport / Kerberos、silent sign-in は初期範囲外。 | WS-Federation, WS-Trust, WS-Fed |
| EndUser | WS-Federation passive sign-in / sign-out をブラウザで開始する利用者。 |  |
| SecurityTokenRequester | UsernameToken credential を提示して WS-Trust Issue を呼び出す active client。 |  |

## Standards

### Web Services Federation Language (WS-Federation) Version 1.2

1.2 — https://docs.oasis-open.org/wsfed/federation/v1.2/ws-federation.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSFed-PassiveSignIn | required | MUST | wsignin1.0 で登録済み wtrealm と許可済み wreply にだけ token を返す |
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
| WSAddressing-MessageIDToAction | required | MUST | MessageID は replay 防止、To は active STS endpoint、Action は Issue として検証する |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
