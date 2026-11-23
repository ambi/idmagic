# MCP と外部サービスの接続

この文書は、Agent から MCP や業務 API へ接続する構成の選び方、ツール発見、上流の資格情報、外部 OAuth 連携を説明する。
[ID と認証](ai-agent-demo-identity.md)、[本人の委任](ai-agent-demo-delegation.md)、[認可と人の承認](ai-agent-demo-authorization.md)と組み合わせて使う。
[ガイドの入口](ai-agent-demo.md)に利用パターンの一覧がある。

## 配置の選択

| 構成 | 向く接続先 | 配置するもの | 上流の資格情報 |
| --- | --- | --- | --- |
| セルフホスト型 Gateway | 閉域やローカルの MCP、自分で運用する MCP | `idmagic-mcp-gateway` と HTTPS の公開口 | デフォルトでは呼び出し元の資格情報を除去。必要なら下流 Token Exchange |
| マネージド型、`token_exchange` | IdMagic のトークンと DPoP を検証できる公開 MCP | IdMagic の API プロセス内で動作 | Gateway 専用鍵に束縛した上流用トークン |
| マネージド型、`token_vault` | 外部 OAuth で本人が接続する公開 MCP | IdMagic 内の Gateway と TokenVault コネクタ | 本人が接続した外部サービスのトークン |
| Envoy `ext_authz` | Envoy の前段がある既存 MCP | Envoy と判定用 Gateway | 設定例は呼び出し元の資格情報を除去 |
| 独自 PEP | MCP 以外の業務 API | 操作直前の評価を組み込んだアプリ | 業務側の設計に従う |

「マネージド型」は Gateway を別途配置しない意味であり、IdMagic を自社運用する場合も選べる。
上流は公開アドレスの HTTPS が前提で、localhost やプライベート IP の MCP はセルフホスト型を選ぶ。
マネージド型の公開 URL と配置方法は作成後に変えられないため、変更時は新しい登録へ移行する。

## セルフホスト型 Gateway

### 接続先の登録と配置

1. MCP リソースサーバーを「セルフホスト型」で登録し、Agent に公開する HTTPS のリソース URI を指定する。
2. 登録結果の評価用クライアント ID とシークレットを Gateway 側に配置する。Agent の認証用クライアントとは分ける。
3. Gateway の設定を作り、前段の TLS、Agent からの経路、上流への到達経路を構成する。
4. Gateway を起動し、カタログを管理画面で受信できることを確認する。
5. 公開するツールを選択し、Agent に操作と対象を割り当てる。
6. Agent の接続先とトークン要求の `resource` を、Gateway の公開 URL にする。

以下は配置先で使う設定例で、各値を自分の登録結果と URL に置き換える。
`client_secret_file` はシークレットだけを格納したファイルへのパスである。

```yaml
mode: proxy
listen: 127.0.0.1:8090
resource: https://mcp.example.com/mcp
upstream: http://127.0.0.1:3001/
idmagic:
  realm_url: https://id.example.com/realms/default
  client_id: "<評価用クライアント ID>"
  client_secret_file: /etc/idmagic-mcp-gateway/client-secret
mapping:
  catalog: true
```

Gateway の実行ファイルは `idmagic-mcp-gateway --config <設定ファイル>` で起動する。
この設定で公開口を作るには、別途 HTTPS の前段が必要である。
`realm_url` と `resource` は HTTPS が必須のため、ローカルの HTTP の issuer をそのまま指定しない。
リポジトリ内で接続までまとめて再現する場合は、TLS の準備も含む `mise run test-mcp-client-walkthrough` を使う。

`mapping.catalog: true` は公開設定を IdMagic に集約し、ツールごとの対応を Gateway へ再入力しない構成である。
独自の型や関係に対応付ける場合は `mapping.tools`、接続準備などは `mapping.methods` に設定する。
宣言も公開登録もないツールは転送されない。
上流は Gateway からだけ到達できるようにし、Agent が上流へ直接接続して認可を迂回できないことを確認する。

### ツール発見と公開

| 方法 | 前提 | 動作と管理者の作業 |
| --- | --- | --- |
| Gateway からの報告 | Gateway が上流へ接続でき、`mapping.catalog: true` | 起動後と上流の一覧変更通知で `tools/list` を報告する。管理画面で受信時刻、追加、変更、消えた候補を確認し、公開するものを選ぶ |
| IdMagic から直接取得 | 有効なセルフホスト型の登録、公開 HTTPS の MCP、対応する Protected Resource Metadata | 管理画面で「MCP サーバーから取得」を実行し、取得結果と候補のスコープを確認して保存する |
| 管理者が登録 | 自動取得できないサーバー、マネージド型の接続先 | 公開するツール名、表示名、規則を入力する |

直接取得は、対象のメタデータが登録した resource と IdMagic の issuer を信頼すると宣言していることを確かめてから、一覧取得専用のトークンを送る。
任意の URL に Agent の資格情報を送って探索する方式ではない。
候補の取得とスコープの提案は、公開設定や割当を自動で変更しない。
直接取得は管理者が実行する操作であり、定期的な自動発見ではない。
閉域の上流は Gateway からのカタログ報告を使う。
マネージド型にはこの直接取得の機能を適用せず、公開するツールを管理者が登録する。

## マネージド型 Gateway

### 上流が IdMagic のトークンを受け入れる構成

1. 上流の MCP を、同じテナントの有効なリソースサーバーとして登録する。
2. Gateway の評価用クライアントで Token Exchange のグラントを有効にする。
3. 新しい接続先を「マネージド型」で登録し、上流の HTTPS URL と認証方式 `token_exchange` を指定する。
4. 公開する操作の規則に、交換を許す上流のリソース URI を `token_exchange_resources` として設定する。
5. 登録結果の公開 URL と Gateway 専用の公開 JWK を控える。
6. 上流は上流宛てのトークンに加え、その公開鍵の JKT に束縛した DPoP Proof だけを受け入れるように設定する。
7. Agent に公開操作を割り当て、公開 URL を `resource` として取得したトークンで接続する。

公開 URL は `<issuer>/mcp/{resource_server_id}` として生成される。
Gateway は毎回操作を評価し、許可された要求だけに、一回限りの交換許可で取得した短命な上流用 DPoP トークンを注入する。
元のトークン、Cookie、任意の認証ヘッダーを上流へそのまま流さない。

公開 URL と上流で audience を分けるだけでは、Agent が直接上流用のトークンを取得したときの迂回を防げない。
上流が専用鍵の JKT まで検査することを接続試験に含める。
操作承認を必須とする規則には `token_exchange_resources` を指定できないため、承認済み操作をこの交換で引き継ぐ構成にしない。

セルフホスト型で同じ上流認証を行う場合は、Gateway の `downstream.resource` と `downstream.dpop_key_file` を設定する。
評価結果の交換許可、評価用クライアントのグラント、規則の交換先、および上流側の専用 JKT の検査は同様に必要である。

### 外部 OAuth を使うマネージド MCP

先に次節のコネクタと本人の接続を用意する。
マネージド型の登録で認証方式 `token_vault`、上流の HTTPS URL、コネクタ ID を指定し、上流への `POST` とその URL を許すコネクタ規則を設定する。
Agent には MCP の操作を割り当て、本人の委任を受けたトークンに必要な `token_vault_connection` の認可詳細を含める。

Gateway は現在の操作割当と条件を検査し、TokenVault は本人の接続、明細、コネクタ規則とスコープを検査する。
Agent 自身の M2M トークンを、本人の接続を選ぶ代わりには使えない。
別の利用者の接続が混ざらないことと、委任または接続の取消後に転送されないことを確かめる。

## 外部サービスとの OAuth 連携

TokenVault は、本人が外部 SaaS に接続して得たアクセストークンと refresh token を暗号化して保管し、Agent に渡さず代理実行する。
Agent への本人の委任と、外部サービスへの本人の OAuth 接続は別々に作る。

### 管理者によるコネクタの準備

1. 外部サービスで OAuth アプリを登録し、IdMagic のコールバック `<issuer>/api/token-vault/v1/callback/{connector_id}` を許可する。コネクタ作成後に得た ID を使って確定する。
2. IdMagic のコネクタ管理で、認可エンドポイント、トークンエンドポイント、任意の失効エンドポイント、外部のクライアント ID とシークレットを登録する。
3. 外部に要求してよい `allowed_scopes` と、HTTP メソッド、オリジン、パス接頭辞、必要スコープの実行規則を登録する。
4. 直接の代理実行を使う場合は、`<issuer>/api/token-vault/v1` をリソースサーバーとして登録する。
5. 認可詳細の型 `token_vault_connection` と、その明細を受け付けるスキーマを登録する。

管理 API の入口は `/api/admin/v1/token-vault/connectors` である。
外部 OAuth のクライアント認証は `client_secret_basic` または `client_secret_post` であり、Agent の認証用クライアントとは別物である。
シークレットは後から読み出せない。
規則は、たとえば `method=GET`、`origin=https://api.example.com`、`path_prefix=/documents`、`required_scopes=["documents.read"]` とし、必要な範囲だけを許す。

### 本人の接続と委任

1. 本人がアカウント画面の「外部サービス接続」から対象コネクタを選ぶ。
2. 外部サービスへリダイレクトされ、本人がログインして必要なスコープに同意する。
3. IdMagic へ戻り、接続が有効になったことと付与スコープを確認する。
4. 「接続済みアプリ」で、使う Agent に必要なスコープの代行を付与する。
5. アプリの認可で次の形の `authorization_details` を要求し、利用者のトークンから Agent の代行トークンへ交換する。

```json
[
  {
    "type": "token_vault_connection",
    "identifier": "<コネクタ ID>",
    "actions": ["GET"],
    "locations": ["https://api.example.com"]
  }
]
```

`authorization_details` は認可要求と Token Exchange では JSON 文字列としてエンコードする。
交換で元の明細を広げることはできないため、元の利用者の認可にもこの明細が必要である。
直接の代理実行では交換先の `resource` を Vault の URI にし、マネージド MCP では Gateway の公開 URL にする。
外部接続の作成はブラウザーセッションと PKCE、state の検証を使い、API トークンで本人の接続を勝手に作れない。
ローカルの接続画面は <http://localhost:5173/realms/default/account/connections> である。

### 外部 API の実行

直接呼ぶ場合は `/api/token-vault/v1/connectors/{connector_id}/execute` に、代行トークンを付けて次の JSON を POST する。
DPoP トークンには対応する Proof も付ける。

```json
{
  "method": "GET",
  "url": "https://api.example.com/documents/handbook"
}
```

応答は上流の `status`、`content_type`、`body_base64` を返し、外部のアクセストークンや任意の応答ヘッダーは返さない。
HTTP 200 は代理実行の応答を受け取った意味であり、業務の成功は内側の `status` も確認する。
`headers` で Agent が `Authorization` を指定することはできない。
HTTPS の公開アドレスだけに接続し、リダイレクトを追わず、メソッドとオリジンとパス接頭辞が規則に合う要求だけを送る。

直接の代理実行では、上流の 401 に対して更新後の再送を一度だけ行う場合がある。
マネージド MCP の転送では同じ要求を再送せず、承認の使用を未使用に戻さない。
結果不明の更新を新しい承認でやり直す前には、上流で実行結果を確認する。

### 拒否と切断の確認

| 試すこと | 期待結果 |
| --- | --- |
| 本人の接続なし、別人の接続だけがある | 本人の接続として利用できない |
| M2M トークン、必要な明細なし、許可外の URL | 拒否され、上流へ届かない |
| 本人の Agent への付与を撤回する | 代行トークンが止まる |
| 本人が外部接続を切断する | 以後の代理実行が止まる。外部の失効要求が失敗してもローカルの停止は有効 |
| 管理者がコネクタを無効化する | 接続を保持したまま新しい連携と代理実行を止める |
| 更新が `invalid_grant` になる | 接続を再認可待ちにし、本人の再接続が必要になる |

コネクタの削除は関連する接続も削除し、外部の失効処理を自動では行わない。
接続を取り消す範囲の違いは[運用](ai-agent-demo-operations.md)も参照する。

## Envoy と業務 API

### Envoy の前段

1. Gateway の `mode` を `envoy_ext_authz` にし、`upstream` と `downstream` は指定しない。
2. `resource` を Envoy の公開 URL にし、[配布する Envoy 設定](../../examples/envoy-mcp-gateway/envoy.yaml)の宛先、経路、本文の上限を配置先に合わせる。
3. Envoy が評価不能時に通さず、上流への再試行や hedging を行わず、呼び出し元の再試行指示ヘッダーを除去することを確認する。
4. 上流を Envoy からだけ到達可能にし、トークンなし、許可外 Origin、クエリ付き URL、Gateway 停止時の要求が上流へ届かないことを試す。

この構成は Gateway が判定し、Envoy が転送する。
Gateway の監査の `permitted` は上流の成功を意味しない。
実在の Envoy と配布設定の確認には、Docker を準備して `mise run test-envoy-mcp-gateway` を実行する。
[Envoy の手順](../../examples/envoy-mcp-gateway/README.md)に期待する HTTP 応答も記載する。

### 業務 API の PEP

1. 評価用 confidential クライアントを作り、保護するリソースの登録に紐付ける。
2. API の操作を `access_rules` の型と関係へ対応付ける。
3. 業務処理の直前に、利用者または Agent のトークン、対象、操作、必要な引数と送信者の証明を評価へ送る。
4. `decision=true` のときだけ業務処理を行い、応答を次の操作に使い回さない。

たとえば文書の ReBAC 規則を登録した API の評価フォームは、次の値を持つ。
これは MCP カタログの組み込みの型とは別の、独自規則の例である。

| フィールド | 値の例 |
| --- | --- |
| `token` | 実行者のアクセストークン |
| `resource_type` | `document` |
| `resource_id` | `handbook` |
| `relation` | `viewer` |
| `attributes` | 条件を使う場合の JSON オブジェクトを表す文字列 |

DPoP ではリソースが受け取った `dpop_proof`、実際の `htm` と `htu`、mTLS では検証済み接続の証明書の拇印も渡す。
通信失敗、200 以外の応答、不正な判定応答では業務処理を始めない。
実装の最小例は[参照 PEP](../../examples/resource-server-pep/README.md)であり、`Enforcer.Protect` と操作結果を保持する `OperationLedger` を含む。

## MCP クライアントと確認

Agent の MCP クライアントは Gateway の公開 URL に接続し、そのリソース向けのトークンを取得する。
401 の Protected Resource Metadata から認可サーバーを確認できるが、すでに登録したクライアントの認証方式と資格情報も必要である。
任意の MCP クライアントへ URL を貼れば Agent の登録や本人の付与まで自動で完了するわけではない。

現行 Gateway は MCP の `2026-07-28` の構文と必要なメタデータを検査する。
接続確認には `mise run test-mcp-client-walkthrough` を使い、[公式 SDK のサンプル](../../examples/mcp-client-walkthrough/README.md)の `PrivateKeyJwtProvider` と設定ファイルを自分のクライアントの参考にする。
この自動検証は読み取りの許可、未割当の削除、条件、承認不足の拒否、取消と停止を扱い、外部 SaaS や実際の LLM の判断は扱わない。

| 追加の構成 | 自動確認のコマンド |
| --- | --- |
| マネージド MCP の登録と操作割当 | `mise run test-go-one -- ./backend/shared/http/server_http TestHostedMcpPublicSetupEnforcesAssignments` |
| 委任した本人の外部接続だけを使う | `mise run test-go-one -- ./backend/shared/http/server_http TestHostedMcpUsesOnlyTheDelegatingUsersVaultConnection` |
| 外部トークンを返さず代理実行する | `mise run test-go-one -- ./backend/shared/http/server_http TestExecuteForwardsWithStoredTokenAndNeverReturnsIt` |
| 切断後の実行を拒否する | `mise run test-go-one -- ./backend/shared/http/server_http TestDisconnectRefusesLaterExecution` |

これらはテスト用の上流で検証するため、外部 SaaS では実際のアプリ登録、コールバック、付与スコープ、切断までを別途確認する。
契約の詳細は [Gateway 実行環境](../modules/mcp-gateway/features/runtime.md)、[マネージド MCP](../modules/oauth2/features/hosted-mcp.md)、[TokenVault](../modules/token-vault/README.md)を参照する。
