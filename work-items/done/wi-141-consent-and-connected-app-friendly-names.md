---
status: completed
archived_from: c846d5e549cbbe72c662efb5054e4abdb53d670b
---

# 同意・連携済みアプリ画面で client_id を人間可読名で表示する

client_id → 表示名の解決ロジックを `internal/shared/adapters/http/support/client_display_name.go`
の `ClientDisplayNameResolver` に集約した。解決順は OAuth2Client.client_name → Application
カタログ名 (OIDC binding = client_id) → client_id フォールバック。composition root で
`ApplicationGate` と同じく一度だけ組み立て、OAuth2 admin consents / authorize 同意画面
(BrowserTransactionResponse) と Authentication account consents の各ハンドラへ注入する。
SCL は `AccountConsentResponse` に `client_name`、`AdminConsentResponse` に `client_name` と
`preferred_username`(optional) を追加し、`BrowserTransactionResponse.client_name` の解決順を
明記、derived artifacts を再生成した。UI は接続済みアプリ / 管理同意画面とも可読名を主表示・
UUID を補助表記 (tooltip / secondary) にし、取消確認メッセージも可読名へ。表示名が無い client は
client_id にフォールバックする。当初 scope 外だった authorize 同意画面 (`/consent`) の UUID 生表示も
同 resolver に寄せて解消した (レビュー指摘対応)。

全文は `git show c846d5e549cbbe72c662efb5054e4abdb53d670b:<このファイルのパス>` で読む。
