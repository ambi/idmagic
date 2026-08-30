# API Rules

## Cursor pagination

管理用の一覧 API は、署名済みで版の付いたキーセット方式のカーソルを RFC 8288 の `Link` レスポンスヘッダーで返す。カーソルは自身のテナント、問い合わせと並び順の同一性、方向、行の境界を束縛する。

## HTTP error responses

汎用 API のエラーレスポンスには、既定形式として RFC 9457 Problem Details（`application/problem+json`、`type`、`title`、`status`、`detail`、`instance`）を使う。`instance` には上記のリクエスト相関用の `request_id` を載せる。HTTP ステータスコードは RFC 9110 に従い、400 はリクエストを解析できないこと（不正な JSON、必須構造の欠落）を、422 は解析できた内容が業務規則に違反すること（不正なロール、参照の不一致、ポリシー違反）を表す。

OAuth2（`backend/oauth2/handlers_http`）、SCIM（`backend/sourcing/scim/handlers_http`）、Dynamic Client Registration（RFC 7591、`backend/oauth2/handlers_http` の一部）、SharedSignals の受信エンドポイント（RFC 8935、`/ssf/streams/{stream_id}/events`）は、各標準が定めるエラーレスポンスを返す。標準に従うクライアントとの相互運用性を保つため、これらには Problem Details を適用しない。この境界は接点ごとに引く。同じパッケージの中でも、ブラウザーや管理コンソールが呼ぶ汎用 API は Problem Details を返し、標準が形を定める相手だけが例外である。

契約側では、この 3 通りの本文をそれぞれ 1 つのモデルが持つ。汎用 API のエラーは `IdMagic.Contract.ProblemDetails` で、`type` / `title` / `status` / `detail` / `instance` を宣言する。個々のエラーは `model <Name>Error is ProblemDetails;` と書き、どの `type` URN 接尾辞 (= サーバーが返す error code) に対応するかを `@doc` で名指しする。標準が形を定める接点のうち OAuth 2.0 / OIDC 系は `IdMagic.Contract.OAuthError` (`error` / `error_description`、RFC 6749 §5.2) を、SCIM は `IdMagic.Contract.ScimProtocolError` (RFC 7644 §3.12) を同じ形で参照する。どちらにも当てはまらない独自形状は、`EndpointRateLimitPolicy` の 429 (`error` / `retry_after_seconds` / `message` と `Retry-After` ヘッダー) と SharedSignals 受信エンドポイントの拒否 (`error` / `message`) の 2 つだけで、それぞれ本文を直接宣言する。同じ error code が接点によって別の本文で返るときは、`AccessDeniedError` と `OAuthAccessDeniedError` のように接点ごとにモデルを分ける。1 つのモデルが 2 つの形を名乗ることは許さない。

エラー本文の文字列は、`DisplayLanguage` にかかわらず常に英語で固定する。`message`、`error_description`、`detail`、プレーンテキストの本文がこれにあたる。翻訳して返すと、同じ失敗が呼び出し側の設定次第で別の文字列になり、ログの照合も相互運用も壊れるためである。翻訳は、安定したエラーコードを鍵として UI 側が行う。Problem Details では `type` の `urn:idmagic:error:` に続く部分がその鍵であり、UI は辞書に無いコードに出会ったとき、`detail` または `title` を英語のまま表示する。

## Declared status codes

operation は、自身のハンドラーと、その手前に立つ guard が書くステータスコードをすべて宣言する。[Wire bodies in the contract](#wire-bodies-in-the-contract) が本文について定める規則 —— 契約に書くのはサーバーが実際に返すものである —— を、ステータス行にも同じように及ぼす。宣言に無いコードが返れば呼び出し側は分岐を持てず、宣言にあって返らないコードは呼び出し側に到達しない分岐を書かせる。

例外は 2 つある。どちらも「その operation の応答ではない」という同じ理由による。

1 つは、共通のエラーハンドラーが、どのハンドラーも写像しなかったエラーに対して最後に書く 500 である。これはすべての operation で同じに出るうえ、呼び出し側が operation ごとに変えられる対応も無い。333 の operation に同じ 1 行を書いても、呼び出し側の分岐は 1 つも増えない。逆に、ハンドラーが固有のエラーコードを添えて自分で書く 5xx —— パスキーの依存先が使えないときの 503 `webauthn_unavailable` のような —— はその operation 固有の結果なので宣言する。

もう 1 つは、テナント解決ミドルウェアが返す 404 `{"error": "tenant_not_found"}` である。これは routing の手前で返るので、どの operation の応答でもない。operation ごとに宣言すれば、その operation 自身が持つ 404 (資源が無い) と同じ行に 2 つの意味が乗り、呼び出し側はどちらなのかを本文から読み直すことになる。ミドルウェアが返すこの 1 つだけは、契約ではなくここに書く。宿主名から解決できないテナントへの要求は、経路にかかわらず 404 と `tenant_not_found` で返る。

401 と 403 は、同じ guard の 2 つの分岐である。認証済みのセッションが無ければ 401 `authentication_required`、あっても権限が足りなければ 403 `access_denied` になる。したがって 403 を宣言する operation は 401 も宣言する。片方だけを宣言することは、呼び出し側に「サインインしていない」を「権限が無い」として扱わせることであり、再認証すれば通る要求を通らないものとして扱わせる。

401 の本文は operation ごとに変わらないので、共有の応答モデルを 1 つ置いて参照する。[HTTP error responses](#http-error-responses) が定める 3 系統のエラー本文に対応して、汎用 API は `IdMagic.Contract.AuthenticationRequiredResponse` (Problem Details)、OAuth 2.0 / OIDC は `IdMagic.Contract.OAuthUnauthorizedResponse`、SCIM は `IdMagic.Contract.ScimUnauthorizedResponse` を参照する。403 が operation ごとのモデルを持つのは、その本文が operation ごとに違う (`AccessDeniedError` / `InsufficientScopeError` / `MfaEnrollmentNotAllowedError`) からであって、様式の統一のためではない。

この節が定める一致は `mise run check-status-drift` が検査する。検査は operation ごとに、契約が宣言する集合と、ハンドラーおよび guard が書くコードを突き合わせる。エラー値から応答を決めるヘルパー (`WriteAccountError` のような写像) は辿らない。どの分岐に入るかはユースケースが返すエラーで決まり、ハンドラーの字面には現れないためである。辿れなかった operation は「合格」ではなく「読み残しあり」として数え、その件数を毎回の出力に書く。

## Interface stability and versioning

管理 API とセルフサービスのアカウント API は外部契約である。外部インターフェースは TypeSpec の契約で 3 つに分類する。

| Stability | 意味 |
| --- | --- |
| `stable` | バージョン付きの外部契約。下記の互換性保証の対象。 |
| `beta` | まだ互換性保証の対象でない外部契約。 |
| `internal` | 外部契約に含まれない。ファーストパーティーのブラウザーセッションからだけ到達でき、API アクセストークンでは到達できない。予告なく変更する。 |

後方互換な変更はフィールドの追加、任意パラメーターの追加、新しいエンドポイントの追加である。破壊的変更はフィールドの削除または名前変更、フィールド型の変更、フィールドの必須化、エラーコードの変更、デフォルト値の変更である。

`stable` と `beta` はパスでバージョンを表す (`/api/admin/v1/...`、`/api/account/v1/...`)。バージョンなしの形は設けない。破壊的変更は既存パスを変更せず新しい接頭辞で導入し、同時に提供するバージョンは最大 2 つとする。

この方式の対象外は、OAuth 2.0 / OIDC、SAML、WS-Federation、SCIM、SharedSignals のプロトコルエンドポイントである。これらの互換性は各標準が規定し、Discovery Metadata (`/.well-known/...`)、`/scim/v2/ServiceProviderConfig`、SAML / WS-Fed メタデータが正である。IdMagic 側のパスの版を重ねると、標準が定めた探索経路と食い違う。

## Deprecation

非推奨の予定は TypeSpec に記録し、散文の側に一覧を持たない。`deprecated_since` を設定したインターフェースはレスポンスに `Deprecation` ヘッダーを付け、廃止時期が決まって `sunset_at` を設定した後は `Sunset` ヘッダーも付ける。`sunset_at` は `deprecated_since` の最低 12 か月後とする。

破壊的変更の検出は、TypeSpec から生成した OpenAPI と、実際に配布した内容を固定したリリースベースラインとの比較で行う。ベースラインを更新してよいのはリリース作業の一部としてだけである。リリースせずに更新すれば実際の後退を検出できなくなり、リリースしても更新しなければベースラインが古くなって同じ結果になる。

## Wire bodies in the contract

TypeSpec が `@body` に宣言する型は、サーバーが実際に受理し返す JSON そのものである。サーバーが送らない封筒を契約側で 1 段挟まない。ハンドラーが `map[string]any{"groups": ...}` のような封筒を書くときにだけ、契約もその封筒を持つ。パスやクエリのパラメータは要求本体のプロパティにしない。本文が JSON でない接点 (CSV のアップロード、SET の受信、XML メタデータ、画像の配信、メトリクスの公開) は、その media type と本文の型をそのまま宣言する。

同じ規則が値にも及ぶ。enum の値は member 名の複製ではなく線上の値そのものを書き、標準が定める値は標準から、独自の値は Go の定数から写す。`unknown` は「任意の JSON 値」と読まれるので、実在する型やモデルがあるならそれを書き、本当に任意でよい場合だけ、なぜ任意なのかを `@doc` に残す。

## String length limits

文字列フィールドの長さ上限は、公開契約、Go の検証、PostgreSQL の制約、UI の入力欄という 4 つの境界に同じ数で現れる。数が同じでも数える単位が違えば別々の上限になるので、単位を先に固定する。

単位は **Unicode コードポイント**である。TypeSpec の `@maxLength`、そこから生成される OpenAPI の `maxLength`、PostgreSQL の `char_length()`、Go の `utf8.RuneCountInString` は、いずれもコードポイントを数える。zog の `String().Max(n)` だけは UTF-8 バイト数を数えるため、文字列フィールドには使わない。代わりに `backend/shared/spec` の `Chars` と `CharsAtMost` を使う。バイト数で数えると、上限 100 の名前が英字なら 100 文字、日本語なら 33 文字になり、契約に書いた数が意味を持たなくなる。

例外は、準拠する標準自身がオクテットで上限を定めている値に限る。メールアドレスは RFC 5321 の 254 オクテット、realm は DNS ラベルの 63 オクテットである。どちらも書式を ASCII に限っているため、実際の値ではコードポイント数と一致する。

上限を置く値は、次の既定の区分から選ぶ。外部の標準も固定の表示面も関与しない値のために、新しい数を持ち込まない。

| Class | Limit | Applies to |
| --- | --- | --- |
| Handle | 64 | IdMagic が採番する Aggregate の ID、および関係名や型名のような語彙的な名前 |
| Name | 100 | 一行の名前 |
| DisplayName | 200 | 利用者に見せる表示名とメールの件名 |
| ExternalID | 256 | 呼び出し側の資源空間から来る識別子 |
| Description | 500 | 数文の説明、パターン、表示テンプレート |
| URI | 2048 | URL と URI |
| Expression | 4096 | CEL のような式 |
| PlainBody | 8000 | 平文の本文 |
| RichBody | 20000 | HTML の本文 |

次の値は外部の標準か固定の表示面から上限が決まるので、区分の外に置く。

| Field | Limit | Why |
| --- | --- | --- |
| メールアドレス | 254 | RFC 5321 が定める経路の上限 |
| `Tenant.realm` | 63 | DNS ラベル |
| `WorkloadTrustBundle.trust_domain` | 255 | DNS 名 |
| `client_id` | 128 | UUID を収めたうえで、他の認可サーバーから移入した値も受けられる幅 |
| パスワード | 128 | `PasswordPolicy` の既定の上限 |
| ブランディングの短いラベル | 80 | サインイン画面とメールの固定枠に収まる幅 |
| ブランディングの補足テキスト | 280 | 同上 |

連携相手が値を決める識別子にも上限を置く。上限を置かないという選択は、実際には上限を PostgreSQL の btree に任せることでしかない。btree v4 は索引行 1 件を 2704 バイトに制限するので、主キーや一意索引の成分になっている列は、宣言の有無にかかわらずそこで頭打ちになる。宣言しなければ、超過は書き込み時の `SQLSTATE 54000` として現れ、どのフィールドをどこまで短くすればよいのかを利用者に伝えられない。索引タプルは値を圧縮しようとするため、その境界がどこに来るかは値の中身によっても動く。

そこで、索引の鍵の成分になる列では、性質の違う 2 つの上限を重ねる。

| Limit | Unit | Role |
| --- | --- | --- |
| 契約の上限 | コードポイント | 公開契約が示す数。TypeSpec の `@maxLength` と Go の `spec.Chars` が持つ |
| 資源の上限 | バイト | 索引行に収まることの保証。Go の `spec.KeyString` と PostgreSQL の `octet_length` が持つ |

資源の上限だけがバイトで数えるのは、btree が制限しているものがバイトだからである。単位をコードポイントに揃える原則の例外は、これで 2 つになる。標準自身がオクテットで上限を定めている値と、上限が守る資源そのものがバイトで測られる場合である。契約の上限をコードポイントで置くだけでは足りない。標準が 1024 文字と定める識別子でも、すべて 3 バイト文字なら 3072 バイトになり、宣言した契約を満たしたまま btree で落ちる。

複合鍵では、列ごとではなく鍵 1 件の合計が索引行に収まらなければならない。鍵ごとにバイトの予算を持ち、その合計を 2400 バイト以下とする。btree の 2704 バイトとの差が、索引タプル自身が使う領域と、将来列を足すための余白になる。この条件を守るのはこの文書ではなく `backend/shared/spec` のテストで、上限を後から広げて予算を割ったときにそこで分かる。

上限には、標準が定める数をそのまま使わない。標準を超える値を出す実装は実在しうるので、標準の数は「上限が実運用で拘束的でないことの根拠」として記録し、上限そのものはその外側に置く。これらは安全のために置く資源の境界であって、業務上望ましい長さではない。

| Column | Code points | Bytes | Why |
| --- | --- | --- | --- |
| `SamlServiceProvider.entity_id` | 2048 | 2048 | URI 区分。`saml-schema-metadata-2.0.xsd` の `entityIDType` は 1024 文字を定めるが、非準拠の SP を拒否しないためその外側に置く |
| SAML AuthnRequest の `ID` | 256 | 256 | 資源の上限。`xs:ID` に長さの規定はない |
| `WsFedRelyingParty.wtrealm` | 2048 | 2048 | URI 区分。WS-Federation は `wtrealm` を URI としか定めない |
| 外部 IdP の subject | 512 | 1024 | 資源の上限。OpenID Connect Core 1.0 §2 は `sub` を 255 ASCII 文字以下と定めるが、SAML の NameID には規定がない |
| 外部 IdP の SAML `Response` の `ID` | 256 | 256 | 資源の上限 |
| DPoP proof と client assertion の `jti` | 256 | 256 | 資源の上限。RFC 7519 に長さの規定はない |
| WebAuthn の credential ID | 2048 | 2048 | 資源の上限。WebAuthn は 1023 バイト以下と定めるので base64url で 1364 文字になる |

IdMagic が採番する鍵の成分にも上限を置くが、置く境界が違う。SCIM の `id`（RFC 7643 が service provider の割り当てと定める値）、署名鍵の `kid`（RFC 7638 の JWK thumbprint）、発行した token の `jti` は、いずれも Handle 区分の 64 とする。実値はそれぞれ hex 32 文字、base64url 43 文字、UUID であり、ASCII しか取らないので契約の上限と資源の上限が同じ数になる。これらには Go 側の検査を置かず、`CHECK` だけを置く。拒否すべき外部の入力が無いためである。超過するのは採番の実装が壊れたときだけで、それは利用者の誤りではないので 422 にしない。

鍵の成分ではない外部由来の値にも上限を置く。btree の制約は受けないが、保存量を無制限にしてよい理由にはならない。`tls_client_auth_subject_dn` は URI 区分の 2048、連携先の同期を隔離した理由を記録する文字列は Description 区分の 500 とする。後者は利用者の入力ではなく連携先が返したエラー文なので、拒否せず書き込み側で切り詰める。長いエラー文を理由に隔離そのものが失敗する方が高くつく。外部入力を通らない固定長のダイジェスト（各種 token hash）には上限を置かない。

4 つの境界は同じ数を持つが、役割は同じではない。

| Boundary | Role |
| --- | --- |
| TypeSpec の `@maxLength` | 公開契約。OpenAPI と生成ドキュメントが示す数の出どころ。 |
| Go の domain スキーマ | 唯一の強制点。ここを通らない書き込み経路を作らない。 |
| PostgreSQL の `CHECK (char_length(col) <= N)` | 最後の防壁。ここで落ちるのは実装の不具合であり、利用者向けエラーの発生源にしない。索引の鍵の成分では `octet_length(col) <= M` を同じ `CHECK` に併記する。 |
| UI の `maxLength` 属性 | 入力の補助。保証ではない。 |

UI だけは数える単位が違う。HTML の `maxLength` が数えるのは UTF-16 のコード単位なので、基本多言語面の外にある絵文字などは 1 文字が 2 と数えられる。UTF-16 のコード単位数はコードポイント数を下回らないため、ずれは必ず厳しい側に出る。入力欄が通した値をサーバが長さで拒むことはなく、逆に絵文字だけの入力では上限の手前で入力欄が止まる。この不一致は許容する。ブラウザーの `maxLength` にコードポイントを数えさせる方法はなく、自前で数え直して入力を書き換えると、入力中の変換や取り消しの邪魔になるためである。

`CHECK` に置いてよいのは、長さのように安定した構造上の境界に限る。スキームの allowlist のような、変わりうる入力規則を DDL へ入れない。規則が変わるたびに全配備でスキーマ移行が必要になるうえ、利用者の入力誤りで落ちる規則をこの層に置くと、上の表が定める「実装の不具合だけが落ちる場所」という役割が崩れるためである。たとえばテナントのフッターリンクに `https` しか許さない規則は Go の domain が持ち、`CHECK` は長さだけを見る。

上限違反は、解析できた内容が業務規則に違反する場合に当たるので、[HTTP error responses](#http-error-responses) が定める 422 で返す。違反したフィールドと上限を `detail` に載せ、何を短くすればよいかを利用者が判断できるようにする。

ただしこれは管理 API の話である。SAML、WS-Federation、OAuth 2.0、SCIM、WebAuthn のように相手側のプロトコルが応答の形を定めている接点では、長さの違反もそのプロトコルが定めるエラーとして返す。AuthnRequest を送ってきた相手に Problem Details を返しても読めない。上限の数は同じでも、それを伝える語彙は接点ごとに違う。

資源の上限は書き込み経路にだけ課す。索引の鍵の成分を検索の条件として受け取る経路（`GET /Users/{id}` のような）は、長い値を渡されても索引行を作らないので、上限を超えた値でも 422 ではなく通常の「見つからない」として扱う。
