---
status: completed
archived_from: a29e71f77ee54b71a4309bf483ebbe77139c9ef5
---

# APIエラーメッセージを汎用英語から具体的な英語に置き換え、EnglishErrorTextを削除する

`WriteBrowserError`、`NewOAuthError`、OAuth 認可 redirect、SCIM error、SAML / WS-Federation の拒否本文へ渡す日本語リテラルを具体的な英語へ置換した。
`err.Error()` 経由で外部へ露出していた監査検索、認可トランザクション、動的な認可拒否理由の日本語発生源も英語化した。
`TestAPIErrorLiteralTextIsEnglish` を追加し、既知の API / protocol error sink に日本語リテラルが再混入した場合に検出する。

全文は `git show a29e71f77ee54b71a4309bf483ebbe77139c9ef5:<このファイルのパス>` で読む。
