# 利用者からの委任とトークン

この文書は、利用者の代理実行、複数 Agent の委任、アプリ間の ID-JAG、トークンの更新を説明する。
Agent の認証用クライアントは[ID と認証](ai-agent-demo-identity.md)で準備する。
[ガイドの入口](ai-agent-demo.md)に利用パターンの一覧がある。

## 利用者の代理実行

本人の権限を使う文書検索や外部 SaaS の操作では、利用者のアクセストークンを起点に Agent 用のトークンへ交換する。
アプリへの OAuth 同意と、Agent に対する本人の委任の両方を用意する。
管理者が Agent を所有していることや、操作を割り当てていることだけでは本人の委任にならない。

### 設定

1. 接続先の MCP リソースサーバーを登録し、利用するスコープと操作を準備する。
2. 利用者がログインするアプリに、リダイレクト URI、`authorization_code`、必要なスコープを登録する。
3. Agent の認証用クライアントに Token Exchange のグラントと必要なスコープを設定し、Agent に操作を割り当てる。
4. 本人がアカウント画面の「接続済みアプリ」を開き、代行を許す Agent、スコープ、有効期限を指定して付与する。
5. アプリは Authorization Code と PKCE のフローで本人をログインさせ、同意の後に利用者のアクセストークンを取得する。

ローカルの本人の画面は <http://localhost:5173/realms/default/account/applications> である。
Agent の付与は `(本人, Agent)` の組で保存され、OAuth クライアントへの同意とは別に表示される。
期限は省略時に 365 日、指定できる上限も 365 日である。
付与の作成は本人のブラウザーセッションで行い、API アクセストークンで Agent が自分への付与を作ることはできない。

ブラウザーの認可要求は `<issuer>/authorize` へ送り、次のパラメーターを指定する。
クライアントに PAR を必須としている場合は、先に認証付きで `/par` へ送り、返された `request_uri` で認可を開始する。

| パラメーター | 値 |
| --- | --- |
| `response_type` | `code` |
| `client_id`、`redirect_uri` | ログイン用アプリの登録値。Agent の交換用クライアントとは区別する |
| `scope` | `openid` と必要なスコープ |
| `state`、`nonce` | 要求ごとのランダム値。戻りの state と ID Token の nonce を検証する |
| `code_challenge`、`code_challenge_method` | 新しい verifier の SHA-256 を base64url にした値、`S256` |
| `resource` | アプリが要求する登録済みの対象。トークンを使う対象と一致させる |
| `authorization_details` | 必要な場合、登録済みの型で作った明細の JSON 文字列 |

戻りの `code` は、トークンエンドポイントへ `grant_type=authorization_code`、同じ `redirect_uri`、元の `code_verifier` とともに送る。
confidential クライアントはクライアント認証も付ける。
利用者にはアプリへの割当とサインインポリシーも適用され、追加認証を求められた場合はその認証を済ませる。
これらの関門は、Agent への本人の付与とは別に満たす必要がある。

自分の UI を実装する場合、付与の一覧と作成は `/api/account/v1/agent-delegations`、撤回は `/{agent_id}/revoke` を使う。
作成する JSON は次の形で、`agent_id` はクライアント ID ではない。
ブラウザーの同一オリジンのセッションと CSRF 対策を伴わせる。

```json
{
  "agent_id": "<Agent の登録結果の ID>",
  "scopes": ["mcp"]
}
```

### 交換と実行

Agent のクライアントで認証し、issuer の `token_endpoint` に次のフォームを送る。
`subject_token` は同じ認可サーバーで発行した利用者のアクセストークンであり、ID Token や外部の JWT には置き換えない。

| フィールド | 値 |
| --- | --- |
| `grant_type` | `urn:ietf:params:oauth:grant-type:token-exchange` |
| `subject_token_type` | `urn:ietf:params:oauth:token-type:access_token` |
| `subject_token` | 本人のアクセストークン |
| `resource` | 登録済みの接続先の URI 一つ |
| `scope` | 本人のトークン、委任、クライアント、接続先の範囲内のスコープ |
| `actor_token` | 別の操作者をトークンで示す構成でのみ、そのアクセストークンを指定 |
| `actor_token_type` | `actor_token` を使うなら `urn:ietf:params:oauth:token-type:access_token` |

この通常の交換では `audience` を送らない。
元のトークンに `may_act` がある場合は、その許可の範囲にも従う。
`authorization_details` を引き継ぐ場合も、元の内容を広げる要求は拒否される。

交換後のトークンでは `sub` に元の本人、`act` に操作者が残る。
認可済みのリソースサーバーからイントロスペクションし、`active=true`、対象、スコープ、`act`、`delegation_mode=on_behalf_of` を確認する。
Gateway や業務 API は、操作者を別の利用者として付け替えず、このトークンで毎回認可を評価する。

### 拒否と取消の確認

| 試すこと | 期待する結果 |
| --- | --- |
| 付与前に交換する | `invalid_grant` で発行されない |
| 付与より広いスコープを要求する | 発行されない。本人の付与と元のトークンの両方の範囲内に直す |
| 付与の期限を過ぎて交換する | 発行されない |
| 本人が付与を撤回し、既発行の代行トークンを使う | オンライン評価で止まり、イントロスペクションは `active=false` になる |
| 別の利用者が同じ Agent へ委任している | 一人の撤回だけで他の利用者の委任は取り消されない |
| 撤回後に同じ Agent へ再付与する | 新しい発行を行う。以前の代行トークンは復活しない |

本人は「接続済みアプリ」から Agent の付与を撤回できる。
Agent 全体を止めると、他の利用者の委任にも影響するため、個人の取消と使い分ける。

## 端末からの利用者の認可

CLI や入力の制約がある端末では、Device Authorization Grant で利用者のトークンを取得できる。
Agent 自体の公開鍵認証とは分けて、利用者がログインするためのクライアントにこのグラントと必要なスコープを登録する。

1. 公開メタデータの `device_authorization_endpoint` へ、`client_id` と必要な `scope` をフォームで送る。
2. 応答の `verification_uri` と `user_code`、または `verification_uri_complete` を本人へ表示する。
3. 本人は別のブラウザーで URL を開き、ログインしてコードとアプリを確認し、承認する。
4. 端末は応答の `interval` を守り、トークンエンドポイントへ `grant_type=urn:ietf:params:oauth:grant-type:device_code`、`device_code`、`client_id` を送り、クライアントの方式に合う認証を行う。
5. 取得した本人のアクセストークンを、前節の Token Exchange の起点にする。本人の Agent への付与も別途必要である。

承認前は `authorization_pending`、間隔が短ければ `slow_down`、拒否時は `access_denied`、期限切れは `expired_token` になる。
成功済みのコードを使い回さない。
この端末のログイン承認だけでは、指定した業務操作を一回承認したことにはならない。
契約は[デバイス認可](../modules/oauth2/features/device.md)が定める。

## 複数エージェントの委任

たとえば受付 Agent から文書検索 Agent へ引き継ぐときも、利用者の `sub` を維持し、操作者を `act` の入れ子として記録する。
単に元のトークンを次の Agent に配る方式にせず、次のクライアントで認証して Token Exchange を行う。

1. 両方の Agent と認証用クライアントを登録し、各段階で必要な操作だけを割り当てる。
2. 利用者が、関与するすべての Agent に必要なスコープを付与する。
3. 次の Agent は受け取った代行トークンを `subject_token` にして、接続先と狭めたスコープを指定する。
4. 結果の `sub`、入れ子の `act`、監査イベント `TokenExchanged` の深さと上限を確認する。
5. 一つの Agent の付与を撤回し、その Agent が発行に関わる連鎖のトークンも止まることを確認する。

本人の付与は、元のトークン、操作者のトークン、要求クライアントに紐付く Agent のすべてで検査する。
一つでも不足すれば発行されない。
テナントの委任の深さのデフォルト上限は 3 であり、Agent のガードレールでさらに狭められる。
スコープや明細を段階的に狭めても、この深さの上限を超える交換は許されない。

DPoP や mTLS に束縛されたトークンを別の Agent に渡す場合、そのトークンの鍵の証明も必要になる。
鍵を共有しない上流接続には、認可評価が発行する一回限りの交換許可を使う[Gateway の上流 Token Exchange](ai-agent-demo-connections.md#上流が-idmagic-のトークンを受け入れる構成)を選ぶ。

## アプリをまたぐアクセス

ID-JAG は、認証元の ID Token から、別の認可サーバーでアクセストークンを発行するための署名済みアサーションを作る機能である。
同じ issuer 内での Agent 間の Token Exchange とは用途も要求形式も異なる。
IdMagic は自身と同じ issuer が発行した ID-JAG を受け入れないため、試すときは異なる二つの realm または認可サーバーを用意する。

### 二つの認可サーバーの準備

1. 発行元に ID-JAG を要求する confidential クライアントと、受け入れ先を SSO の RP として表すクライアントを登録する。受け入れ先にも ID-JAG を引き換える confidential クライアントを登録し、三つの ID を区別して控える。
2. 受け入れ先で発行元への有効な OIDC 接続を登録し、本人の連携ログインなどで、外部の `sub` を受け入れ先の有効な利用者へ解決できる連携 ID を用意する。発行元では本人が RP のアプリへアクセスできるように割り当てる。
3. 受け入れ先で対象リソースとスコープを登録する。
4. 発行元で、要求するクライアント、受け入れ先 issuer、受け入れ先クライアント ID、リソース、スコープ、許可する RAR の型を発行ポリシーに登録する。
5. 受け入れ先で、OIDC 接続と受け入れ先クライアントの組に、スコープの受け入れポリシーを登録する。
6. クライアントに Agent が紐付く場合は、それぞれで本人の付与と Agent の発行条件も満たす。

ポリシーの管理 API は `/api/admin/v1/cross-app-access/issue-policies` と `/api/admin/v1/cross-app-access/redeem-policies` である。
登録内容は、発行元の許可と受け入れ先の許可の両方で絞る。
発行元に登録する JSON は次の形である。

```json
{
  "client_id": "<発行元で ID-JAG を要求するクライアント ID>",
  "audience": "https://resource-id.example.com/realms/default",
  "relying_party_client_id": "<発行元で受け入れ先を表す RP のクライアント ID>",
  "target_client_id": "<受け入れ先で引き換えるクライアント ID>",
  "resources": ["https://mcp.example.com/"],
  "scopes": ["mcp"]
}
```

受け入れ先には次を登録する。
`provider_id` は OIDC 接続の ID であり、issuer 文字列や利用者 ID ではない。

```json
{
  "provider_id": "<発行元への OIDC 接続 ID>",
  "client_id": "<受け入れ先で引き換えるクライアント ID>",
  "scopes": ["mcp"]
}
```

### 発行と受け入れ

発行元で本人をログインさせ、そのクライアント向けの ID Token を取得する。
発行元のトークンエンドポイントでクライアント認証を行い、次のフォームを送る。

| フィールド | 値 |
| --- | --- |
| `grant_type` | `urn:ietf:params:oauth:grant-type:token-exchange` |
| `requested_token_type` | `urn:ietf:params:oauth:token-type:id-jag` |
| `subject_token_type` | `urn:ietf:params:oauth:token-type:id_token` |
| `subject_token` | 本人の ID Token |
| `audience` | 受け入れ先の issuer 一つ |
| `resource` | 発行ポリシーに含まれる受け入れ先リソース |
| `scope` | 発行ポリシー内の必要なスコープ |

`actor_token` は指定しない。
応答の `access_token` 欄には有効期間 300 秒の ID-JAG が入り、`issued_token_type` は ID-JAG の URN、`token_type` は `N_A` になる。
この値を MCP や業務 API のアクセストークンとして直接送らない。

次に受け入れ先のクライアントで認証し、受け入れ先のトークンエンドポイントへ `grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer` と `assertion=<ID-JAG>` を送る。
DPoP に束縛した ID-JAG では、代わりに `grant_type=urn:ietf:params:oauth:grant-type:jwt-dpop` と同じ鍵の Proof が必要である。
mTLS に束縛した場合は同じ証明書を提示する。

成功したら受け入れ先のアクセストークンで対象を呼び、発行元の `IdJagIssued`、受け入れ先の `IdJagRedeemed` と対象リソースを確認する。
同じ ID-JAG の再利用、別のクライアント、未連携の利用者、許可されない issuer、別の送信者の鍵では拒否されることも試す。
このフローでは refresh token は発行されない。

## トークンの更新と送信者の保護

長い対話では、元のグラントが対応し、クライアントに `refresh_token` のグラントがあり、`offline_access` を許可した場合に refresh token を使う。
自律 Agent の `client_credentials` と、監督下 Agent の CIBA 発行を refresh token の経路に置き換えない。

| 操作 | 必要な扱い |
| --- | --- |
| refresh | トークンエンドポイントへ `grant_type=refresh_token` と最新の `refresh_token` を送る。confidential クライアントは認証する |
| ローテーション | 応答の新しい refresh token に置き換える。古い値の再利用は同じファミリーを失効させる |
| スコープ | 元の付与から広げない |
| リソース | 元の付与に束縛した対象だけを指定する。refresh 時に別の対象を追加しない |
| DPoP / mTLS | 元の鍵または証明書の証明を保つ。別の鍵への変更や Bearer への変更はしない |

Token Exchange でも、通常は元のトークンと同じ送信者の証明が必要である。
クライアントアサーションの署名が通ったことは、元の DPoP トークンを使う権利の証明を代替しない。

## 自動の確認

| 確かめること | コマンド |
| --- | --- |
| 付与なしの拒否と付与後の交換 | `mise run test-go-one -- ./backend/oauth2/internal/handlers_http TestExchangeRequiresTheUsersDelegationGrant` |
| 一人の撤回が他人に影響しないこと | `mise run test-go-one -- ./backend/oauth2/internal/handlers_http TestRevocationStopsOnlyThatUsersDelegation` |
| 再付与で古いトークンが復活しないこと | `mise run test-go-one -- ./backend/oauth2/internal/handlers_http TestRegrantDoesNotReviveEarlierTokens` |
| ID-JAG の発行と受け入れ | `mise run test-go-one -- ./backend/shared/http/server_http TestCrossAppAccessIssueAndRedeem_REQ_OAUTH2_066` |
| ID-JAG の DPoP の束縛 | `mise run test-go-one -- ./backend/shared/http/server_http TestCrossAppAccessDPoPBinding_REQ_OAUTH2_071` |

これらはテスト用の利用者、鍵、連携先で契約を検証する。
実際の二つの認可サーバーで試す場合は、上の準備手順でポリシー、連携 ID、鍵の到達性をそれぞれ設定する。
契約の詳細は[トークン](../modules/oauth2/features/token.md)と[同意と本人の付与](../modules/oauth2/features/consent.md)を参照する。
