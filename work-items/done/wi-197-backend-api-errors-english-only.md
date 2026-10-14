---
status: completed
archived_from: 0c08fa8df775732c7219e1c7d5ff81f9fcd8c8fa
---

# バックエンド API エラーの利用者向けメッセージを英語に統一する

SCL に横断的な BackendErrorText 契約を追加し、共通 HTTP、OAuth/OIDC、SCIM、SAML、WS-Federation の API エラー本文を英語へ正規化した。
日本語を含む本文は安定した英語フォールバックへ置換し、error code と HTTP status は維持する。

全文は `git show 0c08fa8df775732c7219e1c7d5ff81f9fcd8c8fa:<このファイルのパス>` で読む。
