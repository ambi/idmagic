# ログインのレイテンシー

## 発火条件

`LoginLatencyBudgetBurn`。`/api/auth/login` の p99 が 5 分にわたり 300 ms を超えたときに発火する。燃やしているのは [SLO-LOGIN-LATENCY](../capacity.md#latency) の error budget である。

## 最初に確認すること

1. **パスワード検証そのものを疑う。** ログインは Argon2id の計算を含み、**これは意図的に高価である**（`backend/shared/security` の `passwords_argon2id`）。パラメーターを上げる変更が直前に入っていないか確認する。上げたなら、遅いのは障害ではなく設定の帰結である。

2. **スロットルの直列化。** ログインスロットルは共有カウンターを `SELECT ... FOR UPDATE` で直列化して更新する（[deployment.md](../deployment.md)）。同一アカウントまたは同一 IP へ試行が集中すると、この行がボトルネックになる。

   ```promql
   sum by (policy, outcome) (rate(authn_login_throttle_total[5m]))
   ```

   `outcome="limited"` が跳ねているなら、遅さの原因は攻撃側の負荷である。[login-throttle-hit-ratio.md](login-throttle-hit-ratio.md) へ移る。

3. **MFA の経路。** TOTP は DataKeys の DEK で復号したシードを使う。提供元がネットワーク越し（OpenBao）なら、その往復が加わる。`GET /api/admin/data-keys/health` で到達性を確認する。

4. **PostgreSQL の接続枠。** 待ちが遅延として現れる。[Sizing rules](../capacity.md#sizing-rules) の接続予算を確認する。

## 緩和

- 特定アカウントまたは IP への集中が原因なら、そこを遮断する。全体のスロットル閾値を緩めない。**緩めれば遅延は消えるが、総当たりへの防御も同時に消える。**
- 容量が原因なら [Degradation order](../capacity.md#degradation-order) に従う。**ログインは最後まで守る対象である。** 管理用の集計とエクスポートを先に止める。
- 直近のリリースと相関するなら後退する。Argon2id のパラメーター変更が含まれていないかを特に確認する。

## 確認

p99 が 10 分間 300 ms を下回ることを確認する。代表テナントで実際にログインを 1 回通し、体感で待たされないことも見る。

## エスカレーション

30 分以内に軽減できなければ SEV-2 を宣言する。`/token` のアラートが同時に出ているなら SEV-1 とする。
