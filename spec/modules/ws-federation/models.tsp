namespace IdMagic.Contract {

@doc("WS-Federation が発行するアサーションの SAML のバージョン。RSTR の TokenType にもなる。Entra と AD FS は WS-Fed のデフォルトとして SAML 1.1（urn:oasis:names:tc:SAML:1.0:assertion）を使う。")
enum WsFedTokenType {
  urn_oasis_names_tc_SAML_1_0_assertion: "urn:oasis:names:tc:SAML:1.0:assertion",
  urn_oasis_names_tc_SAML_2_0_assertion: "urn:oasis:names:tc:SAML:2.0:assertion",
}


@doc("WS-Federation のパッシブの Relying Party（RP）の登録。wtrealm で識別し、許可する `wreply` の閉じた集合、オーディエンス、クレームの発行のポリシーをまとめる。`application_id` が null のカタログの外の RP も許す。null でないときは同じテナントの `wsfed` の Application を指し、一つの Application に結び付く RP は一つまでである。")
model WsFedRelyingParty {
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("RP の識別子。WS-Fed の wtrealm で、SAML のオーディエンスでもある。WS-Federation は URI であることしか定めないので、上限は URI の分類に従う。btree のインデックスの行に値を収めるためのリソースの制限として、別に 2048 バイトの上限を適用する。")
  @minLength(1)
  @maxLength(2048)
  wtrealm: string;
  @doc("対応する Application。通常の CRUD はこれを変えず、関係を管理するのは Application の一括作成と一括削除だけである。")
  application_id?: string;
  @doc("管理 UI での表示名。")
  display_name?: string;
  @doc("許可する `wreply` の閉じた集合。この集合との完全一致だけを受け付ける。")
  reply_urls: string[];
  @doc("発行するアサーションのオーディエンス。空のときは wtrealm を使う。")
  audience?: string;
  @doc("発行するアサーションの SAML のバージョン。空のときは、Entra の WS-Fed が想定するデフォルトの SAML 1.1 を使う。")
  token_type?: WsFedTokenType;
  @doc("この RP に対するクレームの発行のポリシー。")
  claim_policy: ClaimMappingPolicy;
  @doc("Microsoft Entra のドメインフェデレーションのプリセット。フェデレーションを設定するときに使う必須のクレームと sourceAnchor の変換をまとめる。")
  entra_profile?: EntraFederationProfile;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("Microsoft Entra のドメインフェデレーションのプロファイル。IssuerUri、sourceAnchor の属性、パッシブ、アクティブ、MEX のエンドポイントをまとめ、UPN、ImmutableID、永続的な NameID のクレームのプリセットを WS-Fed の RP に適用する。")
model EntraFederationProfile {
  @doc("Entra で確認済みの DNS ドメイン。")
  @minLength(1)
  domain: string;
  @doc("Entra に登録した IssuerUri。発行するトークンの発行者と RP の wtrealm に一致する。")
  @minLength(1)
  issuer_uri: string;
  @doc("ImmutableID の元になる、変わらないユーザーの属性。objectGUID の GUID の文字列か、すでに base64 で符号化した値のどちらかである。")
  @minLength(1)
  source_anchor_attribute: string;
  @doc("変換した ImmutableID を保存する合成の属性名。デフォルトは `entra_immutable_id` である。")
  @minLength(1)
  immutable_id_attribute: string;
  @doc("Entra の PassiveLogOnUri。通常は `/wsfed` である。")
  passive_logon_uri: string;
  @doc("Entra の ActiveLogOnUri。通常は `/trust/usernamemixed` である。")
  active_logon_uri: string;
  @doc("Entra の MetadataExchangeUri。通常は `/trust/mex` である。")
  metadata_exchange_uri: string;
}


@doc("WS-Trust と WS-Federation のメタデータで示す STS のエンドポイント。受け付ける認証方式とバインディングを記述する。")
model SecurityTokenEndpoint {
  @doc("公開するエンドポイントの URL。")
  url: string;
  @doc("`usernamemixed`、`mex`、`passive` など。")
  @minLength(1)
  binding: string;
  @doc("受け付ける認証方式。たとえば `UsernameToken`。")
  auth_methods: string[];
}


@doc("WS-Trust 1.3 の Issue のバインディングの RST から取り出した、検査すべき内容を保持する中間の形。")
model WsTrustRequestSecurityToken {
  @doc("WS-Addressing の MessageID。リプレイの防止に使う。")
  @minLength(1)
  message_id: string;
  @doc("WS-Addressing の `To`。リクエストの宛先の STS のエンドポイントである。")
  to: string;
  @doc("WS-Addressing の `Action`。Issue だけを受け付ける。")
  action: string;
  @doc("WS-Security の UsernameToken のユーザー名。")
  @minLength(1)
  username: string;
  @doc("WS-Security の UsernameToken のパスワードの資格情報。")
  @minLength(1)
  password: string;
  @doc("WS-Security の Timestamp の `Created`。")
  created_at: utcDateTime;
  @doc("WS-Security の Timestamp の `Expires`。")
  expires_at: utcDateTime;
  @doc("RP、すなわちオーディエンス。登録済みの WsFedRelyingParty へ解決する。")
  @minLength(1)
  applies_to: string;
  @doc("求めたトークンの種類。空のときは RP のデフォルトを使う。")
  token_type?: WsFedTokenType;
  @doc("WS-Trust の RequestType。省略または Issue だけを受け付ける。")
  request_type?: string;
  @doc("WS-Trust の KeyType。省略または Bearer だけを受け付ける。")
  key_type?: string;
}


@doc("WS-Trust の Issue に対する RSTR。署名した SAML アサーションを、有効期間と appliesTo とともに含む。")
model WsTrustResponse {
  @doc("元の RST の MessageID。")
  relates_to?: string;
  @minLength(1)
  applies_to: string;
  token_type: WsFedTokenType;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  assertion_id?: string;
}


@doc("WS-Federation のパッシブのサインインで、署名したアサーションを RP へ発行した。")
model WsFedSignInIssued {
  occurredAt: utcDateTime;
  @doc("発行したテナント。")
  tenantId: string;
  @doc("発行先の RP の wtrealm。")
  wtrealm: string;
  @doc("サブジェクト。NameID が解決したユーザーである。")
  userId: string;
}


@doc("WS-Federation のパッシブのサインインを拒否した。未登録の wtrealm、許可する集合の外の `wreply`、検査の失敗などである。")
model WsFedSignInRejected {
  occurredAt: utcDateTime;
  @doc("拒否したテナント。")
  tenantId: string;
  @doc("求められた wtrealm。判別できたときだけ含む。")
  wtrealm?: string;
  @doc("拒否した理由の要約。")
  reason: string;
}


@doc("WS-Federation のサインアウトでローカルのセッションを破棄した。")
model WsFedSignOut {
  occurredAt: utcDateTime;
  @doc("サインアウトしたテナント。")
  tenantId: string;
  @doc("サインアウトを求めた RP の wtrealm。判別できたときだけ含む。")
  wtrealm?: string;
}


@doc("WS-Trust のアクティブ STS で、署名したアサーションを発行した。")
model WsTrustTokenIssued {
  occurredAt: utcDateTime;
  @doc("発行したテナント。")
  tenantId: string;
  @doc("発行先の RP、すなわちオーディエンス。")
  appliesTo: string;
  @doc("サブジェクトのユーザー。")
  userId: string;
}


@doc("WS-Trust のアクティブ STS への Issue のリクエストを拒否した。")
model WsTrustTokenRejected {
  occurredAt: utcDateTime;
  @doc("拒否したテナント。")
  tenantId: string;
  @doc("求められた AppliesTo。判別できたときだけ含む。")
  appliesTo?: string;
  @doc("拒否した理由の要約。")
  reason: string;
}


@doc("Microsoft Entra のドメインフェデレーションのプリセットを設定し、対応する WS-Fed の RP の信頼を作成または更新した。")
model EntraFederationConfigured {
  occurredAt: utcDateTime;
  @doc("設定したテナント。")
  tenantId: string;
  @doc("Entra で確認済みのドメイン。")
  domain: string;
  @doc("Entra に登録した IssuerUri。")
  issuerUri: string;
}


@doc("WS-Federation の Relying Party を登録または更新する JSON のリクエスト。呼び出し元が設定してよいものだけを含み、`application_id`、テナント、タイムスタンプはサーバーが管理する。")
model WsFedRelyingPartyRequest {
  @doc("RP の識別子。WS-Fed の wtrealm で、SAML のオーディエンスでもある。")
  wtrealm: string;
  @doc("管理 UI での表示名。任意で、サーバーは空の名前を拒否せずにそのまま保存する。")
  display_name?: string;
  @doc("許可する `wreply` の閉じた集合。この集合との完全一致だけを受け付ける。")
  reply_urls: url[];
  @doc("発行するアサーションのオーディエンス。空のときは wtrealm を使う。")
  audience?: string;
  @doc("この RP に対するクレームの発行のポリシー。")
  claim_policy: ClaimMappingPolicy;
}

}
