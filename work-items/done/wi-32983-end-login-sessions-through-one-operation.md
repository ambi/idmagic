---
status: completed
---

# ログインセッションの終了を一つの操作に集め、`active_sessions` の使用量を実際の件数に保つ

`spec-diff` は REQ-AUTHENTICATION-045 と REQ-TENANCY-051 の追加を示す。
ログインセッションを失効させる経路（本人、管理者、OIDC のログアウト、SAML の SLO、WS-Federation のサインアウト、主体の無効化）は、すべて一つの終了の操作を通り、保存先の失効が実際に失効させた一度だけ、期限内なら `active_sessions` を減らして `SessionEnded` を発行する。
期限切れ、User の完全削除、受信した SET はセッションごとの減算を持たず、保持期限バッチが有効なセッション（未失効かつ期限内、確認待ちを含む）の件数で `active_sessions` を置き換えて回収する。
掃除は期限切れの 90 日後なので、削除時に減らす方式では障害が直らず、REQ-TENANCY-049 と同じ再集計を採った。依存の向きは組み立て地点が結ぶ形で、モジュール間の依存は増やしていない。

検証の範囲と限界：SLO の経路は `SessionManager.Revoke` の単体、再集計は `RunRetentionSweepOnce` のメモリ構成で観測し、件数と失効の SQL は PostgreSQL の共有契約で確かめた。SAML と WS-Federation の HTTP ハンドラーを通す E2E と race 検査、変異テストは行っていない。
残るリスク：`retention-sweep` が止まると期限切れの分が使用量に残り続ける。毎時の既定では、使用量は最大で直近約 2 時間のログインの件数まで膨らむ。`SessionStore.Revoke` を `authentication/internal` の外から呼べなくする狭め方は、ポートの包みを分ける構造変更として保留した。
