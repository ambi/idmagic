# WsFederation Internals

## WsFederationSignOut
`WsFederationSignIn` が所有する単一のパッシブ HTTP エンドポイントにおけるサインアウトの意味を定める。ローカルセッションを破棄し、`wsignout1.0` では許可済みの `wreply` へのリダイレクトまで行い、`wsignoutcleanup1.0` では破棄だけを行って 200 を返す。
- Input invariant: input.wa == "wsignout1.0" || input.wa == "wsignoutcleanup1.0"
- Input invariant: input.wtrealm == null || wtrealm_registered(input.wtrealm, context.tenant_id)
- Input invariant: input.wreply == null || reply_url_allowed(input.wtrealm, input.wreply, context.tenant_id)
- Result invariant: local_session_cleared
- Result invariant: no_unregistered_redirect

## Tenant signing

受動的な発行でも能動的な発行でも、発行時にリクエスト元テナントの有効な `XmlFederationSigning` 資格情報を取得する。フェデレーションメタデータでは、広告する役割ごとに現在有効な証明書と有効期限内の検証用証明書を公開する。これにより、RP は計画されたローテーションの前後でも検証を継続できる。

署名は、プロセス起動時に保持した状態ではなく、`SigningKeys` から取得した資格情報で行う。これにより、WS-Fed と SAML は XML 署名資格情報のライフサイクルを共有しながら、OAuth2 と JWT の鍵から分離できる。

## Federation metadata

この Context は、各レルムの `/{realm}/federationmetadata/2007-06/federationmetadata.xml` で AD FS 互換の `federationmetadata.xml` を公開し、テナントの発行者（デフォルトテナントでは `/realms/default`）を entityID として広告する。これにより、WS-Fed RP と Microsoft Entra のドメインフェデレーションは、別の導入手順を使わずに発行者、エンドポイント、署名証明書を検出できる。

`EntityDescriptor` は `SecurityTokenServiceType` と `ApplicationServiceType` の両方の `RoleDescriptor` を持ち、`PassiveRequestorEndpoint`、`SecurityTokenServiceEndpoint`、`MetadataEndpoint`、そして署名の `KeyDescriptor` を広告する。署名の証明書は OAuth と OIDC の JWK の形を再利用せず、WS-* が本来使う X.509 の形で公開する。鍵の用途、ローテーション、重なりは `SigningKeys` の責務のままであり、メタデータは WS-* の利用者が既に期待するものを広告すれば足りるからである。`/{realm}/trust/mex` は能動的な STS (下記) の探索として、`usernamemixed` のエンドポイントと UsernameToken を必須とするその方針を公開する。RST と RSTR のやり取り自体はメタデータの一部ではない。

クレームの公開は宣言的に定義する。AD FS のクレーム規則言語は採用せず、`ClaimMappingPolicy` を WS-Fed、WS-Trust、SAML で共有する。対応付けたクレームの集合には AD FS 規則言語ほどの表現力は不要であり、検証コストだけが増えるためである。対応付けのない属性は決して発行しない。

## WS-Trust active STS scope

能動的な WS-Trust への対応は、汎用的な相互運用ではなく、Microsoft 365 型のリッチクライアントによるサインインを対象とする。SOAP、WS-Security、WS-Addressing、SAML の署名は相互に関係するため、対応するバインディングを増やすほど再送や XML 署名ラッピングの攻撃面が広がる。そこで、初期の対応範囲は意図的に狭くする。

能動的な STS のエンドポイントは `/trust/usernamemixed` だけであり、受け付けるのは WS-Trust 1.3 の `Issue` のみである。`Validate`、`Renew`、`Cancel` は実装しない。認証には UsernameToken だけを使い、既存の `UserRepository`、`PasswordHasher`、`LoginAttemptThrottle` で検証する。Kerberos と IWA の `windowstransport` は範囲外とし、将来の実装に委ねる。

WS-Addressing と WS-Security の必須要素（`MessageID`、`To`、`Action`、UsernameToken、Timestamp、`AppliesTo`）はフェイルクローズで検証する。Timestamp は期限切れの値と遠い未来の値を拒否し、`MessageID` は有効期間の短いリプレイ防止ストアに記録する。`AppliesTo` は登録済みの WS-Fed RP に解決できなければならず、未登録の宛先は拒否する。発行する Assertion の audience と recipient はその RP に限定し、クレームは RP の `ClaimMappingPolicy` を通じて発行する。これにより、リプレイや audience の取り違えが RP の境界を越えることを防ぐ。RSTR は SOAP 1.2 で署名済み SAML Assertion を返し、RST が SAML 1.1 または SAML 2.0 を明示的に要求しない場合は SAML 1.1 をデフォルトとする。

## Entra domain federation profile

`EntraFederationProfile` は、WS-Federation RP 用の定型設定である。ドメイン、IssuerUri、sourceAnchor 属性、受動・能動・MEX の各エンドポイントを受け取り、wtrealm と audience に同じ IssuerUri を持つ `WsFedRelyingParty` を作成または更新する。定型設定にすることで、クレーム設定の JSON を手書きする必要がなくなる。手書きの設定を誤ると Entra 側では原因を特定しにくく、設定時に sourceAnchor の安定性や一意性も保証できない。

必須クレームは定型設定で固定し、フェイルクローズで扱う。UPN は `preferred_username` から `http://schemas.xmlsoap.org/claims/UPN` として発行する。ImmutableID は正規化した sourceAnchor（`entra_immutable_id`）から導き、永続的な NameID と `http://schemas.xmlsoap.org/claims/nameidentifier` の両方に含める。sourceAnchor は、プロファイル設定時には既存ユーザーの欠落、重複、変換できない値を拒否し、発行時には対象ユーザーの ImmutableID を導けない場合にクレーム発行を拒否する。

GUID の形をした sourceAnchor の値は、ImmutableID として使う前に .NET の `Guid.ToByteArray()` のバイト順 — AD FS と Entra の慣行 — で base64 に符号化する。既に base64 の値はそのまま通す。このバイト順を誤ると、Entra は assertion を元の社内の同じユーザーへ関連付けられず、アカウントの重複やサインインの失敗を招く。プロファイルのデフォルトのトークンの型は SAML 1.1 であり、Entra と AD FS の WS-Fed のデフォルトに合わせている。Hybrid Azure AD Join の端末の登録 (`windowstransport` とコンピューターアカウントの Kerberos) は明確に範囲外であり、設定の案内では managed や PHS、あるいは AD FS の併存のデプロイへ誘導する。
