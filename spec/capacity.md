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

### Latency

| Endpoint population | Method | Target | Metric | Notes |
| --- | --- | --- | --- | --- |
| `/api/auth/login` | `POST` | p99 ≤ 300 ms | `http_request_duration_seconds` | `SubmitBrowserLogin` |
| `/authorize` | `GET` | p99 ≤ 500 ms | `http_request_duration_seconds` | `Authorize` |
| `/par` | `POST` | p99 ≤ 200 ms | `http_request_duration_seconds` | `PushAuthorizationRequest` |
| `/token` | `POST` | p99 ≤ 300 ms | `http_request_duration_seconds` | `Token` |
| `/introspect` | `POST` | p99 ≤ 50 ms | `http_request_duration_seconds` | `Introspect` |
| `/revoke` | `POST` | p99 ≤ 100 ms | `http_request_duration_seconds` | `Revoke` |
| `/userinfo` | `GET`, `POST` | p99 ≤ 100 ms | `http_request_duration_seconds` | `UserInfo`, `PostUserInfo` |
| `/.well-known/openid-configuration`, `/.well-known/oauth-authorization-server` | `GET` | p99 ≤ 20 ms | `http_request_duration_seconds` | OIDC Discovery と RFC 8414 の後継操作を同じ目標で評価する |
| `/jwks`, `/realms/{tenant_id}/jwks` | `GET` | p99 ≤ 20 ms | `http_request_duration_seconds` | 既定テナントと明示テナントを含む |
| `/register` | `POST` | p99 ≤ 500 ms | `http_request_duration_seconds` | `RegisterClient` |
| `/device_authorization` | `POST` | p99 ≤ 300 ms | `http_request_duration_seconds` | `DeviceAuthorization` |
| `/api/auth/federation/oidc/callback`, `/api/auth/federation/saml/callback` | `GET`, `POST` | p95 ≤ 2 s | `http_request_duration_seconds` | 上流とのネットワーク交換を含む |

旧目標にあった「セッション一覧の先頭ページの p95 ≤ 100 ms」は Specification target に復元しない。現行の `route`、`method`、`status_code` ラベルでは `/api/account/v1/sessions` の先頭ページと後続ページを区別できず、測定不能な保証になるためである。低カーディナリティの測定方法を定めた時点で再導入を判断する。

### Non-5xx ratio

| Endpoint population | Target | Metric |
| --- | --- | --- |
| `/api/auth/login`, `/authorize`, `/par`, `/token`, `/revoke`, `/userinfo`, `/register`, `/device_authorization` | ≥ 99.9% | `http_requests_total` |
| `/introspect`, `/.well-known/openid-configuration`, `/.well-known/oauth-authorization-server`, `/jwks`, `/realms/{tenant_id}/jwks` | ≥ 99.99% | `http_requests_total` |

外部連携ログインのコールバックにはレイテンシー目標だけを復元し、非 5xx 比率の新しい数値は設けない。旧 SCL に対応する値がなく、上流 IdP の拒否や障害を IdMagic の失敗と区別する母集団も未定義だからである。

### Availability

| Population | Target | Window and budgeting | Metric |
| --- | --- | --- | --- |
| Non-5xx ratio の対象となる OAuth2/OIDC エンドポイント全体 | ≥ 99.9% | 30 日、5 分の時間区分 | `http_requests_total` と Prometheus のスクレイプ状態 |
| `/token` | ≥ 99.95% | 30 日、5 分の時間区分 | `http_requests_total` と Prometheus のスクレイプ状態 |

### Capacity acceptance

フリートは、[Reference operating profile](#reference-operating-profile) の全データを投入し、[Peak request profile](#peak-request-profile) の要求を同時に 15 分間ウォームアップした後、60 分間処理して次の Specification target を満たす。

| Endpoint | Required rate | Required objectives |
| --- | --- | --- |
| `/token` | 5,000 rps | p99 ≤ 300 ms、非 5xx 比率 ≥ 99.9% |
| `/authorize` | 1,000 rps | p99 ≤ 500 ms、非 5xx 比率 ≥ 99.9% |
| `/introspect` | 20,000 rps | p99 ≤ 50 ms、非 5xx 比率 ≥ 99.99% |

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

## Sizing rules

API レプリカの必要数は、エンドポイント `e` ごとに `ceil(API peak_e ÷ measured sustainable rate per replica_e × 1.5)` を計算した最大値とする。1 レプリカ当たりの持続処理能力は、参照運用プロファイルのデータと混合負荷の下で対応するレイテンシーと非 5xx 比率を満たした Measurement だけを使う。

実測前の Planning assumption は、`/token` が 250 rps、`/authorize` が 100 rps、`/introspect` が 1,000 rps である。この仮定では API は `ceil(max(5000/250, 1000/100, 20000/1000) × 1.5) = 30` レプリカとなるが、達成済みの構成を示す数値ではない。

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
