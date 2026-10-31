namespace IdMagic.Contract {

@doc("JWT の署名アルゴリズム。FAPI 2.0 が認める強い非対称アルゴリズムに限り、HS256 のような対称アルゴリズムは除く。")
enum SignatureAlgorithm {
  PS256: "PS256",
  ES256: "ES256",
}


@doc("SigningKeyLifecycle の内部の状態。Prepared は署名開始前に公開する XML 鍵、Active は現在の署名鍵、Verifying は切り替え後の重複期間に検証だけに使う鍵、Retired は公開を終えた鍵、Archived は監査のために保管する鍵である。")
enum SigningKeyState {
  Prepared: "Prepared",
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
  IdJagSigning: "IdJagSigning",
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


@doc("テナントの管理者に見せる署名鍵の公開メタデータ。用途、スコープ、有効と準備済みの状態、公開 JWK、XML 署名証明書を返し、秘密鍵の素材を返さない。")
model AdminKeyResponse {
  usage: KeyUsage;
  scope_id: string;
  prepared: boolean;
  @doc("XML 署名証明書の公開部分の base64。JWT 用の鍵では省略する。")
  signing_certificate?: string;
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


@doc("操作と対象の鍵集合を指定する。ボディと usage を省略すると Signing の鍵を即時に切り替える。XmlFederationSigning の default スコープは WS-* と既定 SAML が共有し、事前公開と登録確認後の切り替えの二段階を必須とする。SAML 専用スコープでは二段階を任意に選べる。")
model AdminRotateKeyRequest {
  @doc("rotate は即時切り替え、prepare は XML 鍵の事前公開、activate は確認付きの切り替え。省略時は共有 XML 鍵では prepare、ほかでは rotate。")
  action?: "rotate" | "prepare" | "activate";
  @doc("activate で切り替える準備済みの kid。")
  kid?: string;
  @doc("この鍵を使う全連携先の信頼を更新済みと管理者が確認したこと。activate では true を要する。")
  trust_updated?: boolean;
  @doc("対象のスコープ。省略時は default。")
  scope_id?: string;
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


@doc("XML 鍵を事前公開した。署名に使う有効な鍵は変わっていない。")
model SigningKeyPrepared {
  occurredAt: utcDateTime;
  tenantId: string;
  kid: string;
}

@doc("署名鍵を切り替えた。XML の共有鍵では連携先への登録を確認してから行う。")
model SigningKeyRotated {
  occurredAt: utcDateTime;
  @doc("ローテーションした署名鍵が属するテナント。")
  tenantId: string;
  newKid: string;
  previousKid?: string;
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
