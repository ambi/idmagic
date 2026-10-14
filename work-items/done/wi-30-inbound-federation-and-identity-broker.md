---
status: completed
archived_from: 132c3d7ffe37f08a04e3338539d90442a5d526c7
---

# Inbound federation / Identity Broker を導入する

- Authentication-owned inbound OIDC/SAML broker、tenant-scoped durable connection/link/attempt/replay storage、credential-less JIT provisioning、safe linking/unlinking、OAuth transaction resume を実装した。
- Login、admin settings、account security の全表示を日本語・英語辞書へ接続し、通常の UI test は英語既定、明示的な `locale: 'ja'` のみ日本語検証とする開発ルールを RA レベルの `DEVELOPMENT.md` に追加した。
- OIDC/SAML の未信頼入力検証、SSRF 防止、secret reference、step-up、account lockout 防止、replay 防止を adapter/use case/HTTP contract と fuzz test で固定した。

全文は `git show 132c3d7ffe37f08a04e3338539d90442a5d526c7:<このファイルのパス>` で読む。
