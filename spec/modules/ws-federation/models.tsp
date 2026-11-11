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
  @doc("Microsoft Entra のドメインフェデレーションのプリセット。フェデレーションを設定するときに使う必須のクレームと sourceAnchor の属性をまとめる。")
  entra_profile?: EntraFederationProfile;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("Microsoft Entra のドメインフェデレーションのプロファイル。テナントに一つの `urn:federation:MicrosoftOnline` の RP に、構成したドメインと IssuerUri の組、sourceAnchor の属性、パッシブ、アクティブ、MEX のエンドポイントをまとめ、UPN、ImmutableID、永続的な NameID のクレームのプリセットを適用する。")
model EntraFederationProfile {
  @doc("構成したドメインと IssuerUri の組。発行するトークンの発行者は、利用者の UPN のドメインに最も長く一致する組の IssuerUri である。")
  @minItems(1)
  domains: EntraDomainFederation[];
  @doc("ImmutableID と同じ文字列を保存するユーザーの属性。属性スキーマで `unique` で、`editable_by_user` でない属性に限る。objectGUID を使う場合は、取り込みの側で base64 に符号化した値を保存する。Directory Connector は AD の値を組み込みの属性 `on_premises_immutable_id` に保存する。")
  @minLength(1)
  source_anchor_attribute: string;
  @doc("sourceAnchor の値を変えずに ImmutableID として保存する合成の属性名。デフォルトは `entra_immutable_id` である。")
  @minLength(1)
  immutable_id_attribute: string;
  @doc("Entra の PassiveLogOnUri。通常は `/wsfed` である。")
  passive_logon_uri: string;
  @doc("Entra の ActiveLogOnUri。通常は `/trust/usernamemixed` である。")
  active_logon_uri: string;
  @doc("Entra の MetadataExchangeUri。通常は `/trust/mex` である。")
  metadata_exchange_uri: string;
}

@doc("Entra のドメインフェデレーションで構成した一つのドメイン。")
model EntraDomainFederation {
  @doc("Entra で確認済みの DNS ドメイン。小文字で保存する。")
  @minLength(1)
  domain: string;
  @doc("このドメインに Entra で設定した IssuerUri。ドメインの間で重ならない。")
  @minLength(1)
  issuer_uri: string;
  @doc("このドメインを持つ Entra のテナントの ID。GUID を小文字で保存する。持つときは、Graph への接続をこのテナントに限り、観測値のテナントと比べる。持たないときは観測したテナントを記録するだけで比べない。")
  @pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
  entra_tenant_id?: string;
}

@doc("Graph PowerShell の読み戻しの手順が書き出す、一つのドメインの Entra の登録値の観測文書。管理者が手で取り込む。未知の項目は受け付けない。")
model EntraFederationObservation {
  @doc("文書の形式。")
  format: "idmagic.entra-federation-observation.v1";
  @doc("観測した接続先の Entra のテナント ID（`Get-MgContext` の `TenantId`）。")
  @minLength(1)
  entra_tenant_id: string;
  @doc("観測したドメイン。取り込み先のドメインと大文字と小文字を区別せずに一致しなければならない。")
  @minLength(1)
  domain: string;
  @doc("ドメインの認証方式（`Get-MgDomain` の `AuthenticationType`）。フェデレーションしたドメインは `Federated` である。")
  @minLength(1)
  authentication_type: string;
  @doc("Graph から取得した時刻。取り込みの時刻より 5 分を超えて後の値は受け付けない。")
  retrieved_at: utcDateTime;
  @doc("ドメインのフェデレーションの構成（`Get-MgDomainFederationConfiguration`）。構成がなければ空である。")
  @maxItems(4)
  federation_configurations: EntraObservedFederationConfiguration[];
}

@doc("Graph から取得した一つのフェデレーションの構成の値。Graph が値を持たない項目は空文字列である。")
model EntraObservedFederationConfiguration {
  issuer_uri: string;
  passive_sign_in_uri: string;
  active_sign_in_uri: string;
  metadata_exchange_uri: string;
  sign_out_uri: string;
  @doc("署名証明書の公開部分の base64。空白は比べるときに除く。")
  signing_certificate: string;
  @doc("次の署名証明書の公開部分の base64。なければ空である。")
  next_signing_certificate: string;
  preferred_authentication_protocol: string;
  federated_idp_mfa_behavior: string;
}

@doc("Entra への反映の確認の状態。`unverified` は観測記録がない、`matched` と `mismatched` は現在の期待値と比べた結果、`stale` は観測の後に構成か署名鍵が変わり比べていないことを表す。")
enum EntraVerificationStatus {
  unverified,
  matched,
  mismatched,
  stale,
}

@doc("期待値と観測値の異なる一つの項目。証明書は拇印ではなく比べた値の要約を返す。")
model EntraSettingDifference {
  @doc("比べた項目。`entra_tenant_id`、`authentication_type`、`federation_configurations`、`issuer_uri`、`passive_sign_in_uri`、`active_sign_in_uri`、`metadata_exchange_uri`、`sign_out_uri`、`preferred_authentication_protocol`、`federated_idp_mfa_behavior`、`signing_certificate`、`next_signing_certificate` のどれかである。")
  field: string;
  @doc("期待値。証明書は SHA-256 の拇印、構成の件数は数字の文字列である。")
  expected: string;
  @doc("観測値。証明書は読めれば SHA-256 の拇印、読めなければ `unreadable`、空なら空文字列である。")
  observed: string;
}

@doc("一つのドメインの Entra への反映の確認。観測記録は手動で取り込んだ時点の事実であり、継続的な監視の結果ではない。")
model EntraFederationVerification {
  status: EntraVerificationStatus;
  @doc("観測記録の Graph から取得した時刻。記録がなければ省く。")
  retrieved_at?: utcDateTime;
  @doc("観測記録を取り込んで比べた時刻。記録がなければ省く。")
  checked_at?: utcDateTime;
  @doc("観測した Entra のテナント ID。記録がなければ省く。")
  observed_entra_tenant_id?: string;
  @doc("比べた有効な署名証明書の SHA-256 の拇印。記録がなければ省く。")
  signing_certificate_thumbprint?: string;
  @doc("比べた次の署名証明書の SHA-256 の拇印。記録がないか、次の証明書がなければ省く。")
  next_signing_certificate_thumbprint?: string;
  @doc("`matched` と `mismatched` のときの差分。`matched` では空である。`unverified` と `stale` では省く。")
  differences?: EntraSettingDifference[];
}

@doc("署名証明書の要約。")
model EntraCertificateSummary {
  @doc("DER の SHA-256 の 16 進の小文字。")
  sha256_thumbprint: string;
  not_after: utcDateTime;
}

@doc("取り込もうとした観測文書の取得時刻が、保存済みの観測記録の取得時刻より前である。新しい観測を古い観測で上書きしない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:observation_outdated` である。")
model EntraObservationOutdatedError is ProblemDetails;

@doc("指定したドメインを、このテナントの Entra のドメインフェデレーションで構成していない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:not_found` である。")
model EntraFederationDomainNotFoundError is ProblemDetails;


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

@doc("sourceAnchor に指定した属性が、属性スキーマで `unique` でない。値が重なると、二人の利用者に同じ ImmutableID を発行する。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:source_anchor_not_unique` である。")
model SourceAnchorNotUniqueError is ProblemDetails;

@doc("sourceAnchor に指定した属性が、属性スキーマで `editable_by_user` である。利用者が値を他人の値に書き換えられる。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:source_anchor_editable` である。")
model SourceAnchorEditableError is ProblemDetails;

@doc("UPN がフェデレーションするドメインに属する利用者の sourceAnchor の値に問題がある。`type` は、値がないなら `urn:idmagic:error:source_anchor_missing`、空か `@` を含むなら `urn:idmagic:error:source_anchor_invalid` である。")
model SourceAnchorValueError {
  ...ProblemDetails;
  @doc("問題のある値を持つ利用者の数。")
  user_count: int32;
  @doc("問題のある値を持つ利用者のうち、ユーザー名の順で先頭の 5 人までのユーザー名。")
  @maxItems(5)
  sample_usernames: string[];
}

@doc("登録しようとした RP の `wtrealm` は、Entra のドメインフェデレーションの定型設定を持つ RP のものである。定型設定と固定のクレームは、Entra の構成の操作だけが変える。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:entra_relying_party` である。")
model EntraRelyingPartyError is ProblemDetails;

}
