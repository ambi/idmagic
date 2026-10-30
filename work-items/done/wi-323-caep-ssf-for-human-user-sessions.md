---
status: completed
---

# 人間の User の失効を SharedSignals で即時に反映し、外部と相互に伝える

User の失効エポックを追加し、無効化、論理削除、物理削除、本人と管理者のセッションの失効、パスワードと MFA の変更、refresh の再利用検出で発行済みのトークンを無効にして、購読するストリームへ `session-revoked` を投影する REQ-SHAREDSIGNALS-012 を追加した。
受信の主体を User へ広げ、エポックを進めてログインセッションを終わらせる（REQ-SHAREDSIGNALS-010）。
失効の判定は `/introspect`、Bearer、交換の入力、refresh に入り、User がいないか `Active` でないトークンも無効になる（REQ-OAUTH2-047）。
追加した TypeSpec は `UserRevocationEpoch`、`UserAccessRevoked` で、`RevocationReason` と `RefreshTokenReuseDetected` を変えた。表は `user_revocation_epochs` である。

記録は古かったので現状に合わせて書き直し、通常のログアウト、アイドル、絶対期限ではエポックを進めないと決めた。エポックが User 単位で、一つの端末の出来事が全端末を止めるためである。
Authentication と OAuth2 のイベントは、依存の循環を避けて組み立て地点が `UserRevocation` へ変換する。

テストは実装の後に書いたので、テストが実装より先行した順序ではない。

残るリスクは次のとおりである。
Authentication と OAuth2 のイベントからの反応の失敗はログだけに残り、そのときトークンは期限まで有効である（IdManagement の経路は呼び出し元へ誤りを返す）。
判定の読み取りが User のトークンごとに増え、負荷は実測していない。
失効と同じ秒に発行したトークンは無効になる。
受信した SET は CAEP の種類を語り分けず、すべて失効として扱う。
