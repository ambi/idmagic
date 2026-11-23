# Agent の ID と認証

この文書は、Agent の登録、資格情報の紐付け、自律実行のトークン取得、ワークロードの信頼設定を説明する。
利用者の代理実行は[委任とトークン](ai-agent-demo-delegation.md)、操作の許可は[認可と人の承認](ai-agent-demo-authorization.md)で設定する。
[ガイドの入口](ai-agent-demo.md)に機能の全体像がある。

## 登録前の準備

ローカルでは `mise run dev-memory` を起動し、<http://localhost:5173/realms/default/admin> に管理者でログインする。
画面で動く完成例を使う場合は、代わりに `mise run demo-agent` と[デモの設定手順](ai-agent-demo-walkthrough.md#2-設定)を使う。

| 決めるもの | 選択の目安 |
| --- | --- |
| 所有者 | 停止や棚卸しの責任を持つ、同じテナントの有効な利用者。省略時は登録する管理者になる |
| 種別 | 通常の自律実行は「自律」。毎回人の承認を経た発行に限定するなら「監督下」 |
| 認証方式 | Agent が保持する鍵は `private_key_jwt`、証明書は `tls_client_auth`、SPIFFE の JWT-SVID は `spiffe_jwt` |
| 接続先と操作 | 先に MCP リソースサーバーと公開するツールを用意すると、Agent の登録時に選択できる |

Agent にロールを付けて権限をまとめる方式ではなく、操作と対象の割当を使う。
同じ Agent に複数の認証用クライアントを紐付けられるが、一つのクライアントを複数の Agent へ紐付けることはできない。

## 自律エージェントの公開鍵認証

### 管理画面の設定

1. 「エージェント」から登録を開き、名前、種別「自律」、所有者を指定する。
2. 認証用クライアントの新規作成を選び、Agent 側で作った公開鍵を貼り付けるか、公開 JWKS の URL を指定する。
3. 「利用する MCP リソースサーバー」で、先に公開した操作を選ぶ。
4. 登録結果から Agent ID と認証用クライアント ID を控え、Agent 側の `client_id` には後者を設定する。

公開鍵は PEM、JWK、JWKS を受け付け、RSA または P-256 の鍵を使える。
秘密鍵は Agent 側に保管し、管理画面へ貼り付けない。
デモでは画面に表示された公開鍵を使うため、自分で鍵を用意する必要はない。

既存のクライアントを利用する場合は、対象を選んで紐付ける。
紐付けられるのは同じテナントの `private_key_jwt`、`tls_client_auth`、`spiffe_jwt` のクライアントであり、共有シークレットのクライアントは Agent の資格情報にできない。
「後で設定する」を選ぶと ID だけを登録し、認証や操作の許可は準備されない。
登録が途中で失敗した場合は画面の作成済み ID を保持し、失敗した段階から再試行する。

### トークン要求

公開メタデータ `<issuer>/.well-known/openid-configuration` の `token_endpoint` に、フォーム形式で次を送る。
表の値はクライアントが送信前に URL エンコードする。
以下の要求例は自分のクライアントを実装するためのものであり、実行済みの値は[デモ](ai-agent-demo-walkthrough.md)または [MCP SDK の自動検証](../../examples/mcp-client-walkthrough/README.md)で確認できる。

| フィールド | 値 |
| --- | --- |
| `grant_type` | `client_credentials` |
| `client_id` | 登録結果の認証用クライアント ID |
| `client_assertion_type` | `urn:ietf:params:oauth:client-assertion-type:jwt-bearer` |
| `client_assertion` | 登録した公開鍵に対応する秘密鍵で署名した新しい JWT |
| `resource` | 登録済み接続先のリソース URI |
| `scope` | 必要なスコープ。デモのカタログ方式では省略し、現在の操作割当とクライアントの上限から決定させる |

`private_key_jwt` のクライアントアサーションは、`iss` と `sub` をクライアント ID、`aud` をトークンエンドポイントにし、短い `exp`、現在の `iat`、要求ごとに異なる `jti` を持たせる。
デモは `PS256` で署名し、有効期間を 60 秒にする。
要求の組み立てと署名の実装は [`examples/agent-demo/main.go`](../../examples/agent-demo/main.go) の `assertion` と `authenticated` にある。

成功時は `access_token`、`token_type`、`expires_in`、許可された `scope` を受け取る。
このフローには refresh token がないため、期限が来たら新しいクライアントアサーションで発行を要求する。
トークンの `sub` はクライアント ID であり、紐付く Agent は `agent_id` などのクレームで示される。
`sub` を Agent ID とみなして独自に権限を検索しない。

### 許可と拒否の確認

1. 割り当てた操作を Gateway 経由で呼び、成功することを確かめる。
2. 同じトークンで未割当の操作を呼び、上流へ届かずに拒否されることを確かめる。
3. Agent を無効化し、同じトークンでの呼び出しと、新しい発行が止まることを確かめる。
4. 所有者が有効な状態で Agent を再有効化し、新しいトークンで再開する。

署名が通っても、Agent、所有者、クライアント、対象の状態、発行ガードレールの条件を満たさなければ発行されない。
「監督下」に変更した Agent は、この通常の発行では `unauthorized_client` になる。
その場合は [CIBA](ai-agent-demo-authorization.md#発行前の人の承認) を使う。

## mTLS と送信者の束縛

証明書で認証する場合は、OAuth クライアントの認証方式を `tls_client_auth` にし、許可する証明書の Subject DN を登録して Agent に紐付ける。
検証済みクライアント証明書を IdMagic へ渡す TLS 終端を構成し、その証明書でトークンを取得する。
発行したトークンは `cnf.x5t#S256` に証明書の拇印を持ち、リソースへ提示するときも同じ証明書が必要になる。
評価用クライアントが送る `client_certificate_thumbprint` は、リソース側が終端して検証した接続から取得する。

DPoP を使う場合は、クライアント認証に加えてトークン要求に DPoP Proof を付ける。
発行された `DPoP` トークンをリソースへ送るときは、同じ鍵で、実際の `htm`、`htu` とアクセストークンの `ath` を含む新しい Proof を作る。
`private_key_jwt` の署名だけではアクセストークンはその鍵に束縛されない。
クライアント認証の鍵と、トークンの所持を証明する鍵の役割を分けて確認する。

証明書の照合と送信者の束縛は[トークンの仕様](../modules/oauth2/features/token.md#リソースサーバーによるトークンの検証)に従う。
ローカルの HTTP 管理画面を起動しただけでは、mTLS の接続試験にはならない。

## ワークロードによる認証

CI やコンテナーに発行された外部 JWT、または SPIFFE の JWT-SVID を、登録済みの Agent に対応付ける。
次の二つの使い方があり、同じ JWT を無条件にどちらにも流用しない。

| 方式 | JWT の役割 | 要求の主な値 |
| --- | --- | --- |
| ワークロード証明の Token Exchange | 実行環境の証明を IdMagic の短命なアクセストークンへ交換する | `subject_token_type=urn:ietf:params:oauth:token-type:jwt` |
| `spiffe_jwt` のクライアント認証 | JWT-SVID で OAuth クライアントそのものを認証する | `client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-spiffe` |

### 信頼設定と関連付け

1. 「ワークロード ID」の信頼設定で、OIDC または SPIFFE のプロファイルを選ぶ。
2. OIDC では信頼する issuer、SPIFFE ではトラストドメインを指定し、JWKS の URL または鍵、受理する audience、証明の最大有効期間を登録する。
3. issuer のメタデータ取得や鍵の更新確認で、公開情報を取得できることを確かめる。
4. 信頼設定の関連付けに、外部 JWT の `sub` と照合するパターン、および対応先の Agent ID を登録する。
5. プレビューに実際の `sub` を入れ、意図した有効な関連付け一つだけに一致することを確認する。
6. 対応先の Agent に認証用クライアントを紐付け、操作と対象を割り当てる。

管理 API を使う場合、信頼設定の入口は `/api/admin/v1/workload-identity/trust-bundles`、配下の関連付けは `/{trust_bundle_id}/bindings` である。
プレビューは同じ配下の `/bindings/preview` を使う。
OIDC の信頼設定を作る JSON の例は次のとおりである。
URL と audience は実際の発行基盤に合わせ、`trust_domain` は管理用のラベルとして決める。

```json
{
  "name": "文書集計 CI",
  "profile": "oidc",
  "trust_domain": "documents-ci",
  "issuer": "https://workload.example.com",
  "jwks_uri": "https://workload.example.com/jwks",
  "accepted_audiences": ["idmagic-documents"],
  "max_subject_token_ttl_seconds": 600
}
```

作成結果の信頼設定に、`{"subject_pattern":"repo:example/doc-agent:job:nightly","agent_id":"<Agent ID>"}` の関連付けを POST する。
SPIFFE では `profile=spiffe`、`trust_domain` を実際のトラストドメインにし、JWT-SVID の検証に適した鍵を設定する。
`*` や `?` のパターンは一つのパス階層内で照合される。
一つのプレビューが成功しても、将来現れるすべての主体でパターンの重複がないことまでは証明しない。

### 証明をトークンに交換する手順

トークンエンドポイントへ、次のフォームを送る。
要求するクライアントの認証には、上で紐付けたクライアントのアサーションまたは証明書を別途添える。

| フィールド | 値 |
| --- | --- |
| `grant_type` | `urn:ietf:params:oauth:grant-type:token-exchange` |
| `subject_token_type` | `urn:ietf:params:oauth:token-type:jwt` |
| `subject_token` | 信頼設定に合う外部 JWT または JWT-SVID |
| `resource` | 登録した保護対象の URI |
| `scope` | クライアントと対象で許可された必要最小限のスコープ |

対応先の Agent と、認証したクライアントに紐付く Agent が一致すると交換できる。
成功時は `WorkloadTokenExchanged` とトークン発行の記録を確認する。
別の issuer、違う audience、期限切れ、不正な署名、関連付けなし、複数の関連付けへの一致では交換を拒否する。

### JWT-SVID によるクライアント認証

クライアントを `spiffe_jwt` にして対応先の Agent に紐付ける。
自律実行では、公開鍵認証の例の `client_credentials` 要求の認証部分を、`client_id`、上表の `jwt-spiffe` の型、新しい JWT-SVID を入れた `client_assertion` に置き換える。
この JWT-SVID の `aud` は IdMagic の issuer だけでなければならない。
`private_key_jwt` の例のように `/token` を付けたり、Token Exchange 向けの受理 audience に置き換えたりしない。
SPIFFE の鍵は `iss` から任意に探索せず、`sub` のトラストドメインに登録した信頼設定から選ばれる。

信頼設定や関連付けを無効化すると、その経路による以後の認証と交換が止まる。
発行済みトークンも止める必要がある場合は、Agent の無効化を併せて行う。
実際の CI や SPIFFE 基盤は別途用意し、発行した証明の `iss`、`sub`、`aud` と登録値を照合する。

## 確認の入口

公開鍵認証から Gateway の拒否までを再現するには `mise run test-mcp-client-walkthrough` を使う。
ワークロードの検証器は `mise run test-go-package -- ./backend/workloadidentity/internal/usecases` で確認できる。
これはテスト用の署名と信頼設定による検証であり、外部の CI や SPIFFE サーバーに接続する試験ではない。

契約の詳細は [Agent](../modules/identity-management/features/agent.md)、[トークン](../modules/oauth2/features/token.md)、[ワークロード証明の交換](../modules/workloadidentity/features/attestation-exchange.md)を参照する。
