# 容量設計

## 目的

本書は、システムを構成するための負荷、データ量、処理能力の前提と算出規則を所有する。サービス目標、測定境界、容量受入れ目標の正本は [品質要求](../../requirements/quality.md) であり、本書は `SLO-*` と `CAP-*` の値を再掲しない。

本書の数値は、未実測の **Planning assumption** である。実測値を採用する場合は、日付、ソース版、実行環境、データ分布、負荷構成、試験時間、結果の保存先を伴う **Measurement** として記録する。実測が前提を下回っても、品質要求を暗黙に引き下げてはならない。

## 参照運用プロファイル

参照運用プロファイルは容量を算出するための設計入力であり、すべての配備に要求する最小構成でも、プロダクトが超過を拒否するハード上限でもない。

### テナントと活動量の分布

| テナント区分 | テナント数 | テナント当たり利用者数 | 利用者数 | 区分 |
| --- | ---: | ---: | ---: | --- |
| Small | 90,000 | 20 | 1,800,000 | Planning assumption |
| Medium | 9,000 | 500 | 4,500,000 | Planning assumption |
| Large | 900 | 3,000 | 2,700,000 | Planning assumption |
| Very large | 100 | 10,000 | 1,000,000 | Planning assumption |
| Total | 100,000 | — | 10,000,000 | Specification target |

この分布では中央値が 20 ユーザー、p90 が 500 ユーザー、p99 が 3,000 ユーザー、最大が 10,000 ユーザーとなる。10,000 ユーザーを超える単一テナントには、この分布とは別の大規模テナント向け性能検証を要する。

| 活動量 | 値 | 区分 |
| --- | ---: | --- |
| Monthly active users | 4,000,000 | Planning assumption（全ユーザーの 40%） |
| Daily active users | 1,000,000 | Planning assumption（全ユーザーの 10%） |
| Active tenants in the busiest 5-minute interval | 20,000 | Planning assumption（全テナントの 20%） |
| Concurrent valid browser sessions at peak | 200,000 | Planning assumption（全ユーザーの 2%） |
| New browser sessions per day | 500,000 | Planning assumption |

### オブジェクトと保持量

| オブジェクト | 件数または到着率 | 保持の前提 | 一行当たりの想定バイト数 |
| --- | ---: | --- | ---: |
| Tenant | 100,000 | 削除まで | 8 KiB |
| User | 10,000,000 | 削除と匿名化の方針に従う | 4 KiB |
| Application | 500,000 | 削除まで | 8 KiB |
| OAuth2 client | 1,000,000 | 削除まで | 4 KiB |
| Group membership | 50,000,000 | 所属解除まで | 256 B |
| Authentication session row | 500,000 / day | 失効または期限切れ後を含む 90 日 | 1 KiB |
| Refresh token row | 2,000,000 / day | 絶対期限 30 日 | 512 B |
| Authentication event | 1,000,000 / day | 容量見積もりでは 365 日 | 1 KiB |
| Audit event | 5,000,000 / day | 2,555 日 | 1 KiB |

監査イベントの検索属性を 1 件当たり 6 行、1 行 128 B と仮定すると、本体と合わせた物理ストレージの初期予算は約 32 TiB となる。物理予算は `Σ(保持行数 × 実測平均行バイト数) × 2.5 + 30 日分の物理成長量` 以上とする。係数 2.5 はインデックス、MVCC、保守時の空き容量を含み、バックアップ、WAL の別保管、読み取りレプリカは含めない。保持期間の正本は [データライフサイクル設計](../data/lifecycle.md) と各 Context の仕様である。

## ピークリクエストプロファイル

次の値は、同じ最繁 15 分に API へ到達するリクエスト率の Planning assumption である。キャッシュ可能な公開文書は利用者側のリクエスト率と API 到達率を分け、容量算出には API 到達率を使う。

| エンドポイント群 | クライアント側ピーク | API 到達ピーク | 前提 |
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
| Discovery | 20,000 rps | 2,000 rps | gateway または CDN のヒット率を 90% と仮定 |
| JWKS | 50,000 rps | 5,000 rps | gateway または CDN のヒット率を 90% と仮定 |

Discovery と JWKS のヒット率は保証値ではない。コールドキャッシュの容量検証では、API 到達率が表の 10 倍になる条件を含める。

### Worker の実行レーン

| 実行レーン | ピーク到着率 | ハンドラー時間の p95 | 安全係数 1.5 で必要な枠数 |
| --- | ---: | ---: | ---: |
| `latency_sensitive` | 50 jobs/s | 200 ms | 15 |
| `default` | 20 jobs/s | 1 s | 30 |
| `bulk` | 2 jobs/s | 10 s | 30 |

### 非プロトコルリクエストプロファイル

管理コンソール、管理 API 自動化、ポータル、SCIM、Shared Signals は、認証・プロトコル系と処理特性が異なるため別に見積もる。現時点の値はすべて Planning assumption であり、Measurement はない。

| 種別 | 通常時 | 最繁時 | 集中実行時 | 支配的な不確実性 |
| --- | ---: | ---: | ---: | --- |
| Administration console | 50 rps | 300 rps | — | 活動中の管理者割合とテナント当たりの管理者数 |
| Administration API automation | 140 rps | 210 rps | 1,100 rps | 同期間隔と一回当たりのリクエスト数 |
| Account portal | 150 rps | 900 rps | — | ログイン後の訪問率と一訪問当たりのリクエスト数 |
| SCIM | 13 rps | 50 rps | 110 rps | 有効テナント率と全同期の規模 |
| Shared Signals | 30 rps | 90 rps | 300 rps | 有効テナント率とイベント発生率 |

最繁 15 分の合計は中央値で 1,550 rps、集中実行時の値を単純に足した上界は 2,710 rps である。認証・プロトコル系の 41,250 rps に対して到達率では 4–7% だが、一リクエスト当たりの費用が異なるため資源比率には読み替えない。管理 API の同期規模と SCIM 全同期を最初の Measurement 対象にする。

### ドメインイベントの費用

ドメインイベントの監査記録は発行元のリクエストまたはジョブの中で追記するため、発行一件当たり本体一行と検索属性六行のデータベース書き込み、5 ms の追加時間を Planning assumption として発行元の負荷へ含める。再生キューは存在しない。実行時の配信方式は [実行時アーキテクチャ](../../architecture/runtime.md#通信) が所有する。

## 構成算出規則

API レプリカの必要数は、エンドポイント `e` ごとに `ceil(API peak_e ÷ measured sustainable rate per replica_e × 1.5)` を計算した最大値とする。一レプリカ当たりの持続処理能力は、参照運用プロファイルのデータと混合負荷の下で該当する `SLO-*` を満たした Measurement だけを使う。

実測前は `/token` 250 rps、`/authorize` 100 rps、`/introspect` 1,000 rps を仮定する。この場合の API は 30 レプリカだが、達成済みの構成を示す値ではない。管理系、ポータル系、SCIM、Shared Signals を含む混合負荷で測定して置き換える。

実行レーン `l` の必要なワーカー枠は `ceil(peak arrival_l × p95 handler time_l × 1.5)` とする。既定の一プロセス四枠では `latency_sensitive` 四レプリカ、`default` 八レプリカ、`bulk` 八レプリカとなる。レーン間で枠を融通せず、`bulk` の滞留を理由に `latency_sensitive` の枠を減らさない。

PostgreSQL の論理接続予算は `API replicas × API pool limit + worker replicas × worker pool limit + concurrent batches × batch pool limit + operator reserve` とする。実測前は API 一レプリカ 16 接続、Worker 一レプリカ 8 接続、同時 Batch 四個で各 4 接続、運用予約 64 接続とし、参照構成では 720 接続となる。論理接続予算は利用可能接続数の 70% 以下に保つため、この構成には少なくとも 1,029 接続の容量を要する。

ストレージと接続数は 70% を継続的な上限として設計し、残りを障害時の偏り、保守、成長に残す。レプリカ上限を増やす前に PostgreSQL の接続予算を再計算する。実際の自動スケール条件は [スケーリングと過負荷の設計](scaling.md) が所有する。

## 縮退順序

容量超過時も、テナント境界、認証と認可の検証、再送防止、流量制限、監査イベントの完全性を弱めない。状態を確認できないリクエストを成功扱いにすること、監査イベントを黙って破棄すること、テナントをまたいでキャッシュまたは接続枠を共有することは縮退手段に含めない。

1. `bulk` レーンの新規取得と保守 Batch を遅延させる。
2. `default` レーンの新規取得を遅延させ、`latency_sensitive` の専用枠を維持する。
3. 管理用の集計、エクスポート、再同期など、対話的な認証に不要な高コスト処理を 429 または 503 で拒否する。
4. 動的クライアント登録など、既存セッションの認証とトークン処理に不要な書き込みを拒否する。
5. `/authorize`、`/token`、`/introspect`、ログインを受け付けられない場合は、状態を部分的に更新せず 429 または 503 で拒否する。

ステージ一と二は Worker の実行レーンが、ステージ三以降は API の入場制御が実現する。経路の優先度は [System Context の判断](../../contexts/system/decisions.md#load-shedding-by-priority-class) が所有する。高可用性と障害時の遷移は [可用性設計](../reliability/availability.md) が所有する。
