---
status: completed
---

# 監査記録の保持について、決定と実装が食い違っている

監査の記録の保持期間を、7 年ではなく種類ごとの有限の期間（デフォルト 365 日、認証の失敗 30 日、集計、MFA、セッションの終了 90 日）に決めた。
7 年の根拠とされた GDPR 第 30 条はイベントの記録の保存年数を定めず、365 日は PCI DSS v4.0 の 10.5.1 を満たす。
判断と代替案は `docs/modules/audit/decisions.md` の「監査の記録を種類ごとの有限の期間だけ保持する」に置いた。

`spec-diff` の差分は、REQ-AUDIT-013（新しい機能仕様 `docs/modules/audit/features/event-retention.md`）の追加と、`AdminAuditEventResponse` の doc から 7 年保持の記述を除いた変更である。
振る舞いの変更は一つで、なりすましの記録を無期限に残すのをやめ、デフォルトの 365 日で消すようにした。

保持の方針と掃引は、Authentication から Audit（`audit.Module.PurgeExpiredEvents`）へ移した。
構築関数が保存先を掃引へ渡すので、型アサーションの失敗で掃引が黙って飛ぶ経路はない。
Authentication には自分の記録（集計の窓、ログインセッション、既知の端末）の掃除だけを残し、設定する経路のなかった `MaxDays` を除いた。
移動で、なりすましのイベントが本番で発行されていないこと（[[wi-93-admin-user-impersonation]]）が検査に現れたので、`tools/check/event-emission-debt.json` に載せた。
保持期間を設定で変える手段は [[wi-61776-configurable-audit-retention-periods]] が判断する。
