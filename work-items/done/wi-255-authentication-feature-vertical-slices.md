---
status: completed
archived_from: 59eea587b61ad491894496c46947842ca7de5020
---

# authentication を password/webauthn/mfa/session/recovery の feature 垂直スライスへ再配置する

Authentication の6 feature全層分割に加え、残存portを再精査した。
`EmailChangeTokenStore` と全永続化実装を IdManagement `user` featureへ移設し、
`EmailSender` / `EmailMessage` を shared notification capabilityへ移設した。
未使用の `LoginContinuation` は削除し、Authentication直下にはfeature横断の
`AuthEventBucketStore`だけを残した。DIとArchitectureの依存グラフも新しい所有権へ同期した。

全文は `git show 59eea587b61ad491894496c46947842ca7de5020:<このファイルのパス>` で読む。
