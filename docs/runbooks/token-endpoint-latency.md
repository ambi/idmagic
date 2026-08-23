# `/token` のレイテンシー

## 発火条件

`TokenLatencyBudgetBurn`。`/token` の p99 が 5 分にわたり 300 ms を超えたときに発火する。燃やしているのは [SLO-TOKEN-LATENCY](../capacity.md#latency) の error budget である。

エラー率のアラートが同時に出ているなら、先に [token-endpoint-error-rate.md](token-endpoint-error-rate.md) を見る。**失敗が遅延として現れているだけの場合があり、そのときは遅さを追っても原因に届かない。**

## 最初に確認すること

1. **取得と処理を分ける。** 遅いのが IdMagic の処理なのか、依存先の待ちなのかを先に決める。

   ```promql
   histogram_quantile(0.99, sum by (le, grant_type) (rate(oauth2_token_issuance_duration_seconds_bucket[5m])))
   sum by (route) (rate(http_requests_in_flight{route=~".*/token"}[5m]))
   ```

   `http_requests_in_flight` が伸びているなら滞留、伸びずに個々が遅いなら処理そのものである。

2. **PostgreSQL の接続枠。** [Sizing rules](../capacity.md#sizing-rules) の接続予算を超えると、待ちが遅延として現れる。プールの待機数と、レプリカ数 × プール上限が物理容量の 70% 以下に収まっているかを確認する。

3. **署名の経路。** `KeyProvider` が `VaultTransit` なら、署名はネットワーク越しの往復になる。OpenBao 側のレイテンシーを確認する。ローカル鍵に比べて桁が変わる。

4. **要求構成の変化。** 特定のテナントやクライアントからの流量が跳ねていないか。`endpoint_rate_limit_total{policy="token",outcome="limited"}` が増えていれば、流量制限は効いている。

## 緩和

- 滞留なら API レプリカを増やす。必要数の算出は [Sizing rules](../capacity.md#sizing-rules) にある。**接続予算を同時に見直さないと、レプリカを足して PostgreSQL 側で詰まる。**
- 特定テナントの急増が原因なら、そのテナントの流量制限を絞る。全体の閾値を下げない。
- 直近のリリースと相関するなら後退する。
- 容量不足なら [Degradation order](../capacity.md#degradation-order) に従う。管理用の集計、エクスポート、再同期を先に止める。

## 確認

p99 が 10 分間 300 ms を下回り、`http_requests_in_flight` が発火前の水準に戻ることを確認する。

## エスカレーション

30 分以内に軽減できなければ SEV-2 を宣言する。ログインのレイテンシーまたはエラー率のアラートが同時に出ているなら SEV-1 とする。**認証の入口と発行の両方が劣化している状態は、利用者から見れば全断と区別がつかない。**
