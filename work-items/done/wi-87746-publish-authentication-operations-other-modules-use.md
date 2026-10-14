---
status: completed
archived_from: c79c599788f64c5f59cd8e11394050a303bf10cb
---

# ほかのモジュールが使う Authentication の判定と操作を公開する

`mise run spec-diff` は規範の差分なしを報告する。
ほかのモジュールが使う Authentication の判定を、入力だけで決まる計算として各機能の `domain` へ移した（ACR の語彙と判定、step-up の判定、セッションの cookie の名前とセンチネルエラー、パスワードの方針の判定と既定値）。`usecases` には定数、型、センチネルエラーの別名だけを残し、Authentication の内部の参照と `errors.Is` の判定は変えていない。
SAML と WS-Federation は、自分で定義した `SessionRevoker` のポートでログアウトのセッションを失効させる。
IdManagement は、`trusteddevice/ports.UserDeviceRevoker` で無効化に伴う信頼済みデバイスの失効を行い、返されたイベントを自分の発行先で発行する。
境界の負債の台帳から、本項目の 10 件と、ACR の判定の移動で解消した OAuth2 の 1 件を消し、台帳は 117 件から 106 件になった。
認証の判定、HTTP の応答、セッションの状態、発行するイベントは変えていない。

全文は `git show c79c599788f64c5f59cd8e11394050a303bf10cb:<このファイルのパス>` で読む。
