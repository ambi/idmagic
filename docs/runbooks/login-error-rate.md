# ログインのエラー率

## 発火条件

`LoginErrorRateBudgetBurn`。`/api/auth/login` の 5xx 比率が 5 分にわたり 0.1% を超えたときに発火する。燃やしているのは [SLO-PRIMARY-ERRORS](../capacity.md#non-5xx-ratio) の error budget である。

**資格情報の誤りでは発火しない。** パスワード不一致、MFA 失敗、無効化されたユーザーの拒否はいずれも 4xx であり、仕様どおりの振る舞いである（[observability.md](../observability.md) の「予期された業務上の失敗を ERROR にしない」）。発火しているなら、判定そのものができていない。

## 最初に確認すること

1. **拒否の内訳を見る。** 5xx と、フェイルクローズによる拒否を分ける。

   ```promql
   sum by (outcome, reason_class, method) (rate(authn_login_attempts_total[5m]))
   sum by (policy, outcome) (rate(authn_login_throttle_total[5m]))
   ```

   `reason_class` が資格情報の誤りに偏っているなら、これは攻撃であって障害ではない。[login-throttle-hit-ratio.md](login-throttle-hit-ratio.md) へ移る。

2. **フェイルクローズしている依存先を特定する。** ログイン経路は次をいずれも必須とし、到達できなければ**拒否する**。到達不能がそのまま拒否として現れる。

   | 依存先 | 失うと何が起きるか |
   |---|---|
   | PostgreSQL | ログインスロットルの状態を確認できず拒否（[deployment.md](../deployment.md)） |
   | レート制限ストア | 全ポリシーでフェイルクローズに拒否 |
   | DataKeys の提供元 | TOTP シードを復号できず MFA が通らない |

3. **`/readyz?verbose`。** 依存ごとに `healthy` / `degraded` / `unavailable` を返す。

4. **直近の配備、テナント設定、認証ポリシーの変更。**

## 緩和

- 依存先の障害なら、その依存先を復旧させる。**フェイルクローズを迂回する設定変更を緩和策にしない。** スロットルを無効化すればログインは通るが、同時に総当たりへの防御も落ちる。
- 直近のリリースと相関するなら後退する。
- 単一レプリカの `memory` ランタイムで運用していないことを確認する。共有状態を失うと閾値が実質的に緩む（[deployment.md](../deployment.md)）。

## 確認

5xx 比率が 10 分間基準へ戻り、`authn_login_attempts_total{outcome="success"}` が発火前の水準に戻ることを確認する。代表テナントで実際にログインを 1 回通す。

## エスカレーション

15 分以内に軽減できなければ SEV-1 を宣言する。**ログインが通らない状態は、既存セッションが生きている間だけ影響が見えにくい。** セッションの期限が切れ始めると一斉に顕在化するため、影響が小さく見えても宣言を遅らせない。
