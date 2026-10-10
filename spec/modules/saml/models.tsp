namespace IdMagic.Contract {

@doc("プロトコルのアサーションがサブジェクトをどう名指し、どのクレームを含むか。SAML と WS-Federation の Relying Party の契約は同じポリシーを復号するので、どちらもここを指す。`NameIdConfiguration` と `ClaimMappingRule` は ClaimMapping モジュールのものであり、このモデルは送受信される組に名前を付けるだけである。")
model ClaimMappingPolicy {
  name_id: NameIdConfiguration;
  rules?: ClaimMappingRule[];
}


@doc("SAML のサービスプロバイダーは Application が所有しており、単独では削除できない。代わりに Application を削除する。HTTP 409 で返す。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:application_owned_protocol` である。")
model ApplicationOwnedProtocolError is ProblemDetails;


@doc("SAML IdP プロファイルはサービスプロバイダーから参照されており、削除できない。HTTP 409 で返す。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:profile_in_use` である。")
model IdPProfileInUseError is ProblemDetails;


@doc("テナントのデフォルトの SAML IdP プロファイルは、変更も削除もできない。HTTP 409 で返し、`type` は使用中の拒否と同じコードの `urn:idmagic:error:profile_in_use` である。")
model DefaultIdPProfileError is ProblemDetails;


@doc("SAML 2.0 のメッセージのバインディング。SP 起点の AuthnRequest の受信と SAMLResponse の返信に使う。HTTP-Redirect は deflate+base64 を GET のクエリに入れ、HTTP-POST は base64 を自動送信するフォームで運ぶ。")
enum SamlBinding {
  urn_oasis_names_tc_SAML_2_0_bindings_HTTP_Redirect: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
  urn_oasis_names_tc_SAML_2_0_bindings_HTTP_POST: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
}


@doc("発行するアサーションの NameID の形式。SP のメタデータと AuthnRequest の NameIDPolicy のどちらも求めることができる。AuthnRequest が unspecified 以外を指定したときはそれに従い、何も指定しないときは SP のクレームのポリシーで設定した形式を使う。")
enum SamlNameIDFormat {
  urn_oasis_names_tc_SAML_2_0_nameid_format_unspecified: "urn:oasis:names:tc:SAML:2.0:nameid-format:unspecified",
  urn_oasis_names_tc_SAML_1_1_nameid_format_emailAddress: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
  urn_oasis_names_tc_SAML_2_0_nameid_format_persistent: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
  urn_oasis_names_tc_SAML_2_0_nameid_format_transient: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
}


@doc("IdP プロファイルを複数の SP で共有してよいかどうか。")
enum SamlIdentityProviderProfileMode {
  shared: "shared",
  dedicated: "dedicated",
}


@doc("テナントの中の SAML IdP の信頼の境界。デフォルトのプロファイルは、テナント全体の entityID と既存の SAML のエンドポイントを使う。追加のプロファイルは、それぞれプロファイルの id を含む、固有の entityID、エンドポイント、XmlFederationSigning の鍵のスコープを得る。")
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


@doc("管理コンソール向けの IdP プロファイルの表現。永続化したプロファイルに加えて、リクエストのテナントの正規の発行者からサーバーが導いた設定と、現在割り当てられている SP の数を返す。")
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


@doc("SAML 2.0 の Web Browser SSO のサービスプロバイダー（SP）の登録。entityID で識別し、許可する AssertionConsumerService の URL の閉じた集合、オーディエンス、NameID の形式、署名のポリシー、クレームの発行のポリシーをまとめる。`application_id` が null のカタログの外の SP も許す。null でないときは同じテナントの `saml` の Application を指し、一つの Application に結び付く SP は一つまでである。")
model SamlServiceProvider {
  @doc("属するテナント。SP の登録と SAML のエンドポイントはレルムごとに分かれる。")
  tenant_id: string = "default";
  @doc("テナントの中でこの SP に割り当てた SAML IdP プロファイル。SSO または SLO のリクエストは、経路のプロファイルと一致するときだけ受け付ける。")
  @minLength(1)
  idp_profile_id: string = "default";
  @doc("SP の識別子（SAML の entityID）。AuthnRequest の Issuer と完全に一致しなければならない。saml-schema-metadata-2.0.xsd の entityIDType は 1024 文字と定めるが、仕様に従わない SP を拒否しないように、ここでの上限はその外に置く。btree のインデックスの行に値を収めるためのリソースの制限として、別に 2048 バイトの上限を適用する。")
  @minLength(1)
  @maxLength(2048)
  entity_id: string;
  @doc("対応する Application。通常の CRUD はこれを変えず、関係を管理するのは Application の一括作成と一括削除だけである。")
  application_id?: string;
  @doc("管理 UI での表示名。")
  display_name?: string;
  @doc("許可する AssertionConsumerService の URL の閉じた集合。AuthnRequest が ACS の URL を指定したときは、この集合との完全一致だけを受け付け、ACS の URL と index がどちらもないときだけ、最初の要素をデフォルトの返信先にする。index の解決は、最初の実装では提供しない。")
  acs_urls: string[];
  @doc("Single Logout の返信を送る、SP の SingleLogoutService の URL。")
  slo_url?: string;
  @doc("発行するアサーションの AudienceRestriction。空のときは entity_id を使う。")
  audience?: string;
  @doc("この SP に対する NameID と属性（クレーム）の発行のポリシー。")
  claim_policy: ClaimMappingPolicy;
  @doc("発行する Assertion がエンベロープ署名を含むかどうか。デフォルトでは含む。")
  sign_assertion: boolean = true;
  @doc("Response 全体がエンベロープ署名を含むかどうか。「レスポンスに署名する」ことを求める SP では有効にする。")
  sign_response: boolean = false;
  @doc("AuthnRequest または LogoutRequest の署名を検証する、SP ごとのポリシー。true のときは、Redirect バインディングの署名か XML 署名のどちらかを `authn_request_signing_certificate_pem` で検証する。IdP のメタデータの WantAuthnRequestsSigned には反映しない。")
  want_authn_requests_signed: boolean = false;
  @doc("AuthnRequest または LogoutRequest の署名の検証に使う、SP の X.509 証明書の PEM。`want_authn_requests_signed` が true のとき必須である。")
  authn_request_signing_certificate_pem?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("SP 起点の SSO で受信した `<samlp:AuthnRequest>` から取り出した、検査すべき内容を保持する中間の形。HTTP-Redirect は deflate+base64、HTTP-POST は base64 で、どちらも解析の前に復号する。復号した XML には大きさの上限があるので、deflate 爆弾は通らない。Version は 2.0 でなければならず、IssueInstant は必須で、許容する時計のずれと最大の経過時間の両方の内になければならない。")
model SamlAuthnRequest {
  @doc("リクエストの id。SAMLResponse の InResponseTo とアサーションの SubjectConfirmationData で往復する。")
  @minLength(1)
  id: string;
  @doc("SP の entityID。登録済みの SamlServiceProvider へ解決する。")
  @minLength(1)
  issuer: string;
  @doc("`AuthnRequest/@Version`。`2.0` だけを受け付ける。")
  @minLength(1)
  version: string;
  @doc("`AuthnRequest/@IssueInstant`。未来へのずれまたは最大の経過時間を超えるリクエストは拒否する。")
  issue_instant: utcDateTime;
  @doc("リクエストが指定する AssertionConsumerService の URL。許可する集合との完全一致だけを受け付ける。")
  acs_url?: string;
  @doc("AssertionConsumerServiceIndex。ACS の URL と併用すると拒否し、最初の実装は明示的な index を解決しないので、指定すること自体も拒否する。")
  acs_index?: int64;
  @doc("`AuthnRequest/@ProtocolBinding`。省略または HTTP-POST だけを受け付ける。")
  protocol_binding?: SamlBinding;
  @doc("リクエストの宛先の IdP の SSO のエンドポイントの URL。")
  destination?: string;
  @doc("`NameIDPolicy/@Format`。unspecified 以外は、発行するアサーションの NameID の形式になる。")
  name_id_format?: SamlNameIDFormat;
  @doc("再認証の要求。true のときは、最近再認証していない呼び出し元をサインインへ戻す。")
  force_authn: boolean = false;
  @doc("対話をしないことの要求。true で既存のセッションを使えないときは、サインインのページへ移らずに NoPassive のレスポンスを返す。")
  is_passive: boolean = false;
}


@doc("SP へ返す `<samlp:Response>`。Status の Success、IdP の entityID を保持する Issuer、署名した Assertion を含み、SP が設定していれば Response 全体にも署名する。Destination は確定した ACS の URL で、SP 起点のフローでは InResponseTo は AuthnRequest の id である。HTTP-POST バインディングで base64 に符号化し、ACS へ自動 POST する。")
model SamlResponse {
  @doc("Response の id。")
  @minLength(1)
  id: string;
  @doc("許可する集合から選んだ、確定した ACS の URL。")
  @minLength(1)
  destination: string;
  @doc("SP 起点のフローでの AuthnRequest の id。IdP 起点のフローでは空である。")
  in_response_to?: string;
  issue_instant: utcDateTime;
  @doc("埋め込む署名したアサーション。")
  assertion: SamlAssertion;
}


@doc("SAMLResponse に埋め込む `<saml:Assertion>`。Subject（NameID と bearer の SubjectConfirmation）、AudienceRestriction、AuthnStatement、AttributeStatement を含む。bearer の SubjectConfirmationData は、Recipient（ACS）、NotOnOrAfter、SP 起点のフローでは InResponseTo を含む。")
model SamlAssertion {
  @minLength(1)
  id: string;
  @doc("IdP の entityID。")
  @minLength(1)
  issuer: string;
  @doc("SP のオーディエンス。デフォルトは entity_id である。")
  @minLength(1)
  audience: string;
  @doc("bearer の SubjectConfirmationData の Recipient、すなわち ACS の URL。")
  recipient: string;
  not_before: utcDateTime;
  not_on_or_after: utcDateTime;
  authn_instant: utcDateTime;
  name_id_format: SamlNameIDFormat;
  @minLength(1)
  name_id_value: string;
}


@doc("SAML の SSO で、署名した SAMLResponse を SP へ発行した。")
model SamlSignInIssued {
  occurredAt: utcDateTime;
  @doc("発行したテナント。")
  tenantId: string;
  @doc("発行先の SP の entityID。")
  entityId: string;
  @doc("サブジェクト。NameID が解決したユーザーである。")
  userId: string;
}


@doc("SAML の SSO のリクエストを拒否した。未登録の entityID、許可する集合の外の ACS の URL、割り当てのないサブジェクト、解析の失敗などである。")
model SamlSignInRejected {
  occurredAt: utcDateTime;
  @doc("拒否したテナント。")
  tenantId: string;
  @doc("求められた SP の entityID。判別できたときだけ含む。")
  entityId?: string;
  @doc("拒否した理由の要約。")
  reason: string;
}


@doc("SAML の Single Logout でローカルのセッションを破棄した。LogoutRequest があったときは、登録済みの SLO の URL へ LogoutResponse を返す。")
model SamlLogout {
  occurredAt: utcDateTime;
  @doc("ログアウトしたテナント。")
  tenantId: string;
  @doc("ログアウトを求めた SP の entityID。判別できたときだけ含む。")
  entityId?: string;
}


@doc("SAML のサービスプロバイダーを登録または更新（upsert）する JSON のリクエスト。呼び出し元が設定してよいものだけを含み、`application_id`、テナント、タイムスタンプはサーバーが管理する。")
model SamlServiceProviderRequest {
  @doc("SP の識別子（SAML の entityID）。AuthnRequest の Issuer と完全に一致しなければならない。")
  entity_id: string;
  @doc("管理 UI での表示名。")
  display_name: string;
  @doc("この SP に割り当てる、同じテナントの SAML IdP プロファイル。")
  idp_profile_id?: string;
  @doc("許可する AssertionConsumerService の URL の閉じた集合。")
  acs_urls: url[];
  @doc("SP の SingleLogoutService の URL。")
  slo_url?: url;
  @doc("発行するアサーションの AudienceRestriction。空のときは entity_id を使う。")
  audience?: string;
  @doc("この SP に対する NameID と属性（クレーム）の発行のポリシー。")
  claim_policy: ClaimMappingPolicy;
  @doc("発行する Assertion がエンベロープ署名を含むかどうか。省くと含む。")
  sign_assertion?: boolean;
  @doc("Response 全体がエンベロープ署名を含むかどうか。")
  sign_response?: boolean;
  @doc("AuthnRequest と LogoutRequest の署名を検証するかどうか。")
  want_authn_requests_signed?: boolean;
  @doc("それらの署名の検証に使う、SP の X.509 証明書の PEM。`want_authn_requests_signed` が true のとき必須である。")
  authn_request_signing_certificate_pem?: string;
}


@doc("追加の SAML IdP プロファイルを作成または更新する JSON のリクエスト。デフォルトのプロファイルは、この方法では作成も変更もできない。")
model SamlIdentityProviderProfileRequest {
  @doc("プロファイルの表示名。")
  name: string;
  @doc("プロファイルを複数の SP で共有してよいかどうか。")
  mode: SamlIdentityProviderProfileMode;
}


@doc("リクエストが指定した SAML の ID プロバイダーのプロファイルが、このテナントに存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:not_found` である。")
model SamlIdentityProviderProfileNotFoundError is ProblemDetails;

}
