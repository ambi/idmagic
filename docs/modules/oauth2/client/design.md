# クライアントの設計

この文書は、[クライアント](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

### Client ID Metadata Documents（CIMD）による登録不要のクライアント解決

RFC 7591 の Dynamic Client Registration に加え、パスを持つ `https` URL 形式の `client_id` は `OAuth2Client` Repository ではなく、クライアントがホストする Client ID Metadata Document からその場で解決する。解決結果は永続化しない。Repository に `client_id` がない場合は文書を取得して 5 分間キャッシュし、他の処理経路と同じ `OAuth2Client` の形へ変換する。これにより、`redirect_uri` の照合、同意画面の描画、PKCE、スコープの処理に CIMD 固有の分岐は不要になる。統合点は `OAuth2ClientRepository` を埋め込み、`FindByID` だけを上書きする Decorator、`client/cimd_http.ClientRepositoryWithCIMD` である。Repository で見つかれば取得前に終了し、他のメソッド（`Save`、`Delete`、`FindAll`、資格情報の一覧）は変更せず委譲する。Composition Root (`cmd/internal/bootstrap`) で一度だけ接続するため、`authorize.go`、`push_authorization_request.go`、`client_auth.go` は変更不要である。

取得には、`tokens_jose.JWKResolver` が `jwks_uri` に使うものと同じ SSRF 対策済みダイヤラー `shared/security/safehttp` を使う。HTTPS のみ、DNS 解決後にパブリック IP だけを許可、検証済み IP への直接接続、環境プロキシの不使用、リダイレクト回数・タイムアウト・レスポンスボディサイズの上限を強制する。プロキシは検査済みの接続経路の外で最終送信先を解決して接続し、トランスポート層の SSRF 境界を迂回するため、直接接続が必要である。共通パッケージでは 2 つの取得処理を別々に実装せず、1 つの堅牢化した実装の背後に置く。MVP が受け入れるのは、`token_endpoint_auth_method` を省略するか `none` と宣言する文書だけであり、それ以外はフェイルクローズに拒否する。文書の `client_id` フィールドは取得元 URL と完全に一致しなければならない。解決したクライアントの `scope` は文書の自己宣言値（デフォルトは `openid`）とし、新しい管理者管理カタログではなく RFC 7591 DCR と同じ自己宣言型の信頼モデルを使う。CIMD で解決したクライアントは `Application` に関連付けない。自己登録した DCR クライアントと同じであり、`ApplicationGate` は Application レコードがない場合をフェイルクローズに拒否せず、許可として扱う。

## セキュリティ

### クライアント認証

`token_endpoint_auth_method` は `private_key_jwt`、`tls_client_auth`、`none`、`client_secret_post`、`client_secret_basic` の 5 種類に対応する。FAPI 級の非対称認証から従来の共有シークレット方式までを覆い、新しいクライアントを最も強い選択肢へ誘導しながら、既存環境の移行を継続できるようにする。`client_secret_jwt`（HMAC）は意図的に実装しない。`private_key_jwt` を利用できる場合、対称鍵による代替は能力を増やさずリスクだけを増やすためである。クライアント認証の失敗では `client_id` の登録有無を明かさず、常に `401 invalid_client` を返す。未登録の `client_id` にも同程度の検証コストを課し、タイミングオラクルを避ける。

`handlers_http` の `private_key_jwt` 検証は固定した規則の集合に従い、Discovery Metadata での広告とサーバーの実際の検査を一致させる。署名アルゴリズムは `PS256` と `ES256` だけを許可し（`none` と HMAC は不可）、`iss == sub == client_id`、このサーバーの発行者またはエンドポイント URL に一致する `audience`、クライアントが登録したインラインの `jwks` または `jwks_uri` から解決した署名鍵、上限付きのアサーション有効期間、`jti` の一度限りの利用を検査する。`jti` は DPoP と TTL および監査上の意味が異なるため、DPoP のリプレイストアとは別のストアに置く。同じリクエストで `client_assertion` と Basic 認証またはシークレット認証を組み合わせた場合は、RFC 6749 §2.3 に従い `invalid_request` として拒否する。
