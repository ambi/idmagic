---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 本番運用可能な EmailSender adapter (SMTP) を導入する

ADR-035 を起こし、SMTP-only / PLAIN over TLS のみ / fail-open /
retry なし / display name なしの方針を確定。
`notification.SMTPEmailSender` adapter を追加し、bootstrap の
hardcode を `EMAIL_SENDER` env による切替に置き換えた。
console は引き続き既定で残る。port シグネチャ・SCL は無変更。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
