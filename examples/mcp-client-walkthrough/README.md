# 実在の MCP クライアントで確かめる MCP の執行

公式の MCP TypeScript SDK で書いたクライアントを、改修していない MCP サーバーの手前に置いた `idmagic-mcp-gateway` へ接続し、IdMagic が個々のツールの呼び出しを許可、拒否、停止する様子を確かめる手順である。
クライアントとサーバーは IdMagic 固有のコードを持たない。
クライアントが行うのは、MCP の 2026-07-28 版の認可で定められた発見とトークンの取得だけである。

同じ手順を `mise run test-mcp-client-walkthrough` が自動で再現し、毎日の定期実行で確かめる。

## 確かめる結果

| 段階 | 操作 | 結果 |
| --- | --- | --- |
| 接続 | クライアントが最初の要求を送る | ゲートウェイが 401 を返し、クライアントは保護リソースのメタデータ（RFC 9728）から IdMagic を見つけ、Agent の秘密鍵で署名したアサーション（RFC 7523）でゲートウェイ宛てのトークン（RFC 8707）を得る |
| 許可 | 割り当てた `read_document` を呼ぶ | MCP サーバーで実行される |
| 割当のない操作 | 割り当てていない `delete_document` を呼ぶ | 403。MCP サーバーへ届かない |
| 条件の拒否 | 割り当てたが、条件が拒否する `write_document` を呼ぶ | 403。MCP サーバーへ届かない |
| 承認が必要な操作 | 人の承認を求める `publish_document` を呼ぶ | 503 で安全側に止まり、MCP サーバーへ届かない |
| 撤回 | 管理者が `read_document` の割当を取り消し、同じクライアントで呼ぶ | トークンを取り直さずに、次の呼び出しから 403 |
| 停止 | 管理者が Agent を無効にする | 手元のトークンは 401 になり、クライアントの取り直しも IdMagic が拒否する |

拒否と停止のどの段階でも、MCP サーバーの実行の記録は増えない。
403 には `insufficient_scope` を付けないので、SDK は認可をやり直さずにエラーを返す。
拒否はスコープではなく割当、条件、関係で決まり、トークンを取り直しても結果が変わらないためである。

## 構成

| 構成要素 | このディレクトリのファイル | 役割 |
| --- | --- | --- |
| MCP サーバー | [`server.ts`](server.ts) | 文書を読み書きする 4 つのツールを持つ。認可のコードを持たない |
| Agent | [`agent.ts`](agent.ts) | `@modelcontextprotocol/client` の `PrivateKeyJwtProvider` だけで接続する |
| 鍵の作成 | [`keygen.ts`](keygen.ts) | Agent の鍵の組を作る |
| 再現試験 | [`walkthrough.ts`](walkthrough.ts) | 下の手順を、IdMagic とゲートウェイの実行ファイルと公開の管理 API で順に行う |

ゲートウェイと IdMagic は `https://` の URL を求める。
本番では利用者のリバースプロキシが TLS を終端する。
再現試験は自己署名の証明書で TLS の前段を自分で置く。

## 手順

前提として、`https://` で公開した IdMagic のテナントと、管理画面に入れる管理者が必要である。
リポジトリの開発環境（`./dev.sh`）は `http://` の発行者で動くので、そのままでは使えない。
手元で一通り試すだけなら、TLS の前段まで用意する[自動の再現](#自動の再現)を使う。
ここでは次の値を使う。

| 値 | 例 |
| --- | --- |
| IdMagic の realm の URL | `https://idp.example/realms/default` |
| ゲートウェイの公開 URL | `https://mcp.example/mcp` |
| MCP サーバーの URL（ゲートウェイからだけ届く） | `http://127.0.0.1:3001/` |

依存を入れる。

```sh
cd examples/mcp-client-walkthrough
bun install --frozen-lockfile
```

### 1. MCP サーバーを起動する

```sh
PORT=3001 bun run server.ts
```

ツールが実行されるたびに `executed:` の行を書くので、拒否した呼び出しが届いていないことをこの出力で確かめられる。

### 2. 接続先を登録し、ゲートウェイを起動する

管理画面の「MCP リソースサーバー」で、ゲートウェイの公開 URL を接続先として登録する。
評価用のクライアントを同時に作ると、クライアント ID とシークレットが一度だけ表示される。
シークレットはファイルに置き、ゲートウェイを動かすアカウントだけが読めるようにする。

```yaml
# gateway.yaml
listen: 127.0.0.1:8090
resource: https://mcp.example/mcp
upstream: http://127.0.0.1:3001/
idmagic:
  realm_url: https://idp.example/realms/default
  client_id: "<評価用のクライアント ID>"
  client_secret_file: /etc/idmagic-mcp-gateway/secret
mapping:
  catalog: true
```

```sh
idmagic-mcp-gateway --config gateway.yaml
```

設定の各項目は [MCP Gateway の実行環境](../../docs/modules/mcp-gateway/features/runtime.md#設定)が定める。
`https://mcp.example/mcp` から `127.0.0.1:8090` への転送は、TLS を終端するリバースプロキシで行う。
起動したゲートウェイは MCP サーバーのツールの一覧を IdMagic へ報告する。

### 3. ツールを公開し、条件と承認を設定する

接続先の編集で、報告された 4 つのツールを「公開する」。
次に「条件や上限を JSON で編集する」を開く。
欄には公開したツールごとの規則が配列で入っているので、`tool.name` が `write_document` と `publish_document` の要素に、それぞれ次のキーを足して更新する。

```json
{ "condition": { "deny_when": "token.principal_type == \"agent\"" } }
```

```json
{ "requires_approved_operation": true, "authorization_detail_type": "document_publication" }
```

一つ目の `deny_when` の式は、Agent が自分の資格で呼んだ更新を拒否する。
条件の書き方は[条件の評価](../../docs/modules/oauth2/features/resource-access.md#条件の評価)を参照する。

### 4. Agent を登録する

鍵の組を作る。

```sh
bun run keygen.ts agent-private.jwk.json > agent-public.jwk.json
```

管理画面の「エージェント」で Agent を登録し、「公開鍵からクレデンシャルを作成」に `agent-public.jwk.json` の内容を貼り付ける。
作成されたクライアント ID を控える。
「利用する接続先」で、`read_document`、`write_document`、`publish_document` を割り当てる。
`delete_document` は割り当てない。

### 5. クライアントで呼ぶ

```sh
export MCP_URL=https://mcp.example/mcp
export IDMAGIC_REALM=https://idp.example/realms/default
export AGENT_CLIENT_ID=<4 で控えたクライアント ID>
export AGENT_PRIVATE_JWK=agent-private.jwk.json

bun run agent.ts                                # ツールの一覧
bun run agent.ts read_document handbook         # 許可
bun run agent.ts delete_document handbook       # 割当のない操作。403
bun run agent.ts write_document handbook        # 条件の拒否。403
bun run agent.ts publish_document handbook      # 承認が必要な操作。503
```

### 6. 撤回と停止

「利用する接続先」で `read_document` の割当を外してから `read_document` を呼ぶと 403 になる。
割当を戻すと再び実行される。
エージェントの操作メニューで「無効化」を選ぶと、次の呼び出しは 401 になり、トークンの取り直しも拒否される。

各判断は、管理画面の「監査イベント」に `ResourceAccessEvaluated` として残る。
ゲートウェイのログとは、評価の `X-Request-ID` で突き合わせる。

## 自動の再現

```sh
mise run test-mcp-client-walkthrough
```

タスクは IdMagic とゲートウェイを現在のソースからビルドし、メモリの永続化と開発用のシードで IdMagic を起動する。
管理者として認可コードフローでログインし、管理画面と同じ管理 API で手順 2 から 6 を行い、各段階の結果と MCP サーバーへの到達を確かめる。
localhost で待ち受けるので、サンドボックスのある実行環境では外で実行する。

## 確認した範囲

確認した版は次のとおりである。
版は [`package.json`](package.json) と [`bun.lock`](bun.lock) で固定し、依存の更新のときにこの試験で確かめ直す。

| 対象 | 版 |
| --- | --- |
| `@modelcontextprotocol/client` | 2.3.1（2026-07-28 版に固定） |
| `@modelcontextprotocol/server` | 2.3.1 |
| Bun | 1.4.4 |

次は確かめていない。

| 範囲 | 理由または担当 |
| --- | --- |
| 承認を経た操作の実行 | ゲートウェイが承認した操作を扱えず、安全側で止まる。wi-83492 が扱う |
| 利用者から Agent への委譲 | SDK のクライアントは Agent 自身の `client_credentials` だけを行う |
| DPoP で束縛したトークン | この手順は Bearer のトークンだけを使う |
| Envoy の背後のゲートウェイ | `mise run test-envoy-mcp-gateway` が Go SDK で確かめる |
| ほかの言語の SDK と、MCP クライアントの製品 | 未確認 |

## 期待する結果にならない場合

| 症状 | 確認すること |
| --- | --- |
| トークンの要求が `invalid_client` になる | `client_id` を省いたクライアントアサーションを受け入れない古い IdMagic である。REQ-OAUTH2-028 に対応した版を使う。Agent の公開鍵とクライアント ID の組も確かめる |
| クライアントが保護リソースのメタデータを拒否する | `MCP_URL` と、接続先の URI とゲートウェイの `resource` が末尾まで一致するか |
| ゲートウェイが起動しない | `resource` と `realm_url` が `https://` か。設定の誤りでは起動せず、理由を標準エラーに書く |
| ツールが公開の候補に出ない | ゲートウェイが MCP サーバーへ届くか。接続先の編集で報告の時刻を確かめる |
| 割り当てたツールが 403 になる | 手順 4 の割当と、手順 3 の条件 |
