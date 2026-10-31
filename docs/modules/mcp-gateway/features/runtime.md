# MCP Gateway の実行環境

## 概要

この文書は、改修していない MCP サーバーの手前で `tools/call` を実行前に認可する実行単位 `idmagic-mcp-gateway` の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 2026-07-28 版の MCP の要求の解釈、評価する操作の選択、[リソースアクセスの評価](../../oauth2/features/resource-access.md)の呼び出し、許可した要求の転送、監査ログと指標 |
| 行為者 | ゲートウェイを配置する運用者、MCP クライアント、McpGateway |
| 扱わないもの | 判断と主体の導出は OAuth2 と AccessControl が扱う。`tools/list` の絞り込み、Envoy の `ext_proc` と Envoy の構成での下流向けのトークンの差し替え、`requires_approved_operation` の操作、mTLS に束縛したトークン |

ゲートウェイは単体のバイナリで、状態を持たない。
対象は [MCP の 2026-07-28 版](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)の Streamable HTTP に固定する。

## モデル

### 設定

ゲートウェイは、起動の引数で与えた YAML の設定ファイルと、そこから参照するシークレットのファイルだけを読み、環境変数を読まない。

| 項目 | 内容 | デフォルト値 |
| --- | --- | --- |
| `mode` | `proxy` は自分で上流へ転送する。`envoy_ext_authz` は Envoy の判定の入口になり、転送しない | `proxy` |
| `listen` | MCP クライアントか Envoy を受ける待ち受けのアドレス | 必須 |
| `metrics_listen` | 指標を公開する待ち受けのアドレス | なし（公開しない） |
| `resource` | 登録の `resource` と同じ、ゲートウェイの公開 URL。経路がゲートウェイの MCP の入口になる | 必須 |
| `upstream` | 保護する MCP サーバーの MCP の入口の URL | `proxy` では必須。`envoy_ext_authz` では指定できない |
| `idmagic.realm_url` | テナントの realm の `https://` の URL | 必須 |
| `idmagic.ca_file` | IdMagic のサーバーの証明書を検証する CA の PEM。省くと OS の信頼する CA を使う | なし |
| `idmagic.client_id` | 登録に束縛した機密クライアントの ID | 必須 |
| `idmagic.client_secret_file` | クライアントシークレットを置いたファイル。末尾の改行を除いて使う | 必須 |
| `idmagic.timeout` | 一回の評価の上限 | `5s` |
| `allowed_origins` | 受け入れる `Origin` の値の一覧 | なし |
| `max_body_bytes` | 要求の本文の上限 | `4194304` |
| `downstream.resource` | 下流へ渡すトークンの宛先で、IdMagic に登録した `resource`。指定すると転送ごとにトークンを差し替える | なし（資格情報を除いて転送する） |
| `downstream.dpop_key_file` | 下流のトークンを束縛する P-256 の秘密鍵の PEM。`downstream.resource` を指定すると必須 | なし |
| `mapping.methods` | `tools/call` 以外で転送する方法から操作への対応 | なし |
| `mapping.tools` | ツール名から操作への対応 | なし |

操作は `resource_type`、`relation` と、`resource_id` か `resource_id_argument` のちょうど一方を持つ。
`resource_id_argument` は `tools/call` の `arguments` の直下のキーを名指し、その文字列の値を `resource_id` にする。

```yaml
mapping:
  methods:
    server/discover: { resource_type: mcp_server, relation: discover, resource_id: main }
    tools/list: { resource_type: mcp_server, relation: list_tools, resource_id: main }
  tools:
    read_document: { resource_type: document, relation: viewer, resource_id_argument: document_id }
```

### 公開 URL と登録の関係

MCP クライアントは `resource` を宛先としてトークンを取得し、DPoP の証明の `htu` に `resource` を書く。
ゲートウェイは `htm` を `POST`、`htu` を `resource` として評価へ渡す。
このため `htu` は常に登録の `resource` の配下であり、前段のプロキシで内部の URL が変わっても値は変わらない。

### 宣言のないツールの操作

宣言のない `tools/call` は、資源の型 `mcp_tool`、関係 `unmapped`、資源の識別子にツール名を指定して評価へ送る。
この評価は試行者を監査へ残すためにあり、結果によらず上流へ転送しない。

## 操作

### 要求の執行

#### REQ-MCPGATEWAY-001 設定の誤りでは起動せず、シークレットをログへ出さない

- 必須の項目がないか、未知のキーがあるか、`resource`、`upstream`、`realm_url` の URL が不正な場合、McpGateway は、起動せず、標準エラーへ理由を出して 0 以外で終了する。
- `realm_url` か `resource` が `https://` でない場合、McpGateway は、起動しない。
- `mode` が `proxy` と `envoy_ext_authz` のどちらでもないか、`mode` が `envoy_ext_authz` で `upstream` を持つ場合、McpGateway は、起動しない。
- `downstream.resource` が `https://` の URL でないか、`downstream.dpop_key_file` を P-256 の秘密鍵として読めないか、`mode` が `envoy_ext_authz` で `downstream` を持つ場合、McpGateway は、起動しない。
- 操作が `resource_id` と `resource_id_argument` の両方を持つか、どちらも持たないか、`mapping.methods` が `tools/call` を名指す場合、McpGateway は、起動しない。
- ログと指標を書くとき、McpGateway は、提示されたトークン、DPoP の証明、クライアントシークレット、`resource_id_argument` で名指したもの以外のツールの引数を含めない。
- **判断**：`resource` にも `https://` を求める。トークンの宛先と DPoP の `htu` になる値であり、TLS を前段で終端しても利用者から見える URL は `https://` である。

#### REQ-MCPGATEWAY-002 解釈できない要求は評価せず上流へ送らない

- `resource` の経路以外への要求を受けたとき、McpGateway は、保護リソースのメタデータを除いて 404 を返す。
- `POST` 以外の方法を受けたとき、McpGateway は、405 と `Allow: POST` を返す。
- `Origin` があり `allowed_origins` にない場合、McpGateway は、403 を返す。
- `Content-Type` が `application/json` でないか、`Content-Encoding` が `identity` 以外の場合、McpGateway は、415 を返す。
- 本文が `max_body_bytes` を超える場合、McpGateway は、413 を返す。
- 本文が UTF-8 の JSON として不正か、後続のデータを持つ場合、McpGateway は、400 と JSON-RPC の `-32700` を返す。
- 本文が配列（JSON-RPC のバッチ）であるか、`jsonrpc` が `2.0` でないか、文字列か数の `id` がないか、文字列の `method` がないか、`params` がオブジェクトでない場合、McpGateway は、400 と JSON-RPC の `-32600` を返す。
- 本文のどの階層のオブジェクトでも、大文字と小文字を区別せずに同じになるキーが複数ある場合、McpGateway は、400 と JSON-RPC の `-32600` を返す。
- `MCP-Protocol-Version`、`Mcp-Method`、`Mcp-Name` のヘッダーが必要なのにないか、複数あるか、本文の値と一致しない場合、McpGateway は、400 と JSON-RPC の `-32020` を返す。
- `Mcp-Name` が `=?base64?...?=` の形の場合、McpGateway は、復号した値を本文と比べる。
- `params._meta` の `io.modelcontextprotocol/protocolVersion` が `2026-07-28` でない場合、McpGateway は、400、JSON-RPC の `-32022`、`supported` に `2026-07-28` だけを持つ `data` を返す。
- `tools/call` でなく `mapping.methods` にもない方法の場合、McpGateway は、404 と JSON-RPC の `-32601` を返す。
- この要件のどれかで拒否した場合、McpGateway は、評価を呼ばず、上流へ送らない。
- **判断**：バッチ、通知、古い版を通さない。2026-07-28 版は一つの本文に一つの要求だけを許し、通知を定めていない。
- **判断**：キーの重複を拒否する。JSON の実装ごとに先勝ちと後勝ち、大文字と小文字の扱いが異なり、ゲートウェイと上流で別の方法やツール名に読まれると許可の取り違えになる。
- **例**：EX-MCPGATEWAY-002-01

#### REQ-MCPGATEWAY-003 評価が許可した要求だけを上流へ送る

- 宣言したツールの `tools/call` と宣言した方法を受けたとき、McpGateway は、一つの要求につき一回だけ評価を呼び、結果を再利用しない。
- `decision=true` の応答を受けたとき、McpGateway は、受け取った本文をそのまま上流へ一回だけ転送する。
- `Authorization` が一つでないか、`Bearer` か `DPoP` のトークンを持たないか、トークンがカンマを含む場合、McpGateway は、401、`resource_metadata` を持つ `WWW-Authenticate`、JSON-RPC の `-32001` を返し、評価を呼ばず、上流へ送らない。
- `DPoP` のヘッダーが複数あるか、値がカンマを含む場合、McpGateway は、401、`error="invalid_token"` を持つ `WWW-Authenticate`、JSON-RPC の `-32001` を返し、評価を呼ばず、上流へ送らない。
- 理由が `token_rejected` か `sender_proof_invalid` の不許可を受けたとき、McpGateway は、401、`error="invalid_token"` を持つ `WWW-Authenticate`、JSON-RPC の `-32001` を返し、上流へ送らない。
- それ以外の不許可を受けたとき、McpGateway は、403 と JSON-RPC の `-32000` を返し、上流へ送らない。
- 評価の通信が失敗するか、200 以外の応答か、解釈できない応答を受けた場合、McpGateway は、503 と JSON-RPC の `-32002` を返し、上流へ送らない。
- 評価が 429 と `Retry-After` を返した場合、McpGateway は、同じ `Retry-After` を 503 に付ける。
- 不許可と評価不能で応答するとき、McpGateway は、評価の理由を MCP クライアントへ返さない。
- `resource_id_argument` の値がないか文字列でない場合、McpGateway は、403 と JSON-RPC の `-32000` を返し、評価を呼ばず、上流へ送らない。
- 宣言のない `tools/call` を受けたとき、McpGateway は、[宣言のないツールの操作](#宣言のないツールの操作)を評価し、結果が許可でも 403 と JSON-RPC の `-32000` を返し、上流へ送らない。
- ツール名が 129 文字以上の宣言のない `tools/call` を受けたとき、McpGateway は、403 と JSON-RPC の `-32000` を返し、評価を呼ばず、上流へ送らない。
- **判断**：カンマを複数の値として扱う。トークンと DPoP の証明はカンマを含まず、Envoy は重複したヘッダーを連結して渡すので、値だけでは一つか複数かを区別できない。
- **判断**：`tools/list` と `server/discover` も評価する。未認証の呼び出しへ 401 を返すことが MCP の認可の入口であり、公開の例外を設けると設定の誤りが素通しになる。
- **判断**：拒否の理由を返さない。別の資源宛てのトークンの有効性や権限の構成を呼び出し元へ知らせない。
- **例**：EX-MCPGATEWAY-003-01、EX-MCPGATEWAY-003-02

#### REQ-MCPGATEWAY-004 転送は提示された資格情報を上流へ渡さず、応答を逐次返す

- 上流へ転送するとき、McpGateway は、`Authorization`、`DPoP`、`Cookie` のヘッダーと URL のクエリを除く。
- `downstream` がないとき、McpGateway は、上流へ `Authorization` と `DPoP` を送らない。
- 上流の応答を受けたとき、McpGateway は、状態コード、ヘッダー、本文を変えずに返し、`text/event-stream` の各イベントをためずに書き出す。
- MCP クライアントが接続を閉じたとき、McpGateway は、上流への要求を取り消す。
- 上流へ接続できない場合、McpGateway は、502 と JSON-RPC の `-32003` を返し、再送しない。
- **判断**：トークンを渡さない。MCP の認可の仕様はトークンの素通しを禁じ、このトークンの宛先はゲートウェイの `resource` である。上流はゲートウェイからだけ到達できるように配置する。
- **判断**：長く開く `subscriptions/listen` は開始時に一度だけ評価する。開いた後の権限の変更は既存の接続を切らない。

#### REQ-MCPGATEWAY-005 保護リソースのメタデータを公開する

- `/.well-known/oauth-protected-resource` か、その後に `resource` の経路を続けた位置への `GET` を受けたとき、McpGateway は、評価を呼ばず、`resource` と `authorization_servers` に `idmagic.realm_url` を持つ JSON を 200 で返す。
- **判断**：改修しないサーバーはゲートウェイの `resource` を知らないので、RFC 9728 の文書はゲートウェイが返す。MCP クライアントはこの文書から認可サーバーを見つける。

#### REQ-MCPGATEWAY-006 要求ごとに監査ログと指標を残し、IdMagic の記録と相関できるようにする

- 要求を処理したとき、McpGateway は、標準出力へ JSON の 1 行で、種類 `mcp_gateway_request`、相関 ID、方法、ツール名、資源の型、関係、資源の識別子、結果、判断の理由、評価の応答の `X-Request-ID` を書く。
- 要求を受けたとき、McpGateway は、MCP クライアントの `X-Request-ID` を使わずに相関 ID を作り、評価と上流への要求と応答の `X-Request-ID` に載せる。
- 結果は、`forwarded`、`permitted`、`denied`、`unauthenticated`、`invalid`、`unavailable`、`upstream_failed` のどれかにする。
- ツール別の呼び出しを数えるとき、McpGateway は、`mcp_gateway_requests_total` のラベル `method`、`tool`、`outcome` を使い、`tool` を宣言したツール名か `unmapped` か空に限る。
- 評価の時間を測るとき、McpGateway は、`mcp_gateway_evaluation_duration_seconds` を使う。
- **判断**：主体と Agent をゲートウェイに持たせない。評価に使った主体は IdMagic の `ResourceAccessEvaluated` が記録し、評価の応答の `X-Request-ID` で突き合わせる。評価まで届かない要求と評価できなかった要求には、信頼できる主体がなく、主体を残さない。
- **判断**：指標のラベルを宣言した名前に限る。呼び出し元が決めるツール名で系列を増やさせない。

#### REQ-MCPGATEWAY-007 Envoy の `ext_authz` の入口は判定だけを返し、設定例は失敗時に通さない

- `mode` が `envoy_ext_authz` で許可の判断を得たとき、McpGateway は、本文のない 200、相関 ID の `X-Request-ID`、`x-envoy-auth-headers-to-remove: authorization, dpop, cookie` を返し、結果を `permitted` にして、自分では上流へ送らない。
- `mode` が `envoy_ext_authz` で拒否するとき、McpGateway は、REQ-MCPGATEWAY-002 と REQ-MCPGATEWAY-003 と同じ状態コード、ヘッダー、本文を返す。
- `mode` が `envoy_ext_authz` で要求の URL がクエリを持つ場合、McpGateway は、404 を返し、評価を呼ばない。
- `mode` が `envoy_ext_authz` で `x-envoy-auth-partial-body` が `false` でない MCP の入口への要求を受けた場合、McpGateway は、413 を返し、評価を呼ばない。
- `mode` が `envoy_ext_authz` で `x-envoy-auth-partial-body` を持つメタデータの位置への要求を受けた場合、McpGateway は、404 を返す。
- `x-envoy-auth-partial-body` を持たないメタデータの位置への `GET` を受けたとき、McpGateway は、REQ-MCPGATEWAY-005 の応答を返す。
- [設定例](../../../../examples/envoy-mcp-gateway/README.md)の Envoy は、ゲートウェイの許可の応答を受けたとき、受け取った本文を変えずに、`Authorization`、`DPoP`、`Cookie` を除いて上流へ一回だけ転送し、`text/event-stream` の各イベントをためずに返す。
- 設定例の Envoy は、ゲートウェイの 4xx の拒否の応答を受けたとき、その状態コード、本文、`WWW-Authenticate`、`Allow`、`Retry-After`、`X-Request-ID`、`Content-Type`、`Cache-Control` を MCP クライアントへ返し、上流へ送らない。
- 設定例の Envoy は、ゲートウェイへ接続できないか、応答が遅れるか、ゲートウェイが 5xx を返す場合、503 と JSON-RPC の `-32002` を返し、ゲートウェイの `Retry-After` を含めず、上流へ送らない。
- 設定例の Envoy は、本文が `max_request_bytes` を超える場合、413 を返し、ゲートウェイと上流を呼ばない。
- 設定例の Envoy は、ゲートウェイと上流の `X-Request-ID` が両方あるとき、MCP クライアントと上流へゲートウェイの値だけを渡す。
- **判断**：`ext_proc` ではなく HTTP 版の `ext_authz` を使う。判定まで本文を止める点は変わらず、本文や応答を書き換える要件がなく、gRPC と制御プレーンの依存を増やさない。
- **判断**：評価の流量制限の `Retry-After` を諦める。HTTP 版の `ext_authz` は判定の入口の 5xx を障害として扱い、応答の本文とヘッダーを捨てる。不許可にはならず、再試行の目安だけが失われる。
- **判断**：クエリを拒否する。HTTP 版の `ext_authz` はクエリを除けず、判定していない値を上流へ渡さないためである。
- **判断**：Envoy は 200 を許可と読むので、Envoy 経由の判定の呼び出しへメタデータを返さない。設定例はメタデータだけを `ext_authz` なしでゲートウェイへ送る。
- **判断**：上流へ接続できないときの応答は Envoy の 503 のままにする。認可の結果ではなく、JSON-RPC への変換に設定を増やす理由がない。
- **例**：EX-MCPGATEWAY-007-01

#### REQ-MCPGATEWAY-008 `downstream` があれば許可した呼び出しごとに下流宛のトークンへ差し替える

- `downstream` があるとき、McpGateway は、評価の要求の `token_exchange_resource` に `downstream.resource` を載せる。
- 許可の判断と `token_exchange_grant` を受けたとき、McpGateway は、提示されたトークンと許可を `downstream.dpop_key_file` の鍵の証明とともに `/token` で交換し、結果を `Authorization: DPoP` と `downstream.resource` を `htu` にした新しい証明で上流へ一回だけ転送する。
- 許可の判断に `token_exchange_grant` がないか、交換が失敗した場合、McpGateway は、503 と JSON-RPC の `-32002` を返し、上流へ送らない。
- 交換の結果の `token_type` が `DPoP` でない場合、McpGateway は、503 と JSON-RPC の `-32002` を返し、上流へ送らない。
- 交換が失敗した要求を記録するとき、McpGateway は、結果を `unavailable`、判断の理由を `token_exchange_failed` にし、トークンと許可をログへ出さない。
- `downstream` がないとき、McpGateway は、`token_exchange_resource` を送らず、交換しない。
- **判断**：トークンを呼び出しごとに得て、再利用しない。REQ-OAUTH2-079 の許可は一回限りで、結果を保持しないことでゲートウェイを状態のないままにできる。
- **判断**：`htu` に `downstream.resource` を使う。REQ-MCPGATEWAY の公開 URL と同じ考え方で、下流の前段で内部の URL が変わっても値は変わらない。

### 例

#### EX-MCPGATEWAY-002-01 重複したキーで読み替える要求は上流へ届かない

- **前提**：`read_document` を許可するトークンがある。
- **操作**：`params` に `name` と `NAME` を持つ本文を送る。
- **結果**：400 と `-32600` になり、評価も上流の呼び出しも起きない。

#### EX-MCPGATEWAY-003-01 許可されたツールの呼び出しだけが上流で実行される

- **前提**：Agent に `d1` の `viewer` の関係がある。
- **操作**：`read_document` を `d1`、`d2` の順に呼ぶ。
- **結果**：`d1` は上流のツールを一回実行し、`d2` は 403 になり実行しない。

#### EX-MCPGATEWAY-003-02 評価できないときは上流へ届かない

- **前提**：IdMagic へ接続できない。
- **操作**：`read_document` を呼ぶ。
- **結果**：503 になり、上流のツールは実行されない。

#### EX-MCPGATEWAY-007-01 Envoy の背後でも不許可の呼び出しは上流へ届かない

- **前提**：設定例の Envoy が `envoy_ext_authz` のゲートウェイを判定に使い、`d1` だけを許可する。
- **操作**：`read_document` を `d1`、`d2` の順に Envoy へ送る。
- **結果**：`d1` は資格情報のない要求として上流へ一回届き、`d2` はゲートウェイの 403 と `-32000` になり上流へ届かない。

## セキュリティ上の考慮

上流へゲートウェイ以外から到達できると、すべての執行を迂回できる。
上流はループバックか、ゲートウェイからだけ到達できるネットワークに置く。

クライアントシークレットのファイルは、ゲートウェイを動かすアカウントだけが読めるように置く。
ゲートウェイは TLS を終端しないので、MCP クライアントとの間に TLS を終端する前段を置く。
指標の待ち受けは認証を持たないので、内部のネットワークにだけ公開する。

下流の DPoP の鍵のファイルは、クライアントシークレットと同じく、ゲートウェイを動かすアカウントだけが読めるように置く。
下流へ差し替えるとき、ゲートウェイのクライアントには `urn:ietf:params:oauth:grant-type:token-exchange` のグラントと交換するスコープを許可し、評価の規則の `token_exchange_resources` に下流の `resource` を宣言する。

`envoy_ext_authz` では、上流へ Envoy 以外から到達できず、Envoy のすべての経路が `ext_authz` を通ることが執行の前提になる。
`allowed_headers` から `origin` を除くと `Origin` の検査だけが効かなくなり、ゲートウェイから検出できない。
配置後は[迂回の検査](../../../../examples/envoy-mcp-gateway/README.md#迂回の検査)を行う。
判定の入口へ直接届いても上流へは転送されないが、評価の流量の予算を消費するので、Envoy からだけ到達できるように置く。

`Mcp-Param-*` のヘッダーと本文の一致は、ツールのスキーマを持つ上流が検査する。
ゲートウェイの判断は本文だけを読むので、ヘッダーの値は判断に影響しない。

## 実現方式

### アーキテクチャ

中核は `Inspect(headers, body) -> Message` と `Decide(message, mapping) -> Evaluation` の決定論的な計算である。
評価の呼び出しと HTTP の転送は、この計算の外の作用として編成する。
