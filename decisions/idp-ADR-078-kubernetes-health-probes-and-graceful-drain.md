# idp-ADR-078: 依存ヘルスを検査する liveness/readiness/startup probe と SIGTERM 時の接続ドレインを整備する

## ステータス
採用。`scl.yaml` の `objectives.HealthProbe` / `objectives.ReadinessCheck` / `objectives.GracefulDrain` と該当ソースに反映。

## コンテキスト
現在の idmagic のヘルスチェック `/health` は、起動構成ラベル（persistence / event_sink / observability / authzen）をそのまま JSON で返すだけで、PostgreSQL や Valkey などの依存の実際の接続性を検査していない。
このエンドポイントを Kubernetes の liveness と readiness プローブに流用すると、DB 瞬断で Pod が不必要に再起動ループに陥ったり、接続不全の Pod へトラフィックがルーティングされ続けたりする。
そのため、Kubernetes の標準に合わせ、(1) 生存性 `/livez`、(2) トラフィック受入可否 `/readyz`、(3) 起動完了 `/startupz` を分離し、SIGTERM 受信時に readiness プローブを即 unready に落として接続ドレインを行う仕組みが必要である。

## 決定
1. **プローブの分離**:
   - `/livez`（Liveness）: 自己回復不能なデッドロック等のみで 500/503 エラーを返す。一時的な DB/Valkey 瞬断では 200 OK を返す。
   - `/readyz`（Readiness）: PostgreSQL・Valkey 等の必須依存への到達性（Ping）を並列・短タイムアウト（既定 1s）で実行し、すべて接続可能なら 200 OK を返す。接続障害時は 503 を返す。
   - `/startupz`（Startup）: アプリケーション初期化完了（シードデータ確認など）後に 200 OK を返す。
   - 既存 `/health` は互換性のために起動構成ラベルを返すエンドポイントとして維持する。
2. **Readiness の詳細表示**:
   - `/readyz?verbose` クエリパラメータが指定された場合は、各依存関係（`postgres` や `valkey`）の状態語彙（`healthy` / `degraded` / `unavailable`）を含めた詳細 JSON を返す。
3. **Graceful Drain の実装**:
   - SIGTERM または SIGINT を受信した際、グローバルなシャットダウンフラグをセットし、`/readyz` プローブが即座に `503 Service Unavailable`（`unavailable`）を返すようにする。
   - その後、ロードバランサが対象 Pod をトラフィックルーティングから外す時間を確保するため、ドレイン猶予期間（既定 5秒、`DRAIN_GRACE_PERIOD_SECONDS` 環境変数でカスタマイズ可能）だけ待機してから HTTP サーバのシャットダウン処理（`e.Shutdown`）を開始する。

## 却下した代替案
- **`/health` エンドポイントを readiness と liveness で共用し続ける案**:
  - 前述の通り、一時的な DB の瞬断がコンテナの不要な再起動ループを引き起こし、かつ復旧を遅らせるため却下。

## 影響
- `spec/contexts/system.yaml` に `HealthProbe`、`ReadinessCheck`、`GracefulDrain` の objectives を追加。
- `internal/bootstrap/server.go` でのシグナルハンドリングと graceful shutdown 協調処理の追加。
- `internal/shared/adapters/http/server/health_handler.go` 周辺でのプローブ実装と、`Deps` への `DbPing` / `ValkeyPing` の注入。
