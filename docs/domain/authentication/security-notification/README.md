# セキュリティ通知

アカウントに起きたセキュリティ上の変化を本人へ知らせるメールの契機、宛先、受信設定を扱う。
メールの配送そのものは `EmailSender` ポートが扱う。
コードの機能スライスは `backend/authentication/securitynotification` である。

| 文書 | 内容 |
|---|---|
| [セキュリティ通知の設計判断](decisions.md) | 設計判断 |
| [セキュリティ通知の内部設計](internals.md) | 機構の説明 |
| [セキュリティ通知のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
