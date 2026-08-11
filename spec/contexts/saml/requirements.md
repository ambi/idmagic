# Saml Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-SAML-001: SPは署名証明書を取得できる
- Actor: EndUser
- Given: SP が IdMagic tenant の SAML metadata または証明書ダウンロード URL を参照できる
- Then: SP は active XmlFederationSigning 証明書を PEM で取得する
- Then: 取得した証明書は同時点の SAML metadata に公開される証明書と一致する
- Then: rotation overlap 中の全 trust 証明書は SAML metadata から取得する
- Alternative (federation signing credential が利用不能である): 証明書を返さずエラーを返す

### REQ-SAML-002: SPは割当IdP profileだけを利用できる
- Actor: EndUser
- Given: SPはprofile-aへ割り当てられている
- Given: profile-aとprofile-bは同一テナント内に存在する
- Then: profile-aのSSO endpointへSPのAuthnRequestを送る
- Then: Destination、SP Issuer、profile bindingが一体で検証される
- Then: profile-aのentityIDと署名資格情報でSAMLResponseが発行される
- Alternative (同じ要求をprofile-bのSSO endpointへ送る): SAMLResponseを発行せずSamlSignInRejectedを発行する
- Alternative (profile-aのSSO URLと異なるDestinationを指定する): fail-closedで拒否する

### REQ-SAML-003: 専用profileは固有metadataを公開する
- Actor: EndUser
- Given: テナントにdefault profileとdedicated profileが存在する
- Then: dedicated profileのmetadata URLを取得する
- Then: metadataはprofile固有entityID、SSO/SLO URL、署名証明書を公開する
- Then: default metadataとは異なる署名資格情報を公開する
- Alternative (存在しないまたは別テナントのprofile IDを指定する): metadataと証明書を公開せずnot foundを返す

### REQ-SAML-004: 管理者はSAML IdP profileを共有または専用で管理できる
- Actor: TenantAdministrator
- Given: テナントには変更不能なdefault shared profileが存在する
- Then: 管理者がread-onlyの連携エンドポイント画面から専用profile管理一覧とprofile詳細へ移動する
- Then: 管理者が専用作成画面でshared profileを作成して複数SPから選択する
- Then: 管理者がprofile詳細から専用編集画面へ移り、追加profileのnameまたはmodeを変更する
- Then: 管理者がdedicated profileを作成して1 SPへ割り当てる
- Then: 未使用の追加profileを削除する
- Alternative (profileがSPから参照中またはdefaultである): 削除をconflictで拒否する
- Alternative (dedicated profileを別のSPへも割り当てる): bindingをInvalidRequestErrorで拒否する

### REQ-SAML-005: management API clientはSAML scope内のservice providerだけを操作できる
- Actor: ManagementApiClient
- Given: client は対象 tenant の有効な API access token を提示している
- Then: saml:read scope で service provider を参照できる
- Then: saml:write scope で service provider を登録または削除できる
- Alternative (saml:read だけで変更操作を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant と request tenant が一致しない): 操作は AccessDeniedError で拒否される

### REQ-SAML-006: SAML SP initiated SSO succeeds
- Actor: EndUser
- Given: subject は認証済みで対象 Application に割り当てられている
- Given: SP の entityID、ACS URL、Destination は登録済みである
- Then: 登録済み SP の AuthnRequest を受信する
- Then: Version / IssueInstant / Issuer / ACS / Destination / binding / NameIDPolicy / subject assignment を検証する
- Then: 署名済み SAMLResponse を ACS へ POST し RelayState を同値で返す
- Then: assertion と response の署名は request tenant の active XmlFederationSigning 鍵を使用する
- Alternative (entityID、ACS、Destination、subject assignment のいずれかが不正である): SAMLResponse を発行しない → SamlSignInRejected を発行して fail-closed で拒否する
- Alternative (AuthnRequest の解析または署名検証に失敗する): SamlSignInRejected を発行して protocol error を返す
- Alternative (AuthnRequest の Version、IssueInstant、ProtocolBinding、ACS index、NameIDPolicy format が未対応または矛盾する): assertion を発行しない → 検証済み ACS が確定済みの場合だけ HTTP-POST の SAML protocol error を返す → それ以外は SamlSignInRejected を発行して fail-closed で拒否する
- Alternative (IsPassive=true かつ利用可能な既存セッションがない): ログイン画面へ遷移しない → 検証済み ACS へ HTTP-POST の NoPassive SAML protocol response を返す
- Alternative (同じ tenant / SP / AuthnRequest ID に対する assertion がすでに発行済みである): assertion を発行しない → SamlSignInRejected を発行して fail-closed で拒否する

### REQ-SAML-007: SAML rejects unregistered or mismatched request
- Actor: EndUser
- Given: AuthnRequest の entityID、ACS URL、Destination、subject assignment のいずれかが不正である
- Then: 不正な AuthnRequest を受信する
- Then: SAMLResponse を発行せず SamlSignInRejected を発行する

### REQ-SAML-008: SAML ForceAuthn redirects stale session
- Actor: EndUser
- Given: ForceAuthn=true かつ認証時刻が再認証猶予より古い
- Then: ForceAuthn=true の AuthnRequest を受信する
- Then: 古い認証コンテキストを検出する
- Then: ログインへリダイレクトする

### REQ-SAML-009: SamlSingleSignOn
SAML 2.0 Web Browser SSO の sign-in。SP-initiated は AuthnRequest を HTTP-Redirect / HTTP-POST binding で受信し、Version=2.0、IssueInstant の許容窓、Issuer、Destination、HTTP-POST response ProtocolBinding、ACS URL/index の排他と対応可否、NameIDPolicy format を検証してから service provider を解決する。routeのIdP profile、SPのprofile binding、Destinationが一致しなければfail-closedで拒否する。署名必須 SP では Redirect binding 署名または XML 署名を検証する。未対応または矛盾する要求は fail-closed に拒否し、返信可能な検証済み ACS がある場合だけ HTTP-POST の SAML protocol error を返す。ForceAuthn=true は直近再認証済みでなければログインへ誘導する。IsPassive=true で既存セッションを使えない場合はログインへ遷移せず NoPassive response を返す。IdP-initiated は entityID クエリで対象 SP を選ぶ。未認証はログインへ誘導し、認証往復をまたいで要求を保つ。認証済みなら application 割当ゲートを fail-closed で適用し、未割当 subject には発行しない。SP の claim policy で発行した NameID と attribute を bearer SAML assertion に載せ、割当profileのXML署名資格情報とSP設定に従って assertion / response を署名し、SAMLResponse を ACS へ HTTP-POST で自動送信する。tenant / SP / AuthnRequest ID は assertion 発行直前に TTL replay store へ原子的に一度だけ記録する。InResponseTo・AudienceRestriction・Recipient・Destination を整合させる。
- Precondition: input.saml_request == null || issuer_registered(input.saml_request, context.tenant_id)
- Precondition: input.saml_request == null || request_acs_allowed(input.saml_request, context.tenant_id)
- Precondition: input.saml_request == null || destination_matches_current_sso_endpoint(input.saml_request, context.tenant_id)
- Precondition: requested_profile_matches_service_provider_binding(input.profile_id, input.saml_request, input.entity_id, context.tenant_id)
- Precondition: input.saml_request == null || request_signature_satisfies_policy(input.saml_request, context.tenant_id)
- Precondition: input.saml_request == null || request_semantics_supported_and_fresh(input.saml_request, context.tenant_id)
- Postcondition: relay_state_round_tripped(input.relay_state)
- Postcondition: saml_response_matches_request(input.saml_request, context.tenant_id)
- Postcondition: assertion_issued_at_most_once_per_tenant_sp_and_request_id(input.saml_request, context.tenant_id)

### REQ-SAML-010: SamlSingleLogout
SAML Single Logout。LogoutRequest を HTTP-Redirect / HTTP-POST binding で受信し、Issuer で SP を解決し、routeのIdP profileとSPのprofile bindingを照合し、署名必須 SP では署名を検証する。Destination は選択profileの SLO endpoint、返送先は登録済み SingleLogoutService に限定する。ローカルセッションを破棄し、割当profileの鍵で署名した LogoutResponse を登録済み SLO URL へ返す。entityID / sp 指定のローカル logout も受理するが、判定不能・未登録の返送先へはリダイレクトしない (open redirect 防止)。
- Precondition: input.entity_id == null || issuer_registered(input.entity_id, context.tenant_id)
- Precondition: logout_destination_matches_current_slo_endpoint(context.tenant_id)
- Precondition: requested_profile_matches_service_provider_binding(input.profile_id, input.entity_id, context.tenant_id)
- Precondition: logout_signature_satisfies_policy(input.entity_id, context.tenant_id)
- Postcondition: local_session_cleared
- Postcondition: logout_response_destination_is_registered(input.entity_id, context.tenant_id)
- Postcondition: no_unregistered_redirect

### REQ-SAML-011: PublishSamlMetadata
SAML 2.0 IdP metadata の公開。default routeはrealmのdefault profile、profile routeは指定profileのentityID、SSO / SLO endpoint、対応 NameID format、profile scopeのXmlFederationSigning証明書を <EntityDescriptor><IDPSSODescriptor> として公開する。active 証明書に加え rotation grace 中の verifying 証明書も掲載する。

### REQ-SAML-012: DownloadSamlSigningCertificate
SAML SP への手動 trust 設定用に、request tenant の active XmlFederationSigning X.509 証明書を PEM で公開する。rotation overlap 中の verifying 証明書を含む trust 正本は PublishSamlMetadata とし、このエンドポイントは新規設定用 active 証明書だけを返す。秘密鍵は返さない。

### REQ-SAML-013: CreateSamlIdentityProviderProfile
テナント管理者が追加のsharedまたはdedicated SAML IdP profileを作成する。default profileはこの操作では作成できない。
- Precondition: input.profile.tenant_id == context.tenant_id
- Precondition: input.profile.profile_id != "default"
- Precondition: not input.profile.is_default

### REQ-SAML-014: ListSamlIdentityProviderProfiles
テナント内のSAML IdP profileをdefault先頭、name昇順で一覧する。

### REQ-SAML-015: UpdateSamlIdentityProviderProfile
テナント管理者が追加IdP profileのnameまたはmodeを変更する。default profileは変更できず、dedicatedへの変更時に複数SPから参照されていれば拒否する。

### REQ-SAML-016: DeleteSamlIdentityProviderProfile
参照されていない追加IdP profileを削除する。default profileとSPが参照中のprofileは拒否する。

### REQ-SAML-017: RegisterSamlServiceProvider
SAML service provider の登録 / 更新 (upsert)。entityID、許可 ACS URL、NameID format、audience、署名方針、claim policy を受け取り、テナント境界に閉じて保存する。entityID と ACS URL の閉集合は必須。
- Precondition: input.service_provider.tenant_id == context.tenant_id
- Precondition: input.service_provider.acs_urls.size > 0
- Precondition: not input.service_provider.want_authn_requests_signed or input.service_provider.authn_request_signing_certificate_pem != null

### REQ-SAML-018: ListSamlServiceProviders
テナント内の SAML service provider を entityID 昇順で一覧する。
- Postcondition: service_providers_belong_to_tenant(output.service_providers, context.tenant_id)

### REQ-SAML-019: DeleteSamlServiceProvider
entityID に一致する catalog 外 SAML service provider を削除する (冪等)。application_id が非 NULL の SP は ApplicationOwnedProtocolError (HTTP 409) で拒否する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Saml | SAML 2.0 IdP protocol family。Web Browser SSO Profile に基づき、SP-initiated / IdP-initiated SSO、IdP metadata 公開、署名済み SAMLResponse の発行、Single Logout、SP ごとの AuthnRequest / LogoutRequest 署名検証を扱う。encrypted assertion、ECP、SAML SP / inbound federation は初期範囲外。XML 署名・canonicalization は実績ある library に委ね、自前実装しない。 | SAML, SAML2, SAML 2.0 |
| EndUser | SAML Web Browser SSO または Single Logout をブラウザで開始する利用者。 |  |
| SamlIdentityProviderProfile | テナント内のSAML IdP entityID、エンドポイント、XML署名資格情報をまとめるtrust境界。shared profileは複数SPで共有でき、dedicated profileは最大1 SPだけに割り当てる。 | SAML IdP profile, IdP profile |

## Standards

### Assertions and Protocols for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Core-BearerAssertion | required | MUST | AuthnRequest は Version 2.0 と許容された IssueInstant を持ち、発行 assertion は Audience / Recipient / InResponseTo を検証済み要求と整合させる。同一 tenant / SP / request ID に対する assertion は一度だけ発行する |
| SAML2Core-EncryptedAssertion | excluded | MAY | encrypted assertion |

### Bindings for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Bindings-RedirectPost | required | MUST | AuthnRequest は HTTP-Redirect または HTTP-POST binding で受理し、明示された response ProtocolBinding が HTTP-POST 以外なら拒否する。SAMLResponse と返信可能な protocol error は HTTP-POST で返す |

### Profiles for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Profile-WebBrowserSSO | required | MUST | SP-initiated / IdP-initiated Web Browser SSO を提供する。未対応の ACS index、NameID format、response binding は fail-closed に拒否し、IsPassive=true でログインが必要な場合は NoPassive protocol response を返す |
| SAML2Profile-ECP | excluded | MAY | Enhanced Client or Proxy profile |

### Metadata for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Metadata-IDPSSODescriptor | required | MUST | IdP metadata は SSO endpoint、SLO endpoint、署名証明書、NameID format を公開する |
| SAML2Metadata-WantAuthnRequestsSigned | optional | MAY | SP ごとの trust policy として AuthnRequest / LogoutRequest 署名検証を要求できる |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
