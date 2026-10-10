namespace IdMagic.Contract {

@doc("JWT の署名アルゴリズム。FAPI 2.0 が認める強い非対称アルゴリズムに限り、HS256 のような対称アルゴリズムは除く。")
enum SignatureAlgorithm {
  PS256: "PS256",
  ES256: "ES256",
}


@doc("SigningKeyLifecycle の内部の状態。Active は現在の署名鍵、Verifying はローテーション後の重複期間に発行済みのトークンの検証だけに使う鍵、Retired は JWKS から外れた鍵、Archived は監査のために保管する鍵である。")
enum SigningKeyState {
  Active: "Active",
  Verifying: "Verifying",
  Retired: "Retired",
  Archived: "Archived",
}


@doc("暗号鍵の保管場所と、署名を行う主体。Local は秘密鍵をプロセスの中だけで保持する。Database は秘密鍵を平文のままアプリケーションのデータベースへ保存するので、そのデータベースのバックアップにも含まれる。VaultTransit は秘密鍵を Vault の外へ出さない。平文の暗号鍵を永続化するプロバイダーを選ぶのは明示的な選択であり、デフォルトにはしない。Database は特定の製品を指さない。")
enum KeyProvider {
  Local: "Local",
  Database: "Database",
  VaultTransit: "VaultTransit",
}


@doc("鍵の用途。Signing は OAuth 2.0 と OpenID Connect の JWT 署名を、XmlFederationSigning は SAML 2.0、WS-Federation、WS-Trust の XML 署名と、それらとともに公開する X.509 のメタデータを扱う。一つのテナントの中でも、用途ごとに有効な鍵は別である。")
enum KeyUsage {
  Signing: "Signing",
  XmlFederationSigning: "XmlFederationSigning",
}


@doc("トークンに署名する非対称鍵。テナント、用途、スコープごとに、Active の鍵一つと、0 個以上の Verifying の先行する鍵を公開する（SigningKeyMinJwksOverlap）。署名鍵の一覧、検索、ローテーション、選択は、いずれも `tenant_id`、`usage`、`scope_id` で絞り込む。`tenant_id` は既存の Tenant を指す。`scope_id` は通常 `default` で、追加の SAML IdP プロファイルではそのプロファイルの id とし、XML の署名の資格情報を分ける。")
model SigningKey {
  @doc("鍵が属するテナント。既存の Tenant を指し、テナント B の kid がテナント A の JWKS に現れることはない。")
  tenant_id: string = "default";
  @minLength(1)
  kid: string;
  alg: SignatureAlgorithm;
  @doc("暗号鍵の保管場所と、署名を行う主体。")
  provider: KeyProvider;
  @doc("鍵の用途。")
  usage: KeyUsage;
  @doc("一つのテナントと用途の中で独立した鍵のライフサイクル。通常は `default` で、追加の SAML IdP プロファイルではそのプロファイルの id である。")
  @minLength(1)
  scope_id: string = "default";
  public_jwk: JWK;
  @doc("XmlFederationSigning の鍵に対応する自己署名の X.509 証明書の DER。Signing の鍵では空で、XML の用途では必須である。公開情報であり、秘密鍵は含まない。")
  certificate_der?: bytes;
  state: SigningKeyState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  @doc("鍵が初めて Active になった時刻。")
  activated_at?: utcDateTime;
  @doc("鍵が Retired へ移った時刻。")
  retired_at?: utcDateTime;
  @doc("Verifying の鍵を JWKS に公開し続ける猶予の終わり。これ以降は JWKS から外れ、アーカイブの対象になる。")
  expires_at?: utcDateTime;
  archived_at?: utcDateTime;
}


@doc("JSON Web Key（RFC 7517）。kid で識別する一つの公開鍵。")
model JWK {
  kty: string;
  use?: string;
  kid: string;
  alg?: string;
  extra?: Record<unknown>;
}


@doc("JWK Set（RFC 7517 の 5 節）。/jwks エンドポイントが返す公開鍵の集合である。クライアントはこれを取得してトークンの署名を検証する。")
model JWKS {
  keys: JWK[];
}


@doc("テナントの管理者に見せる署名鍵のメタデータ。秘密鍵も鍵の保管方法も返さず、JWKS と同じ公開 JWK だけを返す。現在の署名鍵と、ローテーション後の重複期間に発行済みのトークンを検証するだけの鍵を見分けられる程度に詳しい。")
model AdminKeyResponse {
  @doc("JWK の鍵 ID。JWKS が公開するものと同じである。")
  kid: string;
  @doc("署名アルゴリズム。現在は PS256 だけである。")
  alg: SignatureAlgorithm;
  @doc("新しい ID トークンとアクセストークンに署名する鍵では true。ローテーション後の重複期間に発行済みのトークンを検証するためだけに JWKS に残る鍵では false。")
  active: boolean;
  @doc("鍵を生成した時刻。SigningKeyMaxAge と SigningKeyMinJwksOverlap はここから測る。")
  created_at: utcDateTime;
  @doc("公開 JWK。JWKS と同じ表現である。")
  public_jwk: JWK;
}


@doc("すべての署名鍵。ローテーションは同じ手順で古い鍵を無効にするので、二つの鍵が同時に有効になることはない。")
model AdminKeyListResponse {
  keys: AdminKeyResponse[];
}


@doc("ローテーションでテナントのどの鍵を置き換えるか。ボディは任意で、ボディがないとき、または `usage` がないときは Signing の鍵をローテーションする。XmlFederationSigning は、WS-Federation とデフォルトの SAML IdP プロファイルが署名に使う、デフォルトのスコープの XML の鍵をローテーションする。それまでの証明書は SigningKeyMinJwksOverlap の間、SAML と WS-Federation のメタデータに残る。")
model AdminRotateKeyRequest {
  @doc("ローテーションする鍵の用途。デフォルトは Signing である。")
  usage?: KeyUsage;
}


@doc("鍵のローテーションの結果。新しく有効になった鍵と、verifying へ退いた鍵の kid からなる。")
model AdminRotateKeyResponse {
  @doc("新しく有効になった署名鍵。")
  next: AdminKeyResponse;
  @doc("それまで有効だった鍵。最初のローテーションでは省く。")
  previous?: AdminKeyResponse;
}


@doc("一つのテナントの署名鍵の健全性。system_admin がテナントを横断して健全性を確認できるように、プロバイダー、有効な kid、鍵の数、プロバイダーへの到達性を返す。秘密鍵は含まない。")
model TenantSigningKey {
  @doc("この行が対象とするテナント。")
  tenant_id: string;
  @doc("このテナントが使う KeyProvider。")
  provider: KeyProvider;
  @doc("鍵の用途。現在は Signing である。")
  usage: KeyUsage;
  @doc("現在有効な署名鍵の kid。まだ鍵のないテナントでは省く。")
  active_kid?: string;
  @doc("有効な鍵の状態。")
  active_state?: SigningKeyState;
  @doc("JWKS が含む鍵の数（active と verifying の合計）。")
  jwks_key_count: int64;
  @doc("KeyProvider に到達できるかどうか。false のときは新たな発行をフェイルクローズで止め、その強制点は OAuth2.Token である。")
  provider_healthy: boolean;
}


@doc("JWKS の署名鍵をローテーションした。デフォルトでは 90 日ごと、または緊急のローテーションとして行う。")
model SigningKeyRotated {
  occurredAt: utcDateTime;
  @doc("ローテーションした署名鍵が属するテナント。")
  tenantId: string;
  newKid: string;
  previousKid?: string;
}


@doc("SigningKeyMinJwksOverlap を過ぎたか、緊急に無効にしたため、署名鍵が Verifying から Retired へ移り、JWKS から外れた。")
model SigningKeyRetired {
  occurredAt: utcDateTime;
  tenantId: string;
  kid: string;
}


@doc("猶予が終わった署名鍵が Retired から Archived へ移った。Archived は終端の状態で、監査のために保管する。")
model SigningKeyArchived {
  occurredAt: utcDateTime;
  tenantId: string;
  kid: string;
  retiredAt: utcDateTime;
  expiresAt: utcDateTime;
  disposedAt: utcDateTime;
}


@doc("`kid` が指す署名鍵がこのテナントに存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:key_not_found` である。")
model SigningKeyNotFoundError is ProblemDetails;


@doc("署名鍵のストアが設定されていないので、鍵を読むこともローテーションすることもできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:key_store_unavailable` である。")
model KeyStoreUnavailableError is ProblemDetails;

}
