---
status: completed
archived_from: 0c1aadf654f5f452bb328d1cb5158239bfaffd4f
---

# 未解析の HTTP 操作を契約テストで検証する

未解析の 282 操作について、正式な HTTP 経路で成功と拒否の応答を観測するテストと操作参照を整備した。
証拠ランナーは一方の成功で他方の未実行、スキップ、失敗を隠さない。
`spec-diff b217be43b` が示した送受信モデルと既存の拒否の宣言を実際の契約へ合わせ、WS-Federation の応答形式と状態コード、およびメール変更の既存の step-up 拒否の仕様例を補った。
承認済みのクォータ認可エラー、SAML LogoutResponse の署名後の整形、POST ログアウトの form state の読み取りを既存要件へ戻した。

全文は `git show 0c1aadf654f5f452bb328d1cb5158239bfaffd4f:<このファイルのパス>` で読む。
