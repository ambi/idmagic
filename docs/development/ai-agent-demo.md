# AI エージェントの認証と認可のデモ

ローカルの IdMagic にエージェントを登録し、秘密鍵による認証、文書単位の認可、人の承認、停止後の失効を試す。
エージェントの HTTP 呼び出しをターミナルから再現するため、LLM の契約や外部の MCP サーバーは不要である。
実際の MCP ツールの実行まで確認する場合は、末尾の追加検証へ進む。

## デモの構成

| 登場するもの | このデモでの役割 |
| --- | --- |
| `alice` | エージェントの所有者、管理者、承認者 |
| Agent | 所有者、種別、状態を管理する非人間の ID |
| Agent に束縛する OAuth クライアント | 公開鍵を登録し、エージェントが秘密鍵の所有を証明するための資格情報 |
| 評価用の OAuth クライアント | 文書サービスの立場で、提示トークンによるアクセスの可否を問い合わせる |
| `document:d1` と `document:d2` | 閲覧を許可する文書と、許可しない文書 |

**認証**では秘密鍵の所有を確かめ、**認可**ではトークンのスコープと文書に対する関係を確かめる。
**ID 管理**では所有者、資格情報との対応、状態、利用状況を確認する。
Agent にロールを書くだけでは、このデモの文書へのアクセスは許可されない。

## 1. ローカル環境と管理者の準備

リポジトリのルートで、[ローカル開発](local-development.md#環境の準備)に従って依存を導入する。
以後のコマンドも同じルートで実行する。

```zsh
mise tasks
mise install
mise run setup
mise run dev
```

ブラウザーで <http://localhost:5173/> を開き、ローカルデモ認証で `alice` / `demo-password-1234` を使ってログインする。
テナントは `default` を使う。
このスタックのデータは使い捨てであり、停止して起動し直すと初期状態になる。
以下の認可モデルも、このローカル環境に作成する。

管理画面の「設定」→「API アクセストークン」で、説明を `AI エージェントのデモ` とし、次のスコープを選んでトークンを発行する。

- `agents:read`
- `agents:write`
- `oauth-clients:write`
- `mcp-resource-servers:write`

表示されたトークンをコピーする。
これはデモの構成を作る管理者用トークンであり、Agent に渡す資格情報は後で作る秘密鍵である。

別のターミナルで macOS 標準の zsh を使い、以後のコマンドを同じセッションへ順に貼り付ける。
管理者トークンは入力待ちになってから貼り付ける。

```zsh
BASE='http://localhost:5173/realms/default'
RESOURCE='https://mcp.agent-demo.example/'
read -r -s 'ADMIN_TOKEN?管理者の API アクセストークン: '
printf '\n'
umask 077
DEMO_DIR=$(mktemp -d "${TMPDIR:-/tmp}/idmagic-agent-demo.XXXXXX")

admin_json() {
  local method=$1 api_path=$2 response
  shift 2
  if response=$(curl --fail-with-body -sS -X "$method" "$BASE/api/admin/v1$api_path" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H 'Content-Type: application/json' "$@"); then
    printf '%s\n' "$response"
  else
    printf '%s\n' "$response" >&2
    return 1
  fi
}

b64url() {
  openssl base64 -A | tr '+/' '-_' | tr -d '='
}
```

`curl`、`jq`、`openssl`、`xxd` を使う。
macOS 標準の `curl`、`openssl`（LibreSSL）、`xxd` と、開発環境で用意する `jq` で実行できる。
古い macOS で `curl` が `--fail-with-body` を受け付けない場合は、`--fail` に置き換える（HTTP エラーの本文は表示されなくなる）。

## 2. エージェントの ID と資格情報の登録

秘密鍵をローカルに作り、公開鍵を JWKS に変換する。
秘密鍵はこのターミナル側に置き、IdMagic には公開鍵だけを登録する。

```zsh
openssl genrsa -out "$DEMO_DIR/agent.pem" 2048
RSA_N=$(openssl rsa -in "$DEMO_DIR/agent.pem" -noout -modulus | cut -d= -f2 | xxd -r -p | b64url)
jq -n --arg n "$RSA_N" '{keys:[{kty:"RSA",kid:"agent-demo-key",use:"sig",alg:"PS256",n:$n,e:"AQAB"}]}' > "$DEMO_DIR/jwks.json"

CLIENT_RESPONSE=$(jq -n --slurpfile jwks "$DEMO_DIR/jwks.json" '{
  client_name:"文書閲覧エージェントのデモ",
  client_type:"confidential",
  redirect_uris:[],
  grant_types:["client_credentials","urn:openid:params:grant-type:ciba"],
  token_endpoint_auth_method:"private_key_jwt",
  scope:"openid profile tools:read",
  jwks:$jwks[0],
  fapi_profile:"none"
}' | admin_json POST /clients --data-binary @-)
printf '%s\n' "$CLIENT_RESPONSE" | jq .
AGENT_CLIENT_ID=$(printf '%s' "$CLIENT_RESPONSE" | jq -er '.client.client_id')

AGENT_RESPONSE=$(admin_json POST /agents --data '{
  "name":"文書閲覧エージェントのデモ",
  "description":"d1 の閲覧を試すローカルデモ",
  "kind":"autonomous",
  "roles":[]
}')
printf '%s\n' "$AGENT_RESPONSE" | jq .
AGENT_ID=$(printf '%s' "$AGENT_RESPONSE" | jq -er '.id')

jq -n --arg id "$AGENT_CLIENT_ID" '{client_id:$id}' \
  | admin_json POST "/agents/$AGENT_ID/credentials" --data-binary @-
```

管理画面の「エージェント」を開き、作成した Agent の所有者が `alice`、種別が「自律」、状態が「有効」であり、資格情報に作成した `client_id` が載っていることを確認する。
所有者を省略したので、管理者トークンを発行した `alice` が所有者になる。
後の関係タプルで使う ID は `AGENT_CLIENT_ID` ではなく `AGENT_ID` である。

Agent に束縛できる認証方式は `private_key_jwt` または `tls_client_auth` に限られる。
初期データの Demo Client は共有シークレット方式なので、この束縛には使わない。

## 3. 秘密鍵による認証

要求ごとに署名した短命の **クライアントアサーション**を作る。
同じアサーションの再利用は拒否されるため、関数を呼ぶたびに新しい `jti` を作る。
署名方式は、この実装が受け付ける `PS256` を使う。

```zsh
client_assertion() {
  local now header payload input signature
  now=$(date +%s)
  header=$(printf '%s' '{"alg":"PS256","typ":"JWT","kid":"agent-demo-key"}' | b64url)
  payload=$(jq -cn --arg id "$AGENT_CLIENT_ID" --arg aud "$BASE/token" \
    --arg jti "$(openssl rand -hex 16)" --argjson now "$now" \
    '{iss:$id,sub:$id,aud:$aud,iat:$now,exp:($now+60),jti:$jti}' | b64url)
  input="$header.$payload"
  signature=$(printf '%s' "$input" | openssl dgst -sha256 \
    -sign "$DEMO_DIR/agent.pem" \
    -sigopt rsa_padding_mode:pss -sigopt rsa_pss_saltlen:32 | b64url)
  printf '%s.%s' "$input" "$signature"
}

agent_post() {
  local endpoint=$1
  shift
  curl -sS "$BASE/$endpoint" \
    --data-urlencode "client_id=$AGENT_CLIENT_ID" \
    --data-urlencode 'client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer' \
    --data-urlencode "client_assertion=$(client_assertion)" "$@"
}

issue_agent_token() {
  agent_post token --data-urlencode 'grant_type=client_credentials' \
    --data-urlencode 'scope=tools:read' --data-urlencode "resource=$RESOURCE"
}
```

まず、署名を付けずに要求する。
HTTP 401 と `invalid_client` になり、トークンが返らないことを確認する。

```zsh
curl -sS -w '\nHTTP %{http_code}\n' "$BASE/token" \
  --data-urlencode "client_id=$AGENT_CLIENT_ID" \
  --data-urlencode 'grant_type=client_credentials' \
  --data-urlencode 'scope=tools:read'
```

正常な発行は、宛先のリソースサーバーを登録してから次の節で実行する。

## 4. 文書単位の認可

### リソースサーバーの登録

文書サービスが評価を問い合わせるためのクライアントと、評価する操作を登録する。
このクライアントは Agent には束縛しない。
このデモでは HTTP の評価 API を直接呼ぶので、`RESOURCE` のホストでサーバーを立てる必要はない。

```zsh
RS_RESPONSE=$(admin_json POST /clients --data '{
  "client_name":"文書サービスの評価用デモ",
  "client_type":"confidential",
  "redirect_uris":[],
  "grant_types":["client_credentials"],
  "token_endpoint_auth_method":"client_secret_basic",
  "scope":"tools:read",
  "fapi_profile":"none"
}')
RS_CLIENT_ID=$(printf '%s' "$RS_RESPONSE" | jq -er '.client.client_id')
RS_CLIENT_SECRET=$(printf '%s' "$RS_RESPONSE" | jq -er '.client_secret')

jq -n --arg resource "$RESOURCE" --arg client "$RS_CLIENT_ID" '{
  resource:$resource,name:"文書サービスのデモ",state:"Active",
  scopes:["tools:read"],client_id:$client,
  access_rules:[{resource_type:"document",relation:"viewer",scopes:["tools:read"]}]
}' | admin_json POST /mcp-resource-servers --data-binary @- | jq .

TOKEN_RESPONSE=$(issue_agent_token)
printf '%s\n' "$TOKEN_RESPONSE" | jq '{token_type,expires_in,scope,error}'
AGENT_TOKEN=$(printf '%s' "$TOKEN_RESPONSE" | jq -er '.access_token')

introspect_agent_token() {
  curl -sS -u "$RS_CLIENT_ID:$RS_CLIENT_SECRET" "$BASE/introspect" \
    --data-urlencode "token=$AGENT_TOKEN"
}
introspect_agent_token | jq .
```

発行が成功し、イントロスペクションが `active:true` を返すことを確認する。
これで鍵による認証とトークン取得を確認できた。
文書へのアクセスは、次の関係を登録して初めて許可される。

### 認可モデルと関係の登録

ブラウザーで「細粒度認可 (ReBAC)」→「認可モデル」→「認可モデルを編集」を開く。
編集の形式を「JSON」に切り替え、`resource_types` の入力欄へ次の配列を貼る。
「発行内容を確認」→「発行」で反映する。

```json
[
  {"name":"user","relations":[]},
  {"name":"agent","relations":[]},
  {
    "name":"document",
    "relations":[{
      "name":"viewer",
      "rewrites":[{"kind":"direct","direct_subject_types":["user","agent"]}]
    }]
  }
]
```

続いて「関係タプル」→「関係タプルを追加」で、次の関係を一つ登録する。
`printf '%s\n' "$AGENT_ID"` で主体 ID を確認できる。
これらの設定は管理者の対話セッションから行う操作であり、管理者用 API アクセストークンでは代用できない。

| 入力欄 | 値 |
| --- | --- |
| リソース型 | `document` |
| リソース ID | `d1` |
| 関係 | `viewer` |
| 主体の形 | `agent` の個別の主体 |
| 主体 ID | `$AGENT_ID` の値 |

### 許可と拒否の比較

次の関数は、文書サービスの資格情報で評価を依頼し、Agent のトークンを本文で提示する。
`decision:true` の場合にだけ文書を返すことが、文書サービス側の役割である。
このデモでは、その判断結果を表示する。

```zsh
evaluate_document() {
  curl -sS -u "$RS_CLIENT_ID:$RS_CLIENT_SECRET" "$BASE/resource-access/evaluate" \
    --data-urlencode "token=$AGENT_TOKEN" \
    --data-urlencode 'resource_type=document' \
    --data-urlencode "resource_id=$1" \
    --data-urlencode 'relation=viewer'
}
evaluate_document d1 | jq .
evaluate_document d2 | jq .
```

`d1` は `decision:true`、関係を登録していない `d2` は `decision:false` になることを確認する。
どちらも評価 API 自体は HTTP 200 を返し得るので、HTTP の成功だけで許可と判断しない。

ブラウザーで `d1` の関係タプルを削除し、同じ `AGENT_TOKEN` のまま `evaluate_document d1` を再実行する。
`decision:false` に変わることを確認し、同じ関係タプルを追加し直して `decision:true` に戻ることを確認する。
トークンを取り直さずに、文書ごとの権限変更を確認できる。

## 5. 発行のガードレール

Agent の詳細画面から「ガードレールを編集」へ進み、次の項目を設定して保存する。
それ以外の項目は空欄のままにする。
保存に成功すると詳細へ戻るので、更新した値と成功の通知を確認する。

| 項目 | 値 |
| --- | --- |
| 発行の回数の上限 | `1` |
| 回数を数える窓（秒） | `3600` |
| トークンの有効期間の上限（秒） | `60` |
| 超えたときの扱い | 拒否 |

```zsh
issue_agent_token | jq '{expires_in,error}'
issue_agent_token | jq .
```

保存後の最初の発行が成功し、`expires_in` が `60` になることを確認する。
次の発行は `access_denied` になり、トークンが返らないことを確認する。
ガードレールの保存より前の発行は、このポリシーの計数には含まれない。

確認後はもう一度「ガードレールを編集」へ進み、「ガードレールを削除」を選んで確認し、この Agent のポリシーを削除する。
ガードレールの変更は次の発行から効くもので、発行済みトークンの失効は後の無効化で確認する。

## 6. 人の承認が必要なエージェント

Agent の編集画面で種別を「監督下 (supervised)」へ変えて保存する。

```zsh
issue_agent_token | jq .
```

`unauthorized_client` になり、自律的な `client_credentials` では新しいトークンを得られないことを確認する。
次に、同じ鍵と同じクライアントで `alice` に承認を依頼する。

```zsh
APPROVAL_RESPONSE=$(agent_post bc-authorize \
  --data-urlencode 'scope=openid profile' \
  --data-urlencode 'login_hint=alice' \
  --data-urlencode 'binding_message=文書閲覧エージェントの承認デモ')
printf '%s\n' "$APPROVAL_RESPONSE" | jq .
AUTH_REQ_ID=$(printf '%s' "$APPROVAL_RESPONSE" | jq -er '.auth_req_id')
POLL_INTERVAL=$(printf '%s' "$APPROVAL_RESPONSE" | jq -er '.interval')

poll_approval() {
  agent_post token --data-urlencode 'grant_type=urn:openid:params:grant-type:ciba' \
    --data-urlencode "auth_req_id=$AUTH_REQ_ID"
}
poll_approval | jq .
```

承認前のポーリングが `authorization_pending` になることを確認する。
ブラウザーで <http://localhost:5173/realms/default/account/approvals> を開き、`alice` として該当する要求を承認する。
ステップアップ認証を求められたら `demo-password-1234` を入力する。
通知メールが届かなくても、この一覧から操作できる。

```zsh
sleep "$((POLL_INTERVAL + 1))"
poll_approval | jq '{token_type,expires_in,scope,access_token,id_token,error}'
```

承認後にアクセストークンと ID トークンが返ることを確認する。
同じ `auth_req_id` をもう一度交換すると `invalid_grant` になり、一回限りの交換であることを確認できる。
拒否する場合も試すには、新しい要求を起票し直し、ブラウザーで拒否してからポーリングする。
この場合は `access_denied` になる。

ここで確認したのは、人の承認を経たトークン発行である。
送金などの業務操作を一回に制限する場合は、承認明細との照合と業務側の重複実行の防止も必要になる。
その契約は[承認した操作の評価](../modules/oauth2/features/resource-access.md#承認した操作の評価)を参照する。

## 7. 無効化と緊急停止

Agent の種別を「自律 (autonomous)」へ戻して保存し、停止直前のトークンを取得する。

```zsh
AGENT_TOKEN=$(issue_agent_token | jq -er '.access_token')
introspect_agent_token | jq '{active}'
evaluate_document d1 | jq .
```

`active:true` と `decision:true` を確認してから、Agent の操作メニューで「無効化」を選ぶ。
トークンを取り直さずに、次を実行する。

```zsh
introspect_agent_token | jq '{active}'
evaluate_document d1 | jq .
issue_agent_token | jq .
```

発行済みトークンが `active:false`、アクセス判定が `decision:false` になり、新規発行も拒否されることを確認する。
停止の直前に通った同じトークンを使うことで、期限切れと停止の効果を区別できる。

「有効化」で戻し、新しいトークンを取得すると再びアクセスできることを確認する。

```zsh
AGENT_TOKEN=$(issue_agent_token | jq -er '.access_token')
evaluate_document d1 | jq .
```

最後に「緊急停止 (kill)」を確定し、同じ停止後の三つの呼び出しを繰り返す。
発行済みトークンと新規発行が拒否され、Agent は「緊急停止」になる。
緊急停止は取り消せず、再有効化も Agent の削除もできないため、この操作はデモの最後に行う。

## 8. 棚卸しと監査

管理画面の「エージェント」→「棚卸し」で、所有者の状態、資格情報の束縛、最終のトークン発行時刻を確認する。
未束縛の所見も試す場合は、別の Agent を資格情報なしで登録し、`unbound` の対象になることを確認する。
新しく作った Agent は、既定の 90 日の休眠判定には該当しない。

「監査イベント」で、次の操作に対応するイベントを確認する。
ガードレールやアクセス判定の詳細は、該当イベントの属性を見る。

| 観察したこと | イベントの例 |
| --- | --- |
| ID と資格情報の対応 | `AgentRegistered`、`AgentCredentialBound` |
| トークンの発行 | `AccessTokenIssued` |
| 発行の上限 | `AgentGuardrailEvaluated`、`AgentGuardrailPolicyChanged` |
| 人による承認 | `BackchannelAuthRequested`、`BackchannelAuthApproved` |
| 文書へのアクセス判定 | `ResourceAccessEvaluated` |
| 停止と再開 | `AgentDisabled`、`AgentEnabled`、`AgentKilled` |

## 終了と追加検証

管理画面でデモ用の管理者 API アクセストークンを失効させる。
`mise run dev` を起動したターミナルで `Ctrl+C` を押し、ローカルスタックを停止する。
秘密鍵を含む一時ディレクトリと、zsh に置いた値と関数を削除する。

```zsh
rm -- "$DEMO_DIR/agent.pem" "$DEMO_DIR/jwks.json"
rmdir -- "$DEMO_DIR"
unset ADMIN_TOKEN AGENT_TOKEN RS_CLIENT_SECRET CLIENT_RESPONSE RS_RESPONSE TOKEN_RESPONSE APPROVAL_RESPONSE
unfunction admin_json b64url client_assertion agent_post issue_agent_token introspect_agent_token evaluate_document poll_approval
```

実際の MCP ツールへの転送と業務作用の抑止は、既存の HTTP 統合テストでも確認できる。
テストはデモ画面とは別の隔離された環境を使う。

```zsh
mise run test-go-one -- ./backend/shared/http/server_http TestMcpGatewayRunsOnlyPermittedToolCalls
mise run test-go-one -- ./backend/shared/http/server_http TestExternalPEPStopsBusinessEffectAfterAgentKill
```

手元の MCP サーバーへつなぐ場合のゲートウェイ設定は、[MCP Gateway の実行環境](../modules/mcp-gateway/features/runtime.md)と[Envoy の構成例](../../examples/envoy-mcp-gateway/README.md)を参照する。
利用者の代理で動く委譲は、[トークン交換](../modules/oauth2/features/token.md)で主体と行為者の扱いを確認してから試す。

### 期待する結果にならない場合

| 応答や症状 | 確認すること |
| --- | --- |
| 管理 API の 403 | 管理者トークンのスコープとテナント。認可モデルと関係タプルはブラウザーの対話セッションで設定する |
| 管理 API の 400 | 標準エラーへ出した JSON の `detail` を確認する。CIBA が未知のグラントとして拒否される場合は、登録検証の修正前の API が動いているため、開発スタックを再起動し、管理者トークンの発行からやり直す |
| `invalid_client` | 登録した公開鍵、`client_id`、署名の `PS256`、`aud` の URL、時計のずれ、アサーションの再利用、Agent と所有者の状態 |
| `invalid_target` | `resource` と登録した `RESOURCE` が末尾の `/` まで一致するか、リソースサーバーが `Active` か |
| `decision:false` | `reasons`、トークンの期限、文書 ID、関係タプルの主体 ID に `AGENT_ID` を使ったか |
| `access_denied` | ガードレールと監査。発行回数のデモ後にポリシーを削除したか |
| `unauthorized_client` | `supervised` のまま自律的に発行しようとしていないか、必要なグラントを登録したか |
| `slow_down` | 応答の `interval` を守り、待ってから新しいアサーションでポーリングする |
| 再起動後に ID が見つからない | 開発スタックは使い捨てなので、管理者トークンの発行からやり直す |
