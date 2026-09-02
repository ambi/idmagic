# Deployment

## Runtime units

システムには次の 3 つの実行単位がある。

- **API プロセス**：`backend/cmd/idmagic/` の `main` パッケージが起動し、`backend/cmd/internal/bootstrap` が依存注入を担う。
- **ワーカー**：`backend/cmd/idmagic-worker/` が永続化されたジョブを取得してハンドラーを実行し、API とは独立して水平スケールできる。
- **バッチ**：`backend/cmd/idmagic-batch/` が外部スケジューラーから起動され、保持期限を過ぎたデータの削除または署名鍵のライフサイクル処理を 1 回実行して終了する。

すべての実行単位は、同じ Go モジュールと Bounded Context の実装を再利用する。実行単位の一覧は別の台帳に重複して持たず、エントリーポイントと対応する `mise` のビルドタスクから導く。

API プロセスは認証・プロトコル系、ポータル系、管理系、SCIM、Shared Signals の受信をすべて 1 つの待受に載せる。種別ごとに実行単位を分けない理由と再検討する条件は [contexts/system/decisions.md](contexts/system/decisions.md#no-api-plane-separation) が持つ。

React の UI はこれらとは別のビルド成果物であり、別のサービスとして配信する。ブラウザーから見える境界を同一オリジンに揃えるのはゲートウェイの役目である。

```text
Browser
  |
  | same origin
  v
Gateway / static server (Caddy, Nginx, CDN + proxy, etc.)
  |-- /login, /consent, /device, /status, /admin/* -> React SPA
  `-- /api/* and OAuth/OIDC endpoints                -> Go
```

Caddy は参照用の設定であり、必須のランタイムではない。同一オリジンの境界、TLS、ヘッダー、経路制御という契約を保つゲートウェイなら置き換えられる。同一オリジンであることは利便のためではなく、Cookie のスコープと `Origin` の検証がこの前提の上に成り立っているためである。

## Horizontal scaling reference topology

本番の参照トポロジは、製品に依存しない次の接続関係を持つ。

```text
Clients
  |
  v
Gateway / load balancer ----> React static assets
  |
  v
Stateless API replicas (N)
  |                         Externally scheduled batch processes
  |                                      |
  v                                      v
PostgreSQL shared state <---- Worker replicas by execution lane
```

ゲートウェイまたは負荷分散装置は TLS、同一オリジン、経路制御、健全な API レプリカへの分散を担い、利用者またはテナントの正となる状態を持たない。API レプリカはプロセス内に正となる状態を持たず、同じ要求をどのレプリカへ送っても PostgreSQL の共有状態から同じセキュリティ判断を行う。この境界により、API レプリカは [capacity.md](capacity.md) の算出結果に従って追加または削除できる。

ワーカーは `latency_sensitive`、`default`、`bulk` の実行レーンごとに専用の実行枠を持ち、API とも他のレーンとも独立して水平スケールする。外部スケジューラーは `idmagic-batch` を一回限りの処理として起動し、常駐ワーカーの実行枠を使用しない。API、ワーカー、バッチは永続状態と一時状態の唯一の共有ストアとして PostgreSQL を使う。

プロセス内キャッシュ、ゲートウェイ、CDN は Discovery と JWKS のように正となる状態から再生成できる公開応答だけを保持できる。認可コード、セッション、再送防止、流量制限の正をキャッシュへ移してはならず、キャッシュを失っても正しさが変わらない構成にする。鍵またはテナント設定の変更時は版または明示的な無効化で古い応答を排除し、TTL は配送失敗に備える上限として使う。

PostgreSQL は論理的に単一の正である。パーティショニング、読み取りレプリカ、接続プール、テナント分散配置はデータ層の容量設計が定め、物理的な冗長化、フェイルオーバー、障害種別ごとの縮退は高可用性設計が定めるため、本書は特定のクラウド製品または物理トポロジを要求しない。

## Health probes and graceful drain

Kubernetes 向けのヘルスチェックは、生存確認、受付可否、起動完了を別々のエンドポイントに分ける。これらを 1 つにまとめると、PostgreSQL の一時的な障害で回復可能な Pod を繰り返し再起動したり、応答できないレプリカへ通信を流し続けたりするためである。従来の `/health` は起動時設定のラベルと、形式の版を持つ有効機能メタデータを返すために残す。このメタデータは `FeatureRegistry` の識別子、機能版、成熟度、更新方針だけを公開し、設定値または無効な機能を含めない。

- **`/livez`** はデッドロックなど回復不能な状態でのみ失敗する。一時的な依存障害では `200` を返し、自然に回復できる Pod を再起動させない。
- **`/readyz`** は必須の依存（PostgreSQL）へ短いタイムアウト（既定値は `1s`）で並行に問い合わせ、到達できなければ `503` を返す。`?verbose` を付けると、依存ごとに `healthy`、`degraded`、`unavailable` の状態を返す。
- **`/startupz`** はアプリケーションの初期化（初期データの確認を含む）が完了すると `200` を返す。
- **`/health`** は後方互換のために残しており、従来どおり起動時の設定のラベルだけを返す。

`SIGTERM` または `SIGINT` を受けると停止状態に入り、`/readyz` は直ちに `503`（`unavailable`）を返す。負荷分散装置が対象を外す時間を確保するため、退避猶予期間（`DRAIN_GRACE_PERIOD_SECONDS`、既定値は `5s`）を待ってから HTTP サーバーの停止を始める。

## Availability and shared state

レプリカを複数動かすには `postgres` のランタイム（`PERSISTENCE=postgres`、`DATABASE_URL`）が必要である。共有される状態は永続的なものも一時的なものも、レプリカごとのプロセスメモリではなくすべて PostgreSQL に置く。

- **永続的**：リフレッシュトークン、監査イベント、認証イベントの集計バケット、ログインセッション。ログイン済みのブラウザーセッションは `authentication_sessions` を唯一の正とするため、API レプリカを再起動または順次入れ替えても有効なセッションは失われない。利用者の操作、ログアウト、アカウントの無効化による失効では行を削除せず、`revoked_at` と `revoke_reason` を記録する。このため、失効リクエストを再送しても安全である。
- **一時的**：認可リクエスト、認可コード、PAR、デバイスコード、DPoP とクライアントアサーションの再送防止、アクセストークンの拒否リスト、WebAuthn のチャレンジ、ログイン試行のスロットル、エンドポイントのレート制限カウンター。いずれも短命で、再試行しても安全である。すべての行が `expires_at` を持ち、読み取りを `expires_at > now()` で絞り込むため、有効期限の正しさは `idmagic-worker` が領域回収のために行う最善努力型の削除処理に依存しない。

ログインスロットルの状態は必ず共有する。レプリカごとにカウンターを持つと、失敗試行が `N` 個のレプリカへ分散され、アカウント単位と IP 単位の閾値がシステム全体では最大 `N` 倍に緩むためである。PostgreSQL の共有カウンターを `SELECT ... FOR UPDATE` で直列化して更新し、すべてのレプリカを通じて試行回数を数える。アカウントと IP の識別子は SHA-256 でハッシュ化し、平文のユーザー名や IP は保存しない。

スロットルはログイン可否の判定に使うため、障害時は**フェイルクローズ**とする。ストアへ到達できず状態を確認できない場合、ログイン試行を許可せず拒否する。複数レプリカで運用する場合は、PostgreSQL も地域冗長や同期スタンバイなどの高可用構成にする。

`memory` のランタイムはこの状態をプロセス内に保持するので、**単一レプリカとテスト専用**である。

## Domain event delivery

ドメインイベントは、それを発行した操作と同じ goroutine の中で、その操作自身の書き込みが済んだ後に配信する。キュー、メッセージ基盤、専用の配信プロセスはいずれも介さない。したがってここで定める保証は、非同期の配送基盤が持つ保証とは種類が違う。以下は**現在守られている保証だけ**であり、書かれていない性質は保証しない。

- **順序**：1 つの goroutine が発行した複数のイベントは、その順序で監査記録に追記される。それ以外の順序は保証しない。異なる要求、異なるワーカー、異なるレプリカが発行したイベントの間に全順序は無く、`occurredAt` は発行元の時計であって順序付けの根拠にはならない。
- **重複**：配信点は再試行しないので、1 回の発行が 2 件の記録になることはない。ただし、発行を含む操作そのものが再実行されれば再び発行される。ジョブのハンドラーは[少なくとも 1 回](contexts/jobs/decisions.md)実行されるので、ジョブ経由で発行されるイベントは重複しうる。監査記録は事実の追記であり、重複した記録は重複した実行が起きたという事実として正しい。
- **遅延**：配信点にキューが無いため、消費者の遅れという状態は存在しない。監査記録の追記は発行元の要求の中で完了し、その所要時間は [capacity.md](capacity.md#measurement-boundary) の測定境界の内側に入る。
- **消失**：**保証しない。** 状態の変更とイベントの発行は同一トランザクションに入っていない。出力、変換、追記のいずれが失敗しても記録を残して先へ進み、発行元の操作は成功したまま返る。コミットと追記の間でプロセスが異常終了すれば、そのイベントは失われ、回復する経路は無い。
- **再生**：持たない。再配送のために保持している発行の意図が無く、監査記録は追記のみの読み取りモデルであってキューではない。

[capacity.md](capacity.md#degradation-order) は「監査イベントを黙って破棄することは縮退手段に含めない」と定める。これは容量超過時に**選んで**捨てることを禁じる規範であって、障害時に失われないことの保証ではない。両者は別のことを述べている。障害時の消失は [threat-model.md](threat-model.md) の THREAT-074 が扱う。

### Atomicity of state change and emission

状態の変更とその通知が同一トランザクションで確定するのは、現在 1 経路だけである。IdManagement の User の変更と、そこから導かれる LifecycleWorkflow の実行の生成は、IdGovernance が実装する境界のポートが 1 つのトランザクションで確定する。この経路だけは、片方だけが残る状態を作らない。

残りは保証されていない。次の 2 つを負債として記録する。

- **外向きプロビジョニングの捕捉**：呼び出し元のコミット直後に、別のトランザクションで配信対象を捕捉する。2 つのコミットの間で異常終了すると捕捉が失われ、回復する経路は無い。
- **ドメインイベントの発行**：上記のとおり、発行元の書き込みとは別に、失敗を伝播しない配信点を通る。

どちらも Outbox に相当する仕組みを持たない。是正は経路ごとに切り出し、この節は現在の状態を述べるにとどめる。保証されていないことを保証されているかのように書けば、記録が残っていることを根拠に判断する運用そのものが誤る。

## HTTP server hardening

外部に公開する HTTP サーバーには、本番環境で安全なタイムアウトとリクエスト本体の上限を適用する。低速な接続や過大なリクエストによる接続枠とメモリの枯渇を防ぐためである（`gosec G112`、CWE-400）。上限を超えた本体は `413` で拒否する。

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_READ_HEADER_TIMEOUT` | `10s` | リクエストのヘッダーを読む上限時間（slowloris の抑止） |
| `HTTP_READ_TIMEOUT` | `30s` | リクエスト全体を読む上限時間 |
| `HTTP_WRITE_TIMEOUT` | `60s` | レスポンスを書く上限時間 |
| `HTTP_IDLE_TIMEOUT` | `120s` | 持続接続の待機時間の上限 |
| `HTTP_MAX_BODY_BYTES` | `1048576` | リクエスト本体の最大バイト数（1 MiB） |

これは多層防御であり、境界プロキシの代わりではない。大量のリクエストと TLS ハンドシェイクを悪用する slowloris には、前段のリバースプロキシで対処する。

## Security response headers

境界ミドルウェアは、すべてのバックエンドレスポンスに `X-Content-Type-Options: nosniff`、`Referrer-Policy: no-referrer`、`X-Frame-Options: DENY`、厳格な `Content-Security-Policy`（`default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'`）を適用する。`frame-ancestors 'none'` と `X-Frame-Options: DENY` により、ログイン、同意、ポータル画面の埋め込みとクリックジャッキングを防ぐ。CSP では `'unsafe-inline'` を使わない。IdMagic が出力する埋め込みスクリプトは、SAML ACS と WS-Fed の POST バインディングで使う固定の自動送信処理だけである。これらはレスポンスごとに `script-src 'sha256-…'` で許可し、`form-action` も送信先エンドポイントに限定する。

CSP と `frame-ancestors` はルートごとの判断が必要なので、IdMagic がヘッダーを設定する。これにより、最小構成のプロキシの背後でも、プロキシがない場合でも保護が成立する。単一ページアプリケーションはゲートウェイが配信し、静的 HTML に対して `script-src 'self'` を含む CSP を設定する。

HSTS は TLS を終端する側が設定する。`Strict-Transport-Security` は既定で無効とし、平文の `http` を使う開発環境に影響させない。TLS がこの区間か、その手前で終端される場合にのみ有効化する。通常の構成では境界のプロキシに任せ（`HSTS_ENABLED=false`）、アプリケーション自身が表明すべき場合に `HSTS_ENABLED=true` とする（`HSTS_MAX_AGE_SECONDS` と `HSTS_INCLUDE_SUBDOMAINS` で調整する）。

画面を壊さずに CSP を厳しくするには、`CSP_REPORT_ONLY=true` で `Content-Security-Policy-Report-Only` を出し、`CSP_REPORT_URI=<url>` で違反を収集し、観察してから強制へ戻す。

## Endpoint rate limiting

`backend/shared/ratelimit`（`ports`、`db_memory`、`db_postgres`）は業務上の Aggregate ではなく、`backend/shared/security` の `EnvelopeCrypto` と同様の技術的な共通機能である。OAuth2 と Authentication の双方に属するエンドポイント（`/authorize`、`/token`、`/par`、`/device_authorization`、`/bc-authorize`、`/api/auth/password_reset/*`）を保護するため、特定の Context には置かない。アカウント単位および IP 単位のログインスロットルとは目的が異なり、その代わりにはならない。

ポートは `Allow(ctx, tenantID, policyID, key, now)` という単一の操作を公開する。`(tenant_id, policy_id, key_hash)` をキーとする固定時間枠のカウンターを持ち、許可と拒否のいずれでも、リクエストごとに 1 回加算する。失敗だけを数えるログインスロットルとは異なる。

`endpoint_rate_limit_counters` は、すべてのリクエストで更新される一時データなので `UNLOGGED` とする。障害でカウンターを失っても時間枠がリセットされるだけで、永続的な安全性の保証は失われない。一方、失うと保証が弱まる `login_throttle_counters` とアクセストークンの拒否リストは `LOGGED` のままにする。

ストアへ到達できない場合は、すべてのポリシーでフェイルクローズに拒否する。保護対象のエンドポイントはすでに PostgreSQL を必須としているため、依存先の種類は増えない。

ポリシーごとに、固定時間枠の最大リクエスト数と枠の長さを秒単位の環境変数で設定できる。運用者はコードを変更せずに閾値を調整できる。

| Policy | Env (max / window) | Default |
| --- | --- | --- |
| `token` | `RATE_LIMIT_TOKEN_MAX_REQUESTS` / `RATE_LIMIT_TOKEN_WINDOW_SECONDS` | 60 / 60s |
| `authorize` | `RATE_LIMIT_AUTHORIZE_MAX_REQUESTS` / `RATE_LIMIT_AUTHORIZE_WINDOW_SECONDS` | 30 / 60s |
| `par` | `RATE_LIMIT_PAR_MAX_REQUESTS` / `RATE_LIMIT_PAR_WINDOW_SECONDS` | 30 / 60s |
| `device_authorization` | `RATE_LIMIT_DEVICE_AUTHORIZATION_MAX_REQUESTS` / `RATE_LIMIT_DEVICE_AUTHORIZATION_WINDOW_SECONDS` | 20 / 60s |
| `backchannel_authentication` | `RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_MAX_REQUESTS` / `RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_WINDOW_SECONDS` | 20 / 60s |
| `password_reset` | `RATE_LIMIT_PASSWORD_RESET_MAX_REQUESTS` / `RATE_LIMIT_PASSWORD_RESET_WINDOW_SECONDS` | 5 / 900s |

キーは `client_id`、IP、`identifier_hash` の組み合わせである。`client_id` はシークレットではないが、IP とパスワード再設定の識別子は保存前に SHA-256 でダイジェスト化する。ログインスロットルの `hashThrottleIdentifier` と同じ方式である。閾値を超えると、`Retry-After` と TypeSpec の `RateLimitedError` を伴う HTTP 429 を返す。
