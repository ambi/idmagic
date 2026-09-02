# Capacity

## Evidence classes

容量に関する数値は、次の三つの区分のいずれかとして扱う。

- **Specification target**：満たすべきサービス目標または容量受入れ目標であり、未達は構成または実装の改善を要する。
- **Planning assumption**：構成を算出するための未実測の入力値であり、製品が達成済みであることを示さない。
- **Measurement**：日付、ソース版、実行環境、データ分布、負荷構成、試験時間、結果の保存先を伴う再現可能な実測値である。

本書に Measurement はまだない。ステージングの容量検証では Planning assumption を Measurement へ置き換え、同じ参照運用プロファイルとサービス目標を使って容量受入れ目標を検証する。実測値が Planning assumption を下回っても Specification target を暗黙に引き下げず、必要な構成または後続の設計変更を明示する。

## Measurement boundary

サービス目標の測定境界は、ゲートウェイから API プロセスへ到達し、`http_requests_total` と `http_request_duration_seconds` に記録された HTTP 要求である。`route` は解決済みのパスではなく登録済みのルートパターンで集約し、realm 接頭辞を持つ同じ操作も同じエンドポイント群へ含める。現行メトリクスはゲートウェイより外側の通信を観測しないため、本書の値だけを根拠に利用者からゲートウェイまでの可用性を保証してはならない。

レイテンシーの母集団には、対象のルートとメソッドで API プロセスが完了した全応答を状態コードにかかわらず含める。クライアントが応答前に切断して状態コードを返せなかった要求はレイテンシーと非 5xx 比率の母集団から除外するが、`http_request_aborts_total` で別に監視し、目標を満たした要求としては数えない。計画停止、デプロイ、依存先障害、流量制限、認証失敗、入力不正は除外条件にしない。

非 5xx 比率は、対象母集団の `http_requests_total` のうち `status_code` が 500 未満である応答の割合とする。したがって、認証失敗、流量制限、入力不正などの 4xx は非 5xx に含む一方、依存先障害を含むすべての 5xx は失敗として数える。

可用性は 5 分の時間区分で評価する。対象要求が 1 件以上完了した区分を観測対象とし、API の Prometheus スクレイプ対象が区分を通じて少なくとも一つ利用可能であり、対象要求に非 5xx 応答が 1 件以上あれば利用可能な区分とする。要求がない区分は成功として水増しせず母集団から除外し、スクレイプ対象が失われた区分は利用不能として数える。非 5xx 比率が部分障害の割合を別に制約するため、可用性はサービスへ到達して処理を完了できるかを時間で評価する。

## Service level objectives

次の Specification target は 30 日の移動窓で評価する。パーセンタイルはフリート全体の `http_request_duration_seconds` ヒストグラムから算出し、各表の `Population` と `Exclusions` は [Measurement boundary](#measurement-boundary) に従う。

**各行は安定した ID を持つ。ID は一度参照されたら変更しない。** 他の文書、アラート定義、負荷試験は**この ID を参照し、数値を再掲しない**。数値を写すと、正本を変えたときに写しが黙って古くなる。数値そのものを持たざるをえない資材（Prometheus の式、k6 のしきい値）は、その数がどの目標に由来するかを ID で名指しする。

アラートは目標そのものを判定しない。**アラートが判定するのは error budget の消費速度であり、時間窓が違う。** たとえば SLO-PRIMARY-ERRORS は 30 日の移動窓で評価するが、対応するアラートは 5 分のバーンレート窓を見る。同じ数を使っていても、それは現在の設計判断であって、一致し続ける義務ではない。

### Latency

| ID | Endpoint population | Method | Target | Metric | Notes |
| --- | --- | --- | --- | --- | --- |
| SLO-LOGIN-LATENCY | `/api/auth/login` | `POST` | p99 ≤ 300 ms | `http_request_duration_seconds` | `SubmitBrowserLogin` |
| SLO-AUTHORIZE-LATENCY | `/authorize` | `GET` | p99 ≤ 500 ms | `http_request_duration_seconds` | `Authorize` |
| SLO-PAR-LATENCY | `/par` | `POST` | p99 ≤ 200 ms | `http_request_duration_seconds` | `PushAuthorizationRequest` |
| SLO-TOKEN-LATENCY | `/token` | `POST` | p99 ≤ 300 ms | `http_request_duration_seconds` | `Token` |
| SLO-INTROSPECT-LATENCY | `/introspect` | `POST` | p99 ≤ 50 ms | `http_request_duration_seconds` | `Introspect` |
| SLO-REVOKE-LATENCY | `/revoke` | `POST` | p99 ≤ 100 ms | `http_request_duration_seconds` | `Revoke` |
| SLO-USERINFO-LATENCY | `/userinfo` | `GET`, `POST` | p99 ≤ 100 ms | `http_request_duration_seconds` | `UserInfo`, `PostUserInfo` |
| SLO-DISCOVERY-LATENCY | `/.well-known/openid-configuration`, `/.well-known/oauth-authorization-server` | `GET` | p99 ≤ 20 ms | `http_request_duration_seconds` | OIDC Discovery と RFC 8414 の後継操作を同じ目標で評価する |
| SLO-JWKS-LATENCY | `/jwks`, `/realms/{tenant_id}/jwks` | `GET` | p99 ≤ 20 ms | `http_request_duration_seconds` | 既定テナントと明示テナントを含む |
| SLO-REGISTER-LATENCY | `/register` | `POST` | p99 ≤ 500 ms | `http_request_duration_seconds` | `RegisterClient` |
| SLO-DEVICE-AUTHORIZATION-LATENCY | `/device_authorization` | `POST` | p99 ≤ 300 ms | `http_request_duration_seconds` | `DeviceAuthorization` |
| SLO-FEDERATION-CALLBACK-LATENCY | `/api/auth/federation/oidc/callback`, `/api/auth/federation/saml/callback` | `GET`, `POST` | p95 ≤ 2 s | `http_request_duration_seconds` | 上流とのネットワーク交換を含む |

旧目標にあった「セッション一覧の先頭ページの p95 ≤ 100 ms」は Specification target に復元しない。現行の `route`、`method`、`status_code` ラベルでは `/api/account/v1/sessions` の先頭ページと後続ページを区別できず、測定不能な保証になるためである。低カーディナリティの測定方法を定めた時点で再導入を判断する。

### Non-5xx ratio

| ID | Endpoint population | Target | Metric |
| --- | --- | --- | --- |
| SLO-PRIMARY-ERRORS | `/api/auth/login`, `/authorize`, `/par`, `/token`, `/revoke`, `/userinfo`, `/register`, `/device_authorization` | ≥ 99.9% | `http_requests_total` |
| SLO-LOOKUP-ERRORS | `/introspect`, `/.well-known/openid-configuration`, `/.well-known/oauth-authorization-server`, `/jwks`, `/realms/{tenant_id}/jwks` | ≥ 99.99% | `http_requests_total` |

母集団が Context をまたぐため、ID は Context ではなく要求の性質で名付ける。`PRIMARY` は主要な要求経路、`LOOKUP` は参照専用で目標の厳しい経路である。

外部連携ログインのコールバックにはレイテンシー目標だけを復元し、非 5xx 比率の新しい数値は設けない。旧 SCL に対応する値がなく、上流 IdP の拒否や障害を IdMagic の失敗と区別する母集団も未定義だからである。

### Availability

| ID | Population | Target | Window and budgeting | Metric |
| --- | --- | --- | --- | --- |
| SLO-OAUTH2-AVAILABILITY | Non-5xx ratio の対象となる OAuth2/OIDC エンドポイント全体 | ≥ 99.9% | 30 日、5 分の時間区分 | `http_requests_total` と Prometheus のスクレイプ状態 |
| SLO-TOKEN-AVAILABILITY | `/token` | ≥ 99.95% | 30 日、5 分の時間区分 | `http_requests_total` と Prometheus のスクレイプ状態 |

### Capacity acceptance

フリートは、[Reference operating profile](#reference-operating-profile) の全データを投入し、[Peak request profile](#peak-request-profile) の要求を同時に 15 分間ウォームアップした後、60 分間処理して次の Specification target を満たす。

| ID | Endpoint | Required rate | Required objectives |
| --- | --- | --- | --- |
| CAP-TOKEN-THROUGHPUT | `/token` | 5,000 rps | SLO-TOKEN-LATENCY と SLO-PRIMARY-ERRORS を満たす |
| CAP-AUTHORIZE-THROUGHPUT | `/authorize` | 1,000 rps | SLO-AUTHORIZE-LATENCY と SLO-PRIMARY-ERRORS を満たす |
| CAP-INTROSPECT-THROUGHPUT | `/introspect` | 20,000 rps | SLO-INTROSPECT-LATENCY と SLO-LOOKUP-ERRORS を満たす |

`SLO-` は 30 日の移動窓で評価する運用上の目標、`CAP-` は固定した試験条件で確かめる容量受入れ目標である。前者は本番の観測、後者は試験で判定する。

このスループットは本番の 30 日窓で要求のない時間まで成功扱いする SLO ではなく、データ分布、要求構成、試験時間を固定した容量受入れ目標である。試験中に流量制限で返した 429 は非 5xx 比率には含むが、Required rate の処理済み要求には数えない。

## Reference operating profile

参照運用プロファイルは Specification target の規模と Planning assumption の分布を組み合わせた設計入力であり、すべてのデプロイに要求する最小構成でも、製品が超過を拒否するハード上限でもない。

### Tenant and activity distribution

| Tenant class | Tenant count | Users per tenant | Users | Classification |
| --- | ---: | ---: | ---: | --- |
| Small | 90,000 | 20 | 1,800,000 | Planning assumption |
| Medium | 9,000 | 500 | 4,500,000 | Planning assumption |
| Large | 900 | 3,000 | 2,700,000 | Planning assumption |
| Very large | 100 | 10,000 | 1,000,000 | Planning assumption |
| Total | 100,000 | — | 10,000,000 | Specification target |

この分布では中央値が 20 ユーザー、p90 が 500 ユーザー、p99 が 3,000 ユーザー、最大が 10,000 ユーザーとなる。10,000 ユーザーを超える単一テナントは容量計画から排除するのではなく、この分布の外側として大規模テナント向けの性能検証を追加で要する。

| Activity | Value | Classification |
| --- | ---: | --- |
| Monthly active users | 4,000,000 | Planning assumption（全ユーザーの 40%） |
| Daily active users | 1,000,000 | Planning assumption（全ユーザーの 10%） |
| Active tenants in the busiest 5-minute interval | 20,000 | Planning assumption（全テナントの 20%） |
| Concurrent valid browser sessions at peak | 200,000 | Planning assumption（全ユーザーの 2%） |
| New browser sessions per day | 500,000 | Planning assumption |

### Object and retention profile

行サイズはテーブル本体の論理サイズを見積もるための Planning assumption であり、インデックス、MVCC の余白、ページの空き、保守作業の余裕は後述のストレージ係数で加える。

| Object | Count or arrival | Retention assumption | Assumed row bytes | Classification |
| --- | ---: | --- | ---: | --- |
| Tenant | 100,000 | 削除まで | 8 KiB | Specification target / Planning assumption |
| User | 10,000,000 | 削除と匿名化の方針に従う | 4 KiB | Specification target / Planning assumption |
| Application | 500,000（5 / tenant） | 削除まで | 8 KiB | Planning assumption |
| OAuth2 client | 1,000,000（2 / application） | 削除まで | 4 KiB | Planning assumption |
| Group | 1,100,000（10 / tenant + 1 / 100 users） | 削除まで | 2 KiB | Planning assumption |
| Group membership | 50,000,000（5 / user） | 所属解除まで | 256 B | Planning assumption |
| Authentication session row | 500,000 / day | 失効または期限切れ後を含む 90 日 | 1 KiB | Planning assumption / Authentication の現行保持方針 |
| Refresh token row | 2,000,000 / day | 絶対期限 30 日 | 512 B | Planning assumption / OAuth2 の現行期限 |
| Authentication event | 1,000,000 / day | 容量見積もりでは保守的に 365 日 | 1 KiB | Planning assumption |
| Audit event | 5,000,000 / day | 2,555 日 | 1 KiB | Planning assumption / Audit の現行保持方針 |

検索属性の副表 `audit_event_search_attributes` は上表に現れていないが、監査イベント 1 件につき、そのイベントが値を持つ軸の数だけ行が増える。1 件当たり 6 行、1 行 128 B を Planning assumption とすると、1 日当たり 3,000 万行、約 3.7 GiB の論理データが加わる。保持期間は本体と同じ 2,555 日であり、外部キーの連鎖削除で本体と一緒に消える。

この仮定では、監査イベントだけで 1 日当たり約 4.8 GiB の論理データ、7 年で約 11.9 TiB の論理データになる。物理ストレージ予算は、`Σ(保持行数 × 実測平均行バイト数) × 2.5 + 30 日分の物理成長量` 以上とする。係数 2.5 はインデックス、MVCC、保守時の空き容量を含む Planning assumption であり、バックアップ、WAL の別保管、読み取りレプリカは含めない。上表の仮定をそのまま使う初期予算は約 32 TiB であり、実測行サイズとデータ層の物理配置に合わせて更新する。

## Peak request profile

次の値は同じ最繁 15 分に生じる API 到達率の Planning assumption である。キャッシュ可能な公開文書は利用者側の要求率と API 到達率を分け、容量算出には API 到達率を使う。

| Endpoint family | Client-side peak | API peak | Assumption |
| --- | ---: | ---: | --- |
| `/token` | 5,000 rps | 5,000 rps | キャッシュ不可 |
| `/authorize` | 1,000 rps | 1,000 rps | キャッシュ不可 |
| `/api/auth/login` | 600 rps | 600 rps | キャッシュ不可 |
| `/par` | 1,000 rps | 1,000 rps | キャッシュ不可 |
| `/introspect` | 20,000 rps | 20,000 rps | キャッシュ不可 |
| `/revoke` | 1,000 rps | 1,000 rps | キャッシュ不可 |
| `/userinfo` | 5,000 rps | 5,000 rps | キャッシュ不可 |
| `/register` | 50 rps | 50 rps | キャッシュ不可 |
| `/device_authorization` | 200 rps | 200 rps | キャッシュ不可 |
| Federated callbacks | 200 rps | 200 rps | 上流交換を含む |
| Session list | 200 rps | 200 rps | 先頭ページと後続ページを含む |
| Discovery | 20,000 rps | 2,000 rps | ゲートウェイまたは CDN のヒット率 90% |
| JWKS | 50,000 rps | 5,000 rps | ゲートウェイまたは CDN のヒット率 90% |

Discovery と JWKS の 90% というヒット率は保証値ではない。キャッシュが空になった場合にも正しさは変わらないが、API 到達率が増えるため、容量検証にはコールドキャッシュの試験を含める。

ジョブの実行枠は API とは別に算出する。

| Execution lane | Peak arrival | p95 handler time | Required slots with 1.5 safety factor | Classification |
| --- | ---: | ---: | ---: | --- |
| `latency_sensitive` | 50 jobs/s | 200 ms | 15 | Planning assumption |
| `default` | 20 jobs/s | 1 s | 30 | Planning assumption |
| `bulk` | 2 jobs/s | 10 s | 30 | Planning assumption |

### Non-protocol request profile

上の表の母集団は認証・プロトコル系だけである。管理コンソール、管理 API の自動化、SCIM、Shared Signals の受信は行を持たず、`/api/account/v1/sessions` のセッション一覧だけがポータル系から 1 行入っている。行が無いことは到達率がゼロであることを示さないので、本節でこの 4 種別を補う。値はすべて Planning assumption であり、Measurement は 1 つも無い。

種別は用途で分けた API のまとまりを指す。管理系は `/api/admin/v1`、ポータル系は `/api/account/v1`、SCIM は `/scim/v2`、Shared Signals の受信は `/ssf` である。Shared Signals のストリーム管理は `/api/admin/v1/shared-signals` にあるので管理系に数え、`/ssf` に残るのはセキュリティイベントの受信 1 経路だけである。

算出の入力は次のとおりで、幅を持つ。中央値は算出に使う代表値、幅はその入力を単独で振り切ったときの範囲である。

| 種別 | 入力 | 中央値 | 幅 |
| --- | --- | ---: | ---: |
| 管理コンソール | テナント当たりの管理者アカウント数 | 2 | 1–5 |
| 管理コンソール | 最繁 15 分に操作している管理者の割合 | 1% | 0.5–3% |
| 管理コンソール | 操作中の管理者 1 人当たりの API 到達率 | 0.15 rps | 0.05–0.3 rps |
| 管理 API 自動化 | 自動化を持つテナント数 | 5,000 | 3,000–8,000 |
| 管理 API 自動化 | 差分同期の間隔 | 15 分 | 5–60 分 |
| 管理 API 自動化 | 差分同期 1 回当たりのリクエスト数 | 25 | 5–100 |
| 管理 API 自動化 | 毎正時への集中係数 | 1.5 | 1.2–3 |
| 管理 API 自動化 | 夜間全同期の 1 テナント当たりリクエスト数 | 500 | 100–5,000 |
| ポータル | ログイン後にポータルを開く割合 | 10% | 5–20% |
| ポータル | 1 訪問当たりの `/api/account/v1` リクエスト数 | 15 | 8–25 |
| SCIM | SCIM を有効にしたテナントの割合 | 10% | 5–25% |
| SCIM | 差分同期の間隔 | 40 分 | 5–60 分 |
| SCIM | 差分同期 1 回当たりのリクエスト数 | 3 | 1–10 |
| SCIM | 同時に全同期を実行するテナント数 | 50 | 10–200 |
| SCIM | 全同期の 1 テナント当たりリクエスト数 | 1,000 | 20–10,000 |
| Shared Signals | 受信ストリームを設定したテナントの割合 | 3% | 1–10% |
| Shared Signals | 1 ストリーム当たりのイベント発生率 | 0.01 events/s | 0.001–0.05 events/s |
| Shared Signals | 集中配信のバースト倍率 | 10 | 3–20 |

管理コンソールとポータルの到達率は [Tenant and activity distribution](#tenant-and-activity-distribution) の 100,000 テナントと、上の表の `/api/auth/login` 600 rps を起点に導く。ポータルを最繁時のログイン率から導くのは、どちらも同じ最繁 15 分の値であり、日次合計から導くと同じ窓の値にならないためである。

| 種別 | 通常時 | 最繁時 | 集中実行時 | 最繁時の幅 | 幅を支配する入力 |
| --- | ---: | ---: | ---: | ---: | --- |
| 管理コンソール | 50 rps | 300 rps | — | 75–2,250 rps | 操作中の管理者の割合、テナント当たりの管理者数 |
| 管理 API 自動化 | 140 rps | 210 rps | 1,100 rps | 10–2,500 rps | 差分同期 1 回当たりのリクエスト数、同期の間隔 |
| ポータル | 150 rps | 900 rps | — | 240–3,000 rps | ポータルを開く割合、1 訪問当たりのリクエスト数 |
| SCIM | 13 rps | 50 rps | 110 rps | 17–1,000 rps | SCIM を有効にしたテナントの割合、同期の間隔 |
| Shared Signals | 30 rps | 90 rps | 300 rps | 3–1,500 rps | 受信ストリームを設定したテナントの割合、イベント発生率 |

集中実行時は、管理 API 自動化の夜間全同期、SCIM の初期プロビジョニングと全同期、Shared Signals の大量失効と再送を指す。管理コンソールとポータルは対話的なので集中実行の形を持たない。上の表の Session list 200 rps はポータルの行の内数であり、別に足さない。

最繁 15 分の合計は中央値で 1,550 rps、集中実行時の値を単純に足した上界で 2,710 rps である。合計は認証・プロトコル系の 41,250 rps に対して到達率で 4–7% にあたるが、1 リクエスト当たりの費用が高いため、この比率をそのまま資源の比率として読んではならない。

幅は 2 桁に及ぶ。管理 API 自動化の差分同期 1 回当たりのリクエスト数と、SCIM の全同期 1 テナント当たりリクエスト数が最も大きく効くので、この 2 つを Measurement の第一の対象とする。

### Domain event emission

ドメインイベントの配信にキューは無く、発行元の要求またはジョブの中で監査記録の追記まで完了する（[deployment.md](deployment.md#domain-event-delivery)）。したがって、キューの深さ、消費者の遅れ、再生のための実行枠は、いずれも算出する対象が存在しない。代わりに費用は発行元の要求に乗るので、次を Planning assumption として算出に含める。

| Item | Value | Classification |
| --- | ---: | --- |
| 発行 1 件当たりのデータベース書き込み | 本体 1 行 + 検索属性 6 行 | Planning assumption |
| 発行を含む要求に上乗せされる時間 | 5 ms | Planning assumption |
| 再生のために確保する実行枠 | 0 | 再配送の経路を持たないため |

アカウントのセキュリティ通知は同じ配信点から購読するが、送信は要求から切り離して走るので、発行元の応答時間には乗らない。上限の見積もりには API プロセスの同時実行数として数える。

## Sizing rules

API レプリカの必要数は、エンドポイント `e` ごとに `ceil(API peak_e ÷ measured sustainable rate per replica_e × 1.5)` を計算した最大値とする。1 レプリカ当たりの持続処理能力は、参照運用プロファイルのデータと混合負荷の下で対応するレイテンシーと非 5xx 比率を満たした Measurement だけを使う。

実測前の Planning assumption は、`/token` が 250 rps、`/authorize` が 100 rps、`/introspect` が 1,000 rps である。この仮定では API は `ceil(max(5000/250, 1000/100, 20000/1000) × 1.5) = 30` レプリカとなるが、達成済みの構成を示す数値ではない。この 3 つの仮定は認証・プロトコル系だけの混合を前提としており、[Non-protocol request profile](#non-protocol-request-profile) の 4 種別を含んでいない。容量検証では 4 種別を含む混合で持続処理能力を測り、この参照値を置き換える。

種別ごとの 1 レプリカ当たり持続処理能力は、実測前は管理系 100 rps、ポータル系 200 rps、SCIM 50 rps、Shared Signals の受信 100 rps を Planning assumption とし、いずれも実測値の 2 倍から 2 分の 1 の幅を持つものとして扱う。この仮定と、[Non-protocol request profile](#non-protocol-request-profile) の各種別の最大到達率 — 管理系は集中実行時の 1,100 rps に通常時の管理コンソール 50 rps を加えた 1,150 rps、ポータル系は最繁時の 900 rps、SCIM は集中同期時の 110 rps、Shared Signals は集中配信時の 300 rps — からは、単独のプレーンで運用した場合の必要レプリカ数が管理系 18、ポータル系 7、SCIM 4、Shared Signals 5 になる。

**種別ごとに実行単位を分ける構成を評価する場合も、同じ式を使う。** ただし 1 レプリカ当たり持続処理能力は混合負荷ごとに別の Measurement であり、単一 Deployment 用の値を分離後のプレーンへ流用してはならない。上の単独運用時のレプリカ数も、単一 Deployment のレプリカ数と足し引きできる量ではない。

分離に固有の常時増分は総需要から生じない。総需要はどちらの構成でも同じだからである。増分は次の 3 つから生じる。第一に、追加するプレーンが独立に持つ可用性の下限レプリカ数と PodDisruptionBudget である。第二に、`ceil` が構成ごとに 1 回ずつ働くことによる切り上げの重複で、プレーン数 `n` に対して最大 `n − 1` レプリカである。第三に、ある種別の谷が別の種別の山を吸収できなくなることによる余裕の重複である。したがって、認証・プロトコル系以外をまとめて 1 つ追加するだけの最小の分離でも、下限レプリカ数 3 と PodDisruptionBudget が 1 組増え、論理接続予算は `下限レプリカ数 × API pool limit` すなわち 48 接続増える。70% 規則により、利用可能接続の必要量は 69 増える。

実行レーン `l` の必要なワーカー枠は `ceil(peak arrival_l × p95 handler time_l × 1.5)` とする。既定の 1 プロセス 4 枠を使う場合、上表の Planning assumption は `latency_sensitive` 4 レプリカ、`default` 8 レプリカ、`bulk` 8 レプリカとなる。レーン間で枠を融通せず、`bulk` の滞留を理由に `latency_sensitive` の枠を減らさない。

PostgreSQL の論理接続予算は `API replicas × API pool limit + worker replicas × worker pool limit + concurrent batches × batch pool limit + operator reserve` とする。実測前は API 1 レプリカ 16 接続、ワーカー 1 レプリカ 8 接続、同時バッチ 4 個で各 4 接続、運用予約 64 接続を Planning assumption とするため、上の参照構成では 720 接続になる。接続予算は物理データベースまたは接続プールの利用可能接続数の 70% 以下に保ち、残りをフェイルオーバー、保守、偏りの吸収に残す。したがって 720 接続を使う構成は少なくとも 1,029 接続の利用可能容量を要するが、接続プールと物理トポロジの選択はデータ層の容量設計で定める。

ストレージは [Object and retention profile](#object-and-retention-profile) の式で算出し、ディスクまたはボリュームの使用率を 70% 以下に保つ。監査の 7 年保持を短縮して容量不足を解消してはならず、保持層の構成変更は Audit の保持方針を満たす形でデータ層の容量設計が扱う。

## Degradation order

容量を超過した場合も、テナント境界、認証と認可の検証、再送防止、流量制限、監査イベントの完全性を弱めない。状態を確認できない要求を成功扱いにすること、監査イベントを黙って破棄すること、テナントをまたいでキャッシュまたは接続枠を共有することは縮退手段に含めない。

縮退は次の順序で行う。

1. `bulk` レーンの新規取得と外部スケジューラーの保守バッチを遅延させる。
2. `default` レーンの新規取得を遅延させ、`latency_sensitive` レーンの専用枠を維持する。
3. 管理用の集計、エクスポート、再同期など対話的な認証に不要な高コスト処理を明示的な 429 または 503 で拒否する。
4. 新しい動的クライアント登録など、既存セッションの認証とトークン処理に不要な書き込みを明示的に拒否する。
5. `/authorize`、`/token`、`/introspect`、ログインを受け付けられない場合は、状態を部分的に更新せず 429 または 503 で拒否する。

Discovery と JWKS は正となる状態から導出できるためキャッシュ済み応答を利用できるが、鍵またはテナント設定の失効と更新を TTL だけに依存させない。障害種別ごとの遷移、冗長化方式、過負荷保護の実装、運用手順は高可用性設計で定める。
