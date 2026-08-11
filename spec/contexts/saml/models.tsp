namespace IdMagic.Contract {

@doc("ClaimMapping context が publish する trust ごとの claim 発行規則の最小 stub。")
model ClaimMappingPolicy {
  configuration?: unknown;
}


@doc("SAML service provider が Application に所有されているため直接削除できない。HTTP 409 とし、Application 削除を利用する。")
model ApplicationOwnedProtocolError {

}


@doc("SAML IdP profileがSPから参照されているため削除できない。HTTP 409とする。")
model IdPProfileInUseError {

}


@doc("テナントのdefault SAML IdP profileは変更または削除できない。")
model DefaultIdPProfileError {

}


@doc("SAML 2.0 message binding。SP-initiated の AuthnRequest 受信と SAMLResponse 返送に用いる。HTTP-Redirect は GET でクエリに deflate+base64 を載せ、HTTP-POST は base64 を自動 POST フォームで運ぶ。")
enum SamlBinding {
  urn_oasis_names_tc_SAML_2_0_bindings_HTTP_Redirect: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
  urn_oasis_names_tc_SAML_2_0_bindings_HTTP_POST: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
}


@doc("発行 assertion の NameID format。SP metadata と AuthnRequest の NameIDPolicy で要求しうる。AuthnRequest が unspecified 以外を指定した場合はそれを尊重し、未指定なら SP の claim policy に定めた format を用いる。")
enum SamlNameIDFormat {
  urn_oasis_names_tc_SAML_2_0_nameid_format_unspecified: "urn:oasis:names:tc:SAML:2.0:nameid-format:unspecified",
  urn_oasis_names_tc_SAML_1_1_nameid_format_emailAddress: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
  urn_oasis_names_tc_SAML_2_0_nameid_format_persistent: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
  urn_oasis_names_tc_SAML_2_0_nameid_format_transient: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
}


@doc("IdP profileを複数SPで共有できるかを表す。")
enum SamlIdentityProviderProfileMode {
  shared: "shared",
  dedicated: "dedicated",
}


@doc("テナント内のSAML IdP trust境界。default profileはテナント共通entityIDと既存SAMLエンドポイントを使う。追加profileはprofile IDを含む固有entityID・エンドポイント・XmlFederationSigning鍵scopeを持つ。")
model SamlIdentityProviderProfile {
  tenant_id: string = "default";
  @minLength(1)
  profile_id: string;
  @minLength(1)
  name: string;
  mode: SamlIdentityProviderProfileMode;
  is_default: boolean = false;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("管理画面向けIdP profile表現。永続profileに加え、request tenantのcanonical issuerからserverが生成した設定値と現在のSP割当数を返す。")
model SamlIdentityProviderProfileAdminResponse {
  profile: SamlIdentityProviderProfile;
  entity_id: string;
  metadata_url: string;
  sso_url: string;
  slo_url: string;
  signing_certificate_url: string;
  signing_certificate_fingerprint_sha256: string;
  service_provider_count: int64;
}


@doc("SAML 2.0 Web Browser SSO の service provider (SP) 登録。entityID で識別し、許可する AssertionConsumerService URL の閉集合、audience、NameID format、署名方針、claim 発行 policy を束ねる。application_id が NULL の catalog 外 SP も許容し、非 NULL の場合は同一 tenant の saml Application を参照して Application 1件に最大1 SP だけが紐づく。")
model SamlServiceProvider {
  @doc("所属テナント ID。SP 登録と SAMLエンドポイントは realm ごとに分離する。")
  tenant_id: string = "default";
  @doc("同一テナント内でこのSPに割り当てられたSAML IdP profile。SSO/SLO要求はrouteのprofileと一致する場合だけ受理する。")
  @minLength(1)
  idp_profile_id: string = "default";
  @doc("SP の識別子 (SAML entityID)。AuthnRequest の Issuer と完全一致しなければならない。")
  @minLength(1)
  entity_id: string;
  @doc("対応する Application の ID。通常 CRUD では変更せず、Application 一括作成・削除だけが関係を管理する。")
  application_id?: unknown;
  @doc("管理 UI 表示名。")
  display_name?: string;
  @doc("許可する AssertionConsumerService URL の閉集合。AuthnRequest が ACS URL を指定する場合はこの集合との完全一致のみ受理し、ACS URL と index がともに省略された場合だけ先頭を既定の返信先とする。初期実装では index 解決は提供しない。")
  acs_urls: string[];
  @doc("SP の SingleLogoutService URL。Single Logout の返送先。")
  slo_url?: string;
  @doc("発行 assertion の AudienceRestriction。空なら entity_id を用いる。")
  audience?: string;
  @doc("この SP 向けの NameID と attribute (claim) 発行 policy。")
  claim_policy: ClaimMappingPolicy;
  @doc("発行する Assertion を enveloped 署名するか。既定で署名する。")
  sign_assertion: boolean = true;
  @doc("Response 全体を enveloped 署名するか。SP が \"Sign Response\" を要求する場合に有効化する。")
  sign_response: boolean = false;
  @doc("SP ごとの AuthnRequest / LogoutRequest 署名検証 policy。true の場合は authn_request_signing_certificate_pem で Redirect binding 署名または XML 署名を検証する。IdP metadata の WantAuthnRequestsSigned には反映しない。")
  want_authn_requests_signed: boolean = false;
  @doc("AuthnRequest / LogoutRequest 署名検証に使う SP の X.509 証明書 PEM。want_authn_requests_signed=true の場合は必須。")
  authn_request_signing_certificate_pem?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("SP-initiated SSO で受信した <samlp:AuthnRequest> から検証対象を抽出した中間表現。HTTP-Redirect は deflate+base64、HTTP-POST は base64 を復号して解析する。復号後 XML には上限を課し、deflate 爆弾を防ぐ。Version は 2.0、IssueInstant は必須で現在時刻の許容 skew と最大年齢内でなければならない。")
model SamlAuthnRequest {
  @doc("要求 ID。SAMLResponse の InResponseTo と assertion の SubjectConfirmationData に往復させる。")
  @minLength(1)
  id: string;
  @doc("SP の entityID。登録済み SamlServiceProvider に解決する。")
  @minLength(1)
  issuer: string;
  @doc("AuthnRequest/@Version。\"2.0\" だけを受理する。")
  @minLength(1)
  version: string;
  @doc("AuthnRequest/@IssueInstant。未来の skew と最大年齢を超える要求は拒否する。")
  issue_instant: utcDateTime;
  @doc("要求が指定する AssertionConsumerService URL。許可集合との完全一致のみ受理する。")
  acs_url?: string;
  @doc("AssertionConsumerServiceIndex。ACS URL との併用は拒否し、初期実装では明示的な index 指定を解決しないため拒否する。")
  acs_index?: int64;
  @doc("AuthnRequest/@ProtocolBinding。省略または HTTP-POST のみを受理する。")
  protocol_binding?: SamlBinding;
  @doc("要求が宛てた IdP の SSO endpoint URL。")
  destination?: string;
  @doc("NameIDPolicy/@Format。unspecified 以外なら発行 assertion の NameID format に反映する。")
  name_id_format?: SamlNameIDFormat;
  @doc("再認証の要求。true の場合、直近再認証済みでなければログインへ戻す。")
  force_authn: boolean = false;
  @doc("対話を禁止する要求。true で既存セッションを使えない場合はログインへ遷移せず NoPassive response を返す。")
  is_passive: boolean = false;
}


@doc("SP へ返送する <samlp:Response>。Status Success、IdP entityID の Issuer、署名済み Assertion を含み、SP 設定により Response 全体も署名する。Destination は確定した ACS URL、InResponseTo は SP-initiated の AuthnRequest ID。HTTP-POST binding で base64 化し ACS へ自動 POST する。")
model SamlResponse {
  @doc("Response ID。")
  @minLength(1)
  id: string;
  @doc("確定した ACS URL (許可集合内)。")
  @minLength(1)
  destination: string;
  @doc("SP-initiated の AuthnRequest ID。IdP-initiated では空。")
  in_response_to?: string;
  issue_instant: utcDateTime;
  @doc("埋め込む署名済み assertion。")
  assertion: SamlAssertion;
}


@doc("SAMLResponse に埋め込む <saml:Assertion>。Subject (NameID と bearer SubjectConfirmation)、AudienceRestriction、AuthnStatement、AttributeStatement を持つ。bearer SubjectConfirmationData は Recipient (ACS)、NotOnOrAfter、SP-initiated では InResponseTo を含む。")
model SamlAssertion {
  @minLength(1)
  id: string;
  @doc("IdP の entityID。")
  @minLength(1)
  issuer: string;
  @doc("SP の audience (既定 entity_id)。")
  @minLength(1)
  audience: string;
  @doc("bearer SubjectConfirmationData の Recipient (ACS URL)。")
  recipient: string;
  not_before: utcDateTime;
  not_on_or_after: utcDateTime;
  authn_instant: utcDateTime;
  name_id_format: SamlNameIDFormat;
  @minLength(1)
  name_id_value: string;
}


@doc("SAML SSO で署名済み SAMLResponse を SP へ発行した。")
model SamlSignInIssued {
  occurredAt: utcDateTime;
  @doc("発行したテナント ID。")
  tenantId: string;
  @doc("発行先 SP の entityID。")
  entityId: string;
  @doc("subject (NameID 由来の user)。")
  userId: string;
}


@doc("SAML SSO 要求を拒否した (未登録 entityID・許可外 ACS URL・未割当 subject・解析失敗など)。")
model SamlSignInRejected {
  occurredAt: utcDateTime;
  @doc("拒否したテナント ID。")
  tenantId: string;
  @doc("要求された SP entityID (判明していれば)。")
  entityId?: string;
  @doc("拒否理由の要約。")
  reason: string;
}


@doc("SAML Single Logout でローカルセッションを破棄した。LogoutRequest がある場合は LogoutResponse を登録済み SLO URL へ返す。")
model SamlLogout {
  occurredAt: utcDateTime;
  @doc("logout したテナント ID。")
  tenantId: string;
  @doc("logout を要求した SP の entityID (判明していれば)。")
  entityId?: string;
}

}
