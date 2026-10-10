# Feature: 運用の例

## Rule: REQ-SYSTEM-001 Operator は分離された運用資産で SLO を検証する

### Background:

- Given API、フロントエンドゲートウェイ、Worker は個別の実行単位としてデプロイされる
- And `MetricsExposition` の公開範囲は管理ネットワークに制限される
- And OAuth2/OIDC のサービス目標、母集団、時間窓、除外条件は `docs/requirements/quality.md` に定められている
- And 各サービス目標は `docs/design/observability/monitoring.md` の HTTP RED メトリクスと Prometheus のスクレイプ状態に対応づけられている

### Example: EX-SYSTEM-001-01 通常経路

- When Operator が環境のオーバーレイを選んで運用マニフェストを適用する
- Then API の生存、受付可否、起動完了の各プローブは、それぞれ `LivenessProbe`、`ReadinessProbe`、`StartupProbe` を呼ぶ
- Then Prometheus が `MetricsExposition` をスクレイプし、定められた母集団と時間窓で OAuth2/OIDC の可用性、レイテンシー、非 5xx 比率を表示および評価する

### Example: EX-SYSTEM-001-02 PostgreSQL へ到達できない

- When Operator が環境のオーバーレイを選んで運用マニフェストを適用する
- But PostgreSQL へ到達できない
- Then `ReadinessProbe` は `unavailable` を返し、API は新規トラフィックを受けない
- And `LivenessProbe` は `healthy` を維持し、依存障害だけでは再起動しない

### Example: EX-SYSTEM-001-03 Prometheus Operator が導入されていない

- When Operator が環境のオーバーレイを選んで運用マニフェストを適用する
- But Prometheus Operator が導入されていない
- Then `ServiceMonitor` は適用対象から外し、標準の Prometheus スクレイプ設定で `MetricsExposition` を収集する

## Rule: REQ-SYSTEM-002 オーケストレーション用プローブはプロセスのライフサイクルと依存先の状態を区別する

### Example: EX-SYSTEM-002-01 通常経路

- When Operator が初期化完了後に生存、受付可否、起動完了の各プローブを呼ぶ
- Then すべて `200 healthy` を返す

### Scenario Outline: 条件ごとの結果

- When Operator が初期化完了後に生存、受付可否、起動完了の各プローブを呼ぶ
- But <condition>
- Then <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-SYSTEM-002-02 | 初期化中またはグレースフルドレイン中である | 生存確認は `200 healthy` を維持する | 受付可否または起動完了の確認は 503 を返す |
  | EX-SYSTEM-002-03 | 設定済みの永続化依存先へ到達できない | 受付可否の確認は `503 unavailable` を返す | 生存確認は `200 healthy` を維持する |

## Rule: REQ-SYSTEM-012 PostgreSQL クエリの期限は結果の読み取り完了まで維持される

### Example: EX-SYSTEM-012-01 通常経路

- Given PostgreSQL の永続化とクエリタイムアウトが設定されている
- When System が共通の永続化アダプターで単一レコードまたは複数レコードを返すクエリを開始する
- Then クエリが `Row` または `Rows` を返す
- When 呼び出し側が期限内に `Scan` または反復処理を完了する
- Then 結果が `context canceled` にならず返され、接続が解放される

### Scenario Outline: 条件ごとの結果

- Given PostgreSQL の永続化とクエリタイムアウトが設定されている
- When System が共通の永続化アダプターで単一レコードまたは複数レコードを返すクエリを開始する
- Then クエリが `Row` または `Rows` を返す
- When 呼び出し側が期限内に `Scan` または反復処理を完了する
- But <condition>
- Then <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-SYSTEM-012-02 | 結果の読み取り中にクエリタイムアウトの期限へ到達する | 読み取りは `deadline exceeded` で中断される | 結果を閉じると接続とタイムアウトのリソースが解放される |
  | EX-SYSTEM-012-03 | 単一レコードを返すクエリに該当するレコードが存在しない | `Scan` は `no rows` を返す | `no rows` は正常なクエリ結果として扱われ、サーキットブレーカーの失敗率を増加させない |
