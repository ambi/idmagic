---
status: completed
archived_from: dc0961d035553d99cd2b9ce5a472240293d05838
---

# 連携エンドポイント参照とSAML IdP profile変更を分離する

- Vite/Caddy gatewayへSAML、WS-Federation、WS-Trustの公開protocol routeを追加した。
- 連携エンドポイント画面をread-onlyのprotocol分類へ整理し、SAMLはDefaultを含む各profile内にentityID、metadata、SSO、SLO、署名証明書をまとめた。
- SAML IdP profileの一覧・詳細・作成・編集を専用routeへ分離した。
- 一般用語の「エンドポイント」「ベースURL」を日本語表記へ統一し、protocol仕様上の正式なEndpoint名は英語表記を維持した。

全文は `git show dc0961d035553d99cd2b9ce5a472240293d05838:<このファイルのパス>` で読む。
