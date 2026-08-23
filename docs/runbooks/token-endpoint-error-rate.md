# `/token` のエラー率

## 発火条件

`TokenErrorRateBudgetBurn`。`/token` の 5xx 比率が 5 分にわたり 0.1% を超えたときに発火する。燃やしているのは [SLO-PRIMARY-ERRORS](../capacity.md#non-5xx-ratio) の error budget である。

**4xx では発火しない。** クライアントの不正な要求、失効したトークン、スコープ不足はいずれも仕様どおりの応答であり、この経路の失敗ではない。発火しているなら、IdMagic 側かその依存先が要求を処理できていない。

## 最初に確認すること

1. **範囲を絞る。** `grant_type` 別と `status_code` 別に分ける。

   ```promql
   sum by (grant_type, outcome) (rate(oauth2_token_issuance_total[5m]))
   sum by (status_code) (rate(http_requests_total{route=~".*/token"}[5m]))
   ```

   1 つの `grant_type` に偏っているなら、その交付方式の依存先を疑う。`urn:ietf:params:oauth:grant-type:token-exchange` だけなら WorkloadIdentity の外部アテステーション検証、`refresh_token` だけなら PostgreSQL の書き込みである。全体に均等なら共通経路（PostgreSQL、署名鍵）を疑う。

2. **署名鍵に到達できるか。** トークンは署名できなければ発行できない。`KeyProvider` が `VaultTransit` なら OpenBao の到達性と封印状態を確認する。`GET /api/admin/data-keys/health` は DEK 側の提供元到達性を返す（`system_admin` 限定）。

3. **PostgreSQL。** `/readyz?verbose` が依存ごとの状態を返す。`503` なら API は自ら受付不可を宣言している。

4. **直近の配備と設定変更。** リリース、テナント設定、鍵のローテーションのいずれかが直前にあるか。

## 緩和

- 直近のリリースと相関するなら後退する。手順は `mise run rollback-k8s idmagic-api`（[infra/README.md](../../infra/README.md)）。
- 署名鍵の提供元が 1 リージョンで落ちているなら、健全なリージョンへ退避する。**鍵素材を平文へ退避してはならない。** フェイルクローズのまま発行を止めるほうが正しい。
- 容量が原因なら [Degradation order](../capacity.md#degradation-order) に従う。`bulk` レーンと保守バッチを先に止め、`/token` の枠を残す。**順序を飛ばして `/token` を落とさない。**
- レート制限のストアへ到達できずフェイルクローズで拒否しているなら、それは設計どおりの拒否である。ストアを復旧させる。閾値を上げて回避しない。

## 確認

5xx 比率が 10 分間基準へ戻り、`oauth2_token_issuance_total{outcome="success"}` が発火前の水準に戻ることを確認する。発行済みトークンが復旧後の JWKS で検証できることも確かめる（`kid` が消えていないか）。

## エスカレーション

15 分以内に軽減できなければ指揮者を任命し SEV-1 を宣言する。**テナント境界の逸脱が疑われる場合は、軽減より先に該当経路を遮断する**（[authorization.md](../authorization.md)）。
