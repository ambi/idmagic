namespace IdMagic.Contract {

@doc("WS-Federation で発行する assertion の SAML バージョン (RSTR の TokenType にもなる)。 SAML 1.1 (urn:oasis:names:tc:SAML:1.0:assertion) は Entra / AD FS の WS-Fed 既定。")
enum WsFedTokenType {
  urn_oasis_names_tc_SAML_1_0_assertion: "urn:oasis:names:tc:SAML:1.0:assertion",
  urn_oasis_names_tc_SAML_2_0_assertion: "urn:oasis:names:tc:SAML:2.0:assertion",
}


@doc("WS-Federation passive の relying party (RP) 登録。wtrealm で識別し、許可する wreply (返信先) の閉集合、audience、claim 発行 policy を束ねる。application_id が NULL の catalog 外 RP も許容し、非 NULL の場合は同一 tenant の wsfed Application を参照して Application 1件に最大1 RP だけが紐づく。")
model WsFedRelyingParty {
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("RP の識別子 (WS-Fed wtrealm / SAML audience)。")
  @minLength(1)
  wtrealm: string;
  @doc("対応する Application の ID。通常 CRUD では変更せず、Application 一括作成・削除だけが関係を管理する。")
  application_id?: unknown;
  @doc("管理 UI 表示名。")
  display_name?: string;
  @doc("許可する wreply の閉集合。wreply はこの集合との完全一致のみ受理する。")
  reply_urls: string[];
  @doc("発行 assertion の audience。空なら wtrealm を用いる。")
  audience?: string;
  @doc("発行 assertion の SAML バージョン。空なら SAML 1.1 (Entra WS-Fed 互換の既定)。")
  token_type?: WsFedTokenType;
  @doc("この RP 向けの claim 発行 policy。")
  claim_policy: ClaimMappingPolicy;
  @doc("Microsoft Entra domain federation preset。設定時に required claims と sourceAnchor 変換を束ねる。")
  entra_profile?: EntraFederationProfile;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("Microsoft Entra domain federation 用 profile。IssuerUri、sourceAnchor 属性、 passive / active / MEX endpoint を束ね、UPN / ImmutableID / persistent NameID の claim preset を WS-Fed RP に適用する。")
model EntraFederationProfile {
  @doc("Entra で検証済みの DNS domain。")
  @minLength(1)
  domain: string;
  @doc("Entra に登録する IssuerUri。発行 token の issuer / RP wtrealm と一致させる。")
  @minLength(1)
  issuer_uri: string;
  @doc("ImmutableID の元になる不変 user 属性。objectGUID の GUID 文字列または既に base64 化済みの値。")
  @minLength(1)
  source_anchor_attribute: string;
  @doc("変換後 ImmutableID を格納する合成属性名。既定 entra_immutable_id。")
  @minLength(1)
  immutable_id_attribute: string;
  @doc("Entra PassiveLogOnUri。通常 /wsfed。")
  passive_logon_uri: string;
  @doc("Entra ActiveLogOnUri。通常 /trust/usernamemixed。")
  active_logon_uri: string;
  @doc("Entra MetadataExchangeUri。通常 /trust/mex。")
  metadata_exchange_uri: string;
}


@doc("WS-Trust / WS-Federation metadata に広告する STS endpoint。受理する認証方式と binding を表す。")
model SecurityTokenEndpoint {
  @doc("公開 endpoint URL。")
  url: string;
  @doc("例: usernamemixed, mex, passive。")
  @minLength(1)
  binding: string;
  @doc("受理する認証方式。例: UsernameToken。")
  auth_methods: string[];
}


@doc("WS-Trust 1.3 Issue binding の RST から検証対象を抽出した中間表現。")
model WsTrustRequestSecurityToken {
  @doc("WS-Addressing MessageID。replay 防止に使う。")
  @minLength(1)
  message_id: string;
  @doc("WS-Addressing To。要求先 STS endpoint。")
  to: string;
  @doc("WS-Addressing Action。Issue のみ受理する。")
  action: string;
  @doc("WS-Security UsernameToken username。")
  @minLength(1)
  username: string;
  @doc("WS-Security UsernameToken password credential。")
  @minLength(1)
  password: string;
  @doc("WS-Security Timestamp Created。")
  created_at: utcDateTime;
  @doc("WS-Security Timestamp Expires。")
  expires_at: utcDateTime;
  @doc("対象 RP / audience。登録済み WsFedRelyingParty に解決する。")
  @minLength(1)
  applies_to: string;
  @doc("要求 token type。空なら RP 既定。")
  token_type?: WsFedTokenType;
  @doc("WS-Trust RequestType。空または Issue のみ受理する。")
  request_type?: string;
  @doc("WS-Trust KeyType。空または Bearer のみ受理する。")
  key_type?: string;
}


@doc("WS-Trust Issue に対する RSTR。署名済み SAML assertion と lifetime / appliesTo を含む。")
model WsTrustResponse {
  @doc("元 RST の MessageID。")
  relates_to?: string;
  @minLength(1)
  applies_to: string;
  token_type: WsFedTokenType;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  assertion_id?: string;
}


@doc("WS-Federation passive sign-in で署名済み assertion を RP へ発行した。")
model WsFedSignInIssued {
  occurredAt: utcDateTime;
  @doc("発行したテナント ID。")
  tenantId: string;
  @doc("発行先 RP の wtrealm。")
  wtrealm: string;
  @doc("subject (NameID 由来の user)。")
  userId: string;
}


@doc("WS-Federation passive sign-in を拒否した (未登録 wtrealm・許可外 wreply・検証失敗など)。")
model WsFedSignInRejected {
  occurredAt: utcDateTime;
  @doc("拒否したテナント ID。")
  tenantId: string;
  @doc("要求された wtrealm (判明していれば)。")
  wtrealm?: string;
  @doc("拒否理由の要約。")
  reason: string;
}


@doc("WS-Federation sign-out でローカルセッションを破棄した。")
model WsFedSignOut {
  occurredAt: utcDateTime;
  @doc("sign-out したテナント ID。")
  tenantId: string;
  @doc("sign-out を要求した RP の wtrealm (判明していれば)。")
  wtrealm?: string;
}


@doc("WS-Trust active STS で署名済み assertion を発行した。")
model WsTrustTokenIssued {
  occurredAt: utcDateTime;
  @doc("発行したテナント ID。")
  tenantId: string;
  @doc("発行先 RP / audience。")
  appliesTo: string;
  @doc("subject user。")
  userId: string;
}


@doc("WS-Trust active STS の Issue 要求を拒否した。")
model WsTrustTokenRejected {
  occurredAt: utcDateTime;
  @doc("拒否したテナント ID。")
  tenantId: string;
  @doc("要求された AppliesTo (判明していれば)。")
  appliesTo?: string;
  @doc("拒否理由の要約。")
  reason: string;
}


@doc("Microsoft Entra domain federation preset を設定し、対応する WS-Fed RP trust を upsert した。")
model EntraFederationConfigured {
  occurredAt: utcDateTime;
  @doc("設定したテナント ID。")
  tenantId: string;
  @doc("Entra 検証済み domain。")
  domain: string;
  @doc("Entra に登録する IssuerUri。")
  issuerUri: string;
}

}
