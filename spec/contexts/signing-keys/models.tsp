namespace IdMagic.Contract {

@doc("JWT 署名アルゴリズム。FAPI 2.0 が許容する asymmetric な強度の高いものに限定（HS256 等の対称鍵は除外）。")
enum SignatureAlgorithm {
  PS256: "PS256",
  ES256: "ES256",
}


@doc("SigningKeyLifecycle の内部状態。Active は現在の署名鍵、Verifying は回転後の移行期間に既発行トークンだけを検証する鍵、Retired は JWKS から除去済、Archived は監査保管。")
enum SigningKeyState {
  Active: "Active",
  Verifying: "Verifying",
  Retired: "Retired",
  Archived: "Archived",
}


@doc("鍵マテリアルの保管種別と署名の実行主体。Local / Database は private key をプロセス内に保持する dev/test 用、VaultTransit は private key を Vault へ委譲する本番用。Database は特定の製品名を表さない。")
enum KeyProvider {
  Local: "Local",
  Database: "Database",
  VaultTransit: "VaultTransit",
}


@doc("鍵の用途。Signing は OAuth2/OIDC の JWT 署名、XmlFederationSigning は SAML 2.0 / WS-Fed / WS-Trust の XML 署名と X.509 metadata 公開に用いる。同じ tenant でも用途ごとに active 鍵を分離する。")
enum KeyUsage {
  Signing: "Signing",
  XmlFederationSigning: "XmlFederationSigning",
}


@doc("トークン署名用の非対称鍵。テナント・用途・scopeごとに Active 1本＋Verifying 過去鍵 0本以上を公開する（SigningKeyMinJwksOverlap）。列挙・検索・回転・署名鍵選択はすべて tenant_id / usage / scope_id で絞る。tenant_id は既存 Tenant を指す。scope_id は既定で default とし、SAML追加IdP profileではprofile IDを用いてXML署名資格情報を分離する。")
model SigningKey {
  @doc("鍵が属するテナント ID。既存 Tenant を指し、テナント A の JWKS にテナント B の kid は出ない。")
  tenant_id: string = "default";
  @minLength(1)
  kid: string;
  alg: SignatureAlgorithm;
  @doc("鍵マテリアルの保管場所と署名の実行主体。")
  provider: KeyProvider;
  @doc("鍵の用途。")
  usage: KeyUsage;
  @doc("同一tenant・usage内の独立した鍵lifecycle。通常はdefault、追加SAML IdP profileではprofile ID。")
  @minLength(1)
  scope_id: string = "default";
  public_jwk: JWK;
  @doc("XmlFederationSigning 鍵に対応する自己署名 X.509 certificate の DER。Signing 鍵では空、XML用途では必須。公開情報であり秘密鍵を含まない。")
  certificate_der?: bytes;
  state: SigningKeyState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  @doc("一度 Active となった時刻。")
  activated_at?: utcDateTime;
  @doc("Retired 状態に遷移した時刻。")
  retired_at?: utcDateTime;
  @doc("Rotate 後に Verifying 鍵を JWKS へ公開し続ける grace の終了時刻。この時刻以降は JWKS から除外され archive 対象になる。")
  expires_at?: utcDateTime;
  archived_at?: utcDateTime;
}


@doc("JSON Web Key (RFC 7517)。kid で識別される 1 件の公開鍵情報。")
model JWK {
  kty: string;
  use?: string;
  kid: string;
  alg?: string;
  extra?: unknown;
}


@doc("JWK Set (RFC 7517 §5)。/jwks エンドポイントが返す公開鍵集合。クライアントがトークン署名を検証する際に取得する。")
model JWKS {
  keys: JWK[];
}


@doc("テナント管理者向けに公開する署名鍵のメタデータ。秘密鍵と鍵保管実装は返さず、\nJWKS と同じ公開鍵 JWK のみを含める。現在の署名鍵か、回転後の移行期間に\n既発行トークンを検証する鍵かを判断できる粒度を提供する。")
model AdminKeyResponse {
  @doc("JWK Key ID。JWKS で配布されるものと同一。")
  kid: string;
  @doc("署名アルゴリズム (現状 PS256 のみ)。")
  alg: SignatureAlgorithm;
  @doc("true は現在新規 ID Token / Access Token を署名する鍵。false は回転後の移行期間に既発行トークンを検証するためだけに JWKS へ残る鍵。")
  active: boolean;
  @doc("鍵生成時刻 (SigningKeyMaxAge / SigningKeyMinJwksOverlap 評価の起点)。")
  created_at: utcDateTime;
  @doc("公開鍵 JWK。JWKS と同じ表現。")
  public_jwk: unknown;
}


@doc("全署名鍵の一覧。active が複数同時に存在することは無い (rotateInternal で旧鍵は非 active 化)。")
model AdminKeyListResponse {
  keys: AdminKeyResponse[];
}


@doc("鍵ローテーションの結果。新しい active 鍵と直前の active 鍵 (verifying へ降りた) の kid を返す。")
model AdminRotateKeyResponse {
  @doc("新しい active 署名鍵。")
  next: AdminKeyResponse;
  @doc("直前まで active だった鍵 (初回ローテーション時は無し)。")
  previous?: AdminKeyResponse;
}


@doc("テナント 1 件分の署名鍵ヘルス。system_admin がテナント横断で鍵の\n健全性 (provider / active kid / 鍵数 / provider 到達性) を一覧するために返す。\n秘密鍵は含めない。")
model TenantSigningKey {
  @doc("対象テナント ID。")
  tenant_id: string;
  @doc("このテナントが使う KeyProvider。")
  provider: KeyProvider;
  @doc("鍵の用途。現状 Signing。")
  usage: KeyUsage;
  @doc("現在 active な署名鍵の kid。鍵がまだ無いテナントでは無し。")
  active_kid?: string;
  @doc("active 鍵の状態。")
  active_state?: SigningKeyState;
  @doc("JWKS に載る鍵数 (active + verifying)。")
  jwks_key_count: int64;
  @doc("KeyProvider が到達可能か。false のとき FailClosed で新規発行を止める (強制点は OAuth2.Token)。")
  provider_healthy: boolean;
}


@doc("JWKS の署名鍵が回転した（既定 90 日ごと、または緊急回転）。")
model SigningKeyRotated {
  occurredAt: utcDateTime;
  @doc("回転した署名鍵が属するテナント ID。")
  tenantId: string;
  newKid: string;
  previousKid?: string;
}


@doc("署名鍵が Verifying から Retired に移り JWKS から除去された\n(SigningKeyMinJwksOverlap 経過、または緊急無効化)。")
model SigningKeyRetired {
  occurredAt: utcDateTime;
  tenantId: string;
  kid: string;
}


@doc("署名鍵が grace 終了後に Retired から Archived (終端、監査保管) に移った。")
model SigningKeyArchived {
  occurredAt: utcDateTime;
  tenantId: string;
  kid: string;
  retiredAt: utcDateTime;
  expiresAt: utcDateTime;
  disposedAt: utcDateTime;
}

}
